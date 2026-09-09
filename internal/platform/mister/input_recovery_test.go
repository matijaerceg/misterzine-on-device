//go:build linux

package mister

import (
	"encoding/binary"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"os"
	"testing"
	"time"
)

func TestInputDisconnectAndOverflowRelease(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "disconnect", true: "overflow"}[overflow], func(t *testing.T) {
			r, w, e := os.Pipe()
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			in := &Input{ch: make(chan platform.Event, 1), stop: make(chan struct{}), devs: map[string]*device{}}
			d := &device{f: r, name: "fixture", held: map[uint16]platform.Key{}}
			in.wg.Add(1)
			go in.read(d)
			send := func(typ, code uint16, val uint32) {
				b := make([]byte, 16)
				binary.LittleEndian.PutUint16(b[8:], typ)
				binary.LittleEndian.PutUint16(b[10:], code)
				binary.LittleEndian.PutUint32(b[12:], val)
				if _, e := w.Write(b); e != nil {
					t.Fatal(e)
				}
			}
			send(evKey, keyDown, 1)
			select {
			case ev := <-in.ch:
				if !ev.Pressed {
					t.Fatal(ev)
				}
			case <-time.After(time.Second):
				t.Fatal("no press")
			}
			if overflow {
				send(0, 3, 0)
				send(evKey, keyUp, 1)
				send(0, 0, 0)
			} else {
				w.Close()
			}
			select {
			case ev := <-in.ch:
				if ev.Pressed || ev.Key != platform.KeyDown {
					t.Fatal(ev)
				}
			case <-time.After(time.Second):
				t.Fatal("held key not released")
			}
			w.Close()
			in.wg.Wait()
		})
	}
}
func TestInputFullQueueDoesNotLoseRelease(t *testing.T) {
	in := &Input{ch: make(chan platform.Event, 1), stop: make(chan struct{})}
	in.ch <- platform.Event{Key: platform.KeyDown, Pressed: true}
	done := make(chan bool, 1)
	go func() { done <- in.deliver(platform.Event{Key: platform.KeyDown}) }()
	select {
	case <-done:
		t.Fatal("full queue dropped release")
	case <-time.After(20 * time.Millisecond):
	}
	<-in.ch
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("not delivered")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked")
	}
	if (<-in.ch).Pressed {
		t.Fatal("expected release")
	}
	in.ch <- platform.Event{}
	go func() { done <- in.deliver(platform.Event{}) }()
	close(in.stop)
	select {
	case ok := <-done:
		if ok {
			t.Fatal("delivered after stop")
		}
	case <-time.After(time.Second):
		t.Fatal("stop blocked")
	}
}
