package main

import (
	"fmt"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func controlFixtures() []support.Pad {
	return []support.Pad{
		{Node: "event3", Connection: "usb-controller-1.1", Name: "Xbox 360 controller", Vendor: 0x045e, Product: 0x028e, Direct: true, Mapped: true, OK: "B", Back: "A", Menu: "316", Slots: map[string]uint16{"A": 305, "B": 304, "X": 307, "Y": 308, "Start": 315, "Select": 314, "L": 310, "R": 311, "Up": 802, "Down": 803, "Left": 800, "Right": 801}},
		{Node: "event4", Connection: "usb-controller-1.2", Name: "J-PAC keyboard encoder", Vendor: 0xd209, Product: 0x0301, Keyboard: true},
	}
}

func controlFixtureEvent(p support.Pad, code uint16, pressed bool, at time.Time) platform.Event {
	key := platform.KeyOther
	keys := map[string]platform.Key{"A": platform.KeyEnter, "B": platform.KeyBack, "X": platform.KeyTab, "Y": platform.KeySpace, "Start": platform.KeyStart, "Select": platform.KeySelect, "L": platform.KeyPageUp, "R": platform.KeyPageDown, "Up": platform.KeyUp, "Down": platform.KeyDown, "Left": platform.KeyLeft, "Right": platform.KeyRight}
	if k, ok := keys[p.Slot(code)]; ok {
		key = k
	}
	if code == 316 && !p.Keyboard {
		key = platform.KeyMenu
	}
	if p.Keyboard {
		key = map[uint16]platform.Key{1: platform.KeyBack, 28: platform.KeyEnter, 103: platform.KeyUp, 108: platform.KeyDown, 105: platform.KeyLeft, 106: platform.KeyRight, 57: platform.KeySpace}[code]
		if key == platform.KeyNone {
			key = platform.KeyOther
		}
	}
	return platform.Event{Key: key, Code: code, Pressed: pressed, At: at, Source: p.Name, DeviceID: fmt.Sprintf("%04x_%04x", p.Vendor, p.Product), Node: p.Node, Direct: p.Direct, Keyboard: p.Keyboard}
}
