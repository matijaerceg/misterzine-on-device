package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
	"testing"
)

func TestRawIdentitySeparatesSameNamedPads(t *testing.T) {
	a, _ := saverApp()
	pads := []support.Pad{{Node: "event1", Name: "USB Gamepad", Vendor: 1, Product: 2, Direct: true, OK: "A"}, {Node: "event2", Name: "USB Gamepad", Vendor: 3, Product: 4, Direct: true, OK: "B"}}
	a.cfg.Support = &SupportHooks{Pads: func() []support.Pad { return pads }}
	ev := a.padEvent(platform.Event{Source: "USB Gamepad", DeviceID: "0003_0004", Node: "event2", Key: platform.KeyBack, Pressed: true})
	if ev.Key != platform.KeyEnter {
		t.Fatal("used another model's OK preference", ev)
	}
	if p, ok := a.currentPad(); !ok || p.Vendor != 3 {
		t.Fatal("Options follows wrong controller", p)
	}
	a.setOKButton("a")
	if a.cfg.OKButtons["0003_0004"] != "a" || a.cfg.OKButtons["0001_0002"] != "" {
		t.Fatal("changed wrong profile")
	}
}
