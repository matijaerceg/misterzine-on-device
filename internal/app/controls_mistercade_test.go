package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func mistercadeMappingApp() (*App, *time.Time, *[]support.Pad, *int) {
	a, clock, pads, saves := mappingApp()
	p := &(*pads)[0]
	p.Vendor, p.Product, p.Name = 0x16d0, 0x1358, "MiSTercade V2"
	p.Keys = []uint16{304, 305, 704, 705, 706, 707}
	p.Axes = []uint16{0, 1, 2, 5, 16, 17}
	a.controlInput.current = padID(*p)
	return a, clock, pads, saves
}

// MiSTercade V2's Generic HID report contains both button bits 17..20 and
// a hat for the same stick. Linux exposes those as 704..707 and 800..803.
// Sources: misteraddons/GP2040-CE, src/drivers/hid/HIDDriver.cpp and
// headers/drivers/hid/HIDDescriptors.h; Linux drivers/hid/hid-input.c.
func TestMiSTercadeBasicSetupDuplicateDirections(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		a, clock, pads, saves := mistercadeMappingApp()
		p := &(*pads)[0]
		a.OpenControls()
		for step, pair := range [][2]uint16{{707, 801}, {706, 800}, {705, 803}, {704, 802}} {
			if reverse {
				pair[0], pair[1] = pair[1], pair[0]
			}
			*clock = clock.Add(30 * time.Millisecond)
			checkPress(a, clock, pair[0], true)
			checkPress(a, clock, pair[1], true)
			*clock = clock.Add(30 * time.Millisecond)
			checkPress(a, clock, pair[0], false)
			if a.mapping.check.step != step {
				t.Fatal("advanced before both reports released")
			}
			checkPress(a, clock, pair[1], false)
			if a.mapping.check.step != step+1 {
				t.Fatalf("reverse=%v: direction %d left setup at step %d", reverse, step, a.mapping.check.step)
			}
		}
		for _, code := range []uint16{304, 305} {
			mappingTap(a, clock, rawControl(*p, code, platform.KeyOther))
		}
		if *saves != 1 || a.mapping.mode != "table" {
			t.Fatalf("setup failed: saves=%d mode=%s", *saves, a.mapping.mode)
		}
		for code, want := range map[uint16]string{707: "right", 706: "left", 705: "down", 704: "up"} {
			if got := a.mapping.draft.Inputs[code]; got != want {
				t.Fatalf("code %d: got %q, want %q", code, got, want)
			}
		}
		// Holding one physical direction edits it, rather than exiting as
		// though the two reports were two separate controls.
		checkHold(a, clock, 707, 801)
		if a.mapping.mode != "button" || a.mapping.check.editing != 707 {
			t.Fatalf("direction hold: screen=%v mode=%s input=%d", a.screen, a.mapping.mode, a.mapping.check.editing)
		}
	}
}

func TestMiSTercadeInvitationAndRealChords(t *testing.T) {
	a, clock, pads, saves := mistercadeMappingApp()
	a.screen = ScreenList
	mappingTap(a, clock, rawControl((*pads)[0], 304, platform.KeyEnter))
	checkHold(a, clock, 707, 801)
	if a.screen != ScreenControls || a.mapping.mode != "basic" || a.mapping.check.step != 0 {
		t.Fatal("one direction dismissed invitation or assigned the starting hold")
	}
	// A real second input is still a chord, even when the direction has
	// two reports. Releasing it cannot turn the remaining hold into a tap.
	for _, code := range []uint16{707, 801, 304} {
		checkPress(a, clock, code, true)
	}
	for _, code := range []uint16{304, 707, 801} {
		checkPress(a, clock, code, false)
	}
	if a.mapping.check.step != 0 || *saves != 0 {
		t.Fatal("real chord accepted as one direction")
	}
	// Cancelling with a real button still works.
	checkHold(a, clock, 304)
	if a.screen == ScreenControls {
		t.Fatal("button hold did not cancel")
	}
	mappingTap(a, clock, rawControl((*pads)[0], 304, platform.KeyEnter))
	checkHold(a, clock, 304, 305)
	if a.screen == ScreenControls {
		t.Fatal("two buttons did not dismiss invitation")
	}
}

func TestMiSTercadeKeepsHatProfilesAndRawTable(t *testing.T) {
	a, clock, _, _ := mistercadeMappingApp()
	a.OpenControls()
	// A profile made on firmware that only reported the hat stays editable.
	q := a.mapping.draft.Clone()
	q.BasicsComplete = true
	if !a.commitControl(q) {
		t.Fatal("could not save existing hat profile")
	}
	checkPress(a, clock, 707, true)
	checkPress(a, clock, 801, true)
	if !a.mapping.check.held[707] || !a.mapping.check.held[801] || !a.mapping.check.tested[707] || !a.mapping.check.tested[801] {
		t.Fatal("raw observations were discarded")
	}
	*clock = clock.Add(1100 * time.Millisecond)
	a.Tick(*clock)
	if a.mapping.mode != "button" || a.mapping.check.editing != 801 {
		t.Fatal("existing hat assignment was not the edit target")
	}
}

func TestMiSTercadeAliasRequiresMatchingHardwareAndCapabilities(t *testing.T) {
	for _, kind := range []string{"other-model", "other-vendor", "keyboard", "no-buttons", "no-hat"} {
		t.Run(kind, func(t *testing.T) {
			a, clock, pads, _ := mistercadeMappingApp()
			p := &(*pads)[0]
			switch kind {
			case "other-model":
				p.Product++
			case "other-vendor":
				p.Vendor++
			case "keyboard":
				p.Keyboard = true
			case "no-buttons":
				p.Keys = nil
			case "no-hat":
				p.Axes = nil
			}
			a.controlInput.current = padID(*p)
			a.OpenControls()
			for _, code := range []uint16{707, 801} {
				checkPress(a, clock, code, true)
			}
			for _, code := range []uint16{707, 801} {
				checkPress(a, clock, code, false)
			}
			if a.mapping.check.step != 0 {
				t.Fatal("unrelated reports merged")
			}
		})
	}
}

func TestMiSTercadeOtherPlayerDirectionDoesNotCancel(t *testing.T) {
	a, clock, pads, _ := mistercadeMappingApp()
	other := (*pads)[0]
	other.Node = "event8"
	other.InstanceID = "16d0_1358_1234567890abcdef"
	*pads = append(*pads, other)
	a.OpenControls()
	for _, code := range []uint16{707, 801} {
		ev := rawControl(other, code, platform.KeyOther)
		ev.At, ev.Pressed = *clock, true
		a.Handle(ev)
	}
	*clock = clock.Add(1100 * time.Millisecond)
	a.Tick(*clock)
	if a.screen != ScreenControls || a.mapping.mode != "basic" || a.mapping.check.step != 0 {
		t.Fatal("other player's direction cancelled or mapped")
	}
}

func TestMiSTercadeCancelledReportCannotAssign(t *testing.T) {
	a, clock, _, saves := mistercadeMappingApp()
	a.OpenControls()
	checkPress(a, clock, 707, true)
	checkPress(a, clock, 801, true)
	ev := rawControl(a.mapping.device, 707, platform.KeyOther)
	ev.At, ev.Cancelled = *clock, true
	a.Handle(ev)
	checkPress(a, clock, 801, false)
	if a.mapping.check.step != 0 || *saves != 0 {
		t.Fatal("lost input counted as a valid tap")
	}
}
