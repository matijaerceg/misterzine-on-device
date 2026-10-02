//go:build linux

package mister

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestRawControlIdentityAndPanelTranslationSuppression(t *testing.T) {
	in := &Input{ch: make(chan platform.Event, 16), stop: make(chan struct{}), devs: map[string]*device{}}
	panel := &device{path: "/dev/input/event4", name: "keyboard encoder", vendor: 0xd209, product: 0x0301, physical: true, held: map[uint16]platform.Key{}}
	in.devs[panel.path] = panel
	virtual := &device{name: "MiSTer virtual input", held: map[uint16]platform.Key{}}
	in.SetPanels(map[string]bool{"d209_0301": true})
	in.emit(virtual, keyEnter, true, 0, time.Now())
	if len(in.ch) != 0 {
		t.Fatal("translated panel event not suppressed")
	}
	in.emit(panel, 2, true, '1', time.Now())
	ev := <-in.ch
	if ev.DeviceID != "d209_0301" || ev.Node != "event4" || !ev.Keyboard || ev.Direct || ev.Text != '1' {
		t.Fatal(ev)
	}
	in.releaseHeld(panel)
	ev = <-in.ch
	if ev.DeviceID != "d209_0301" || ev.Pressed || !ev.Cancelled {
		t.Fatal(ev)
	}
	// Turning beta off or unplugging the panel restores translated input.
	in.SetPanels(nil)
	in.emit(virtual, keyEnter, true, 0, time.Now())
	if ev = <-in.ch; ev.DeviceID != "" || ev.Keyboard || ev.Key != platform.KeyEnter {
		t.Fatal(ev)
	}
	in.SetPanels(map[string]bool{"d209_0301": true})
	delete(in.devs, panel.path)
	if in.panelConnected() {
		t.Fatal("disconnected panel suppresses other pads")
	}
	pad := &device{path: "/dev/input/event3", name: "pad", vendor: 1, product: 2, pad: true, physical: true, mapping: padMapping{direct: true}, grabbed: true}
	if ev = pad.event(platform.KeyStart, 315, true, time.Now()); !ev.Direct || ev.Keyboard || ev.DeviceID != "0001_0002" {
		t.Fatal(ev)
	}
	pad.grabbed = false
	if pad.event(platform.KeyStart, 315, true, time.Now()).Direct {
		t.Fatal("unheld pad offered for remapping")
	}
	panel.physical = false
	if ev = panel.event(platform.KeyEnter, 28, true, time.Now()); ev.DeviceID != "" || ev.Keyboard {
		t.Fatal("virtual encoder acquired a profile", ev)
	}
}
