//go:build linux

package mister

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestKeyboardTextKeepsPadNavigation(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{{"USB keyboard", "aaa! "}, {"MiSTer virtual input", ""}} {
		t.Run(tc.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			in := &Input{ch: make(chan platform.Event, 64), stop: make(chan struct{}), devs: map[string]*device{}}
			d := &device{f: r, name: tc.name, held: map[uint16]platform.Key{}}
			in.wg.Add(1)
			go in.read(d)
			// a + autorepeat, Shift+a, Shift+1, Space, Ctrl+c, Down repeat.
			for _, step := range [][2]uint16{{30, 1}, {30, 2}, {30, 0}, {42, 1}, {30, 1}, {30, 0}, {2, 1}, {2, 0}, {42, 0}, {57, 1}, {57, 0}, {29, 1}, {46, 1}, {46, 0}, {29, 0}, {108, 1}, {108, 2}, {108, 0}} {
				var b [16]byte
				binary.LittleEndian.PutUint16(b[8:], evKey)
				binary.LittleEndian.PutUint16(b[10:], step[0])
				binary.LittleEndian.PutUint32(b[12:], uint32(step[1]))
				if _, err := w.Write(b[:]); err != nil {
					t.Fatal(err)
				}
			}
			w.Close()
			in.wg.Wait()
			close(in.ch)
			text := ""
			spaces, down := 0, 0
			for ev := range in.ch {
				if ev.Pressed {
					if ev.Text != 0 {
						text += string(ev.Text)
					}
					if ev.Key == platform.KeySpace {
						spaces++
					}
					if ev.Key == platform.KeyDown {
						down++
					}
				}
			}
			// Matching is case-insensitive, so Shift+A deliberately stays lowercase.
			if text != tc.want || spaces != 1 || down != 1 {
				t.Fatalf("text=%q space=%d down=%d", text, spaces, down)
			}
		})
	}
}
