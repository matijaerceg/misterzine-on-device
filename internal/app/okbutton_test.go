package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// The user's two pads: the 8BitDo in Xbox mode, defined by position with
// MENU OK on the bottom button (slot B), and an arcade board only readable
// for Start.
func okTestPads() []support.Pad {
	return []support.Pad{
		{Node: "event0", Name: "USB Arcade Controller", Vendor: 0x1234, Product: 0x5678, Map: "Linux default", Slots: map[string]uint16{"Start": 315}},
		{Node: "event3", Name: "Microsoft X-Box 360 pad", Vendor: 0x045e, Product: 0x028e, Mapped: true, Direct: true, OK: "B", Back: "A", Map: "/media/fat/config/inputs/input_045e_028e_v3.map",
			Slots: map[string]uint16{"Up": 802, "Down": 803, "Left": 800, "Right": 801, "A": 305, "B": 304, "X": 308, "Y": 307, "L": 310, "R": 773, "Select": 314, "Start": 315}},
	}
}

func okTestApp(pads func() []support.Pad) *App {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha", Updated: "2026-09-07"}, {Base: "Arcade", K: "b", Title: "Beta", Updated: "2026-09-07"}}
	cfg := Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }, Launcher: func() bool { return true }}
	if pads != nil {
		cfg.Support = &SupportHooks{Pads: pads}
	}
	return New(cfg, data.Ingest(rows, "", now), nil)
}

// A pad whose MENU OK is B confirms with B here: its Enter and back trade
// places, the legends name the buttons the other way round, and the pad
// tester keeps calling the slot by its own name. Other sources are left
// alone, and an override in Options puts A back for that pad only.
func TestOKButtonFollowsMiSTer(t *testing.T) {
	a := okTestApp(okTestPads)
	pad := "Microsoft X-Box 360 pad"
	at := time.Now()
	// nothing pressed yet, but the one defined pad is the current one
	if !a.legendSwapped() || a.btn("A") != "B" || a.btn("Hold B") != "Hold A" || a.btn("X") != "X" || a.label("A") != "A" {
		t.Fatalf("legends: A=%q Hold B=%q", a.btn("A"), a.btn("Hold B"))
	}
	for key, want := range map[platform.Key]platform.Key{platform.KeyEnter: platform.KeyBack, platform.KeyBack: platform.KeyEnter, platform.KeyTab: platform.KeyTab, platform.KeyStart: platform.KeyStart} {
		if got := a.padEvent(platform.Event{Key: key, Code: 304, Pressed: true, At: at, Source: pad}).Key; got != want {
			t.Errorf("%v from the pad became %v, want %v", key, got, want)
		}
	}
	for _, src := range []string{"script", "MiSTer virtual input", "AT Translated Set 2 keyboard", ""} {
		if got := a.padEvent(platform.Event{Key: platform.KeyEnter, Pressed: true, At: at, Source: src}).Key; got != platform.KeyEnter {
			t.Errorf("Enter from %q became %v", src, got)
		}
	}
	// the xbox label set prints the swapped names by position: the bottom
	// button is the pad's A, and it confirms
	a.cfg.ButtonLabels = "xbox"
	if a.btn("A") != "A" || a.btn("B") != "B" || a.label("A") != "B" {
		t.Fatalf("xbox labels: A=%q B=%q label(A)=%q", a.btn("A"), a.btn("B"), a.label("A"))
	}
	a.cfg.ButtonLabels = "mister"
	// the tester names the slot, and says where OK comes from
	a.support.pads = okTestPads()
	if got := a.padPressLine(padPress{at: at, source: pad, code: 304, key: platform.KeyEnter}, nil); got != "Microsoft X-B"+gfx.Ellipsis+" btn 304 = B: details / confirm" {
		t.Fatalf("tester line %q", got)
	}
	if lines := a.padLines(okTestPads()[1]); lines[len(lines)-1] != "  OK button: B (MiSTer's MENU OK)" {
		t.Fatalf("tester pad lines %q", lines)
	}
	// the override: A for this pad, saved by vendor_product, then back to auto
	a.cfg.OKButtons = map[string]string{"045e_028e": "a"}
	if got := a.OKButtons(); len(got) != 1 || got["045e_028e"] != "a" {
		t.Fatalf("override saved as %v", got)
	}
	if a.legendSwapped() || a.padEvent(platform.Event{Key: platform.KeyEnter, Pressed: true, At: at, Source: pad}).Key != platform.KeyEnter {
		t.Fatal("override A did not stop the swap")
	}
	if lines := a.padLines(okTestPads()[1]); lines[len(lines)-1] != "  OK button: A (saved choice)" {
		t.Fatalf("tester pad lines with override %q", lines)
	}
	a.cfg.OKButtons = nil
	if a.OKButtons() != nil || !a.legendSwapped() {
		t.Fatalf("auto again: %v swapped %v", a.OKButtons(), a.legendSwapped())
	}
	// the arcade board presses: it is the current pad now, and it has
	// nothing to choose, so the legends go straight
	a.padEvent(platform.Event{Key: platform.KeyStart, Code: 315, Pressed: true, At: at, Source: "USB Arcade Controller"})
	if a.legendSwapped() || a.btn("A") != "A" {
		t.Fatal("legends should follow the arcade board")
	}
}

// An unknown source passes through; a pad defined with MENU OK elsewhere,
// or not at all, confirms with A and the tester says why; one button in
// both A and B is called out as the define mistake it is.
func TestOKButtonNotes(t *testing.T) {
	a := okTestApp(nil)
	if a.padEvent(platform.Event{Key: platform.KeyEnter, Pressed: true, Source: "some pad"}).Key != platform.KeyEnter {
		t.Fatal("an unknown source should pass through")
	}
	pad := okTestPads()[1]
	for _, c := range []struct{ ok, note string }{
		{"A", "MiSTer's MENU OK"},
		{"", "MiSTer's MENU OK is not defined"},
		{"X", "MiSTer's MENU OK is on X"},
		{"btn 316", "MiSTer's MENU OK is on btn 316"},
	} {
		p := pad
		p.OK = c.ok
		a := okTestApp(func() []support.Pad { return []support.Pad{p} })
		if a.okNote(p) != c.note || a.legendSwapped() {
			t.Errorf("OK %q: note %q swapped %v", c.ok, a.okNote(p), a.legendSwapped())
		}
	}
	same := pad
	same.Slots = map[string]uint16{"A": 305, "B": 305, "Start": 315}
	a = okTestApp(func() []support.Pad { return []support.Pad{same} })
	if lines := a.padLines(same); lines[len(lines)-1] != "  A and B are the same button in MiSTer: no back button here" {
		t.Fatalf("tester lines for A = B: %q", lines)
	}
}
