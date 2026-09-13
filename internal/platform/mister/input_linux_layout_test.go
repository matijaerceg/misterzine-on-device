//go:build linux

package mister

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// A pad with no MiSTer map that reports the Linux gamepad layout (the
// user's 8BitDo Ultimate 2C on the Pi) is read by position, held, with its
// home button as Menu, the bottom button as OK, and the hat and left stick
// moving; a pad without the face buttons gives up Start only, and a
// virtual device is never held.
func TestLinuxDefaultLayout(t *testing.T) {
	bits := buttonBits(304, 305, 307, 308, 310, 311, 314, 315, 316)
	abs := axisBits(0, 1, 2, 5, 16, 17)
	m := fallbackPad(bits, abs, false)
	if !m.direct || m.Mapped || m.Source != "Linux default layout" || m.Code != 315 || m.osd != [2]uint16{316, 316} {
		t.Fatalf("layout: %+v", m)
	}
	want := map[uint16]platform.Key{
		305: platform.KeyEnter, 304: platform.KeyBack, 307: platform.KeyTab, 308: platform.KeySpace,
		310: platform.KeyPageUp, 311: platform.KeyPageDown, 314: platform.KeySelect, 315: platform.KeyStart,
		801: platform.KeyRight, 800: platform.KeyLeft, 803: platform.KeyDown, 802: platform.KeyUp,
		AxisCode(0, true): platform.KeyRight, AxisCode(0, false): platform.KeyLeft, AxisCode(1, true): platform.KeyDown, AxisCode(1, false): platform.KeyUp,
	}
	if len(m.Keys) != len(want) {
		t.Fatalf("keys %v", m.Keys)
	}
	for code, key := range want {
		if m.Keys[code] != key {
			t.Errorf("code %d = %v, want %v", code, m.Keys[code], key)
		}
	}
	if !m.bothEdges(0) || !m.bothEdges(1) || m.bothEdges(2) {
		t.Fatal("the left stick should count both ways, the trigger one way")
	}
	p := m.info("event12", "8BitDo Ultimate 2C Wireless Controller", 0x2dc8, 0x310a)
	if p.Mapped || !p.Direct || p.OK != "B" || p.Back != "A" || p.Slot(304) != "B" || p.Slot(305) != "A" || p.Menu != "316" || p.MenuStick != "axes 0/1" {
		t.Fatalf("pad info %+v", p)
	}
	// only the buttons the node has: no hat, no shoulders, no home button
	m = fallbackPad(buttonBits(304, 305, 307, 308, 315), axisBits(0, 1), false)
	if len(m.Keys) != 9 || m.osd[0] != 0 || m.Slots["L"] != 0 || m.Slots["Up"] != 0 {
		t.Fatalf("sparse pad: %+v", m)
	}
	// d-pad buttons stand in for a missing hat
	m = fallbackPad(buttonBits(304, 305, 544, 545, 546, 547), [8]byte{}, false)
	if m.Keys[544] != platform.KeyUp || m.Keys[547] != platform.KeyRight || m.Code != 0 {
		t.Fatalf("d-pad buttons: %+v", m)
	}
	// no face buttons: Start alone, the rest through Main
	m = fallbackPad(buttonBits(315, 288, 289), [8]byte{}, false)
	if m.direct || m.Code != 315 || len(m.Keys) != 0 || m.Source != "Linux default" {
		t.Fatalf("joystick-class pad: %+v", m)
	}
	// a uinput pad (Zaparoo) is Main's whatever it reports
	m = fallbackPad(bits, abs, true)
	if m.direct || m.Code != 315 || len(m.Keys) != 0 {
		t.Fatalf("virtual pad: %+v", m)
	}
}
