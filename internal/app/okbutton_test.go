package app

import (
	"strings"
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

func okRow(a *App) panelEntry {
	for _, e := range a.optionsEntries() {
		if e.kind == "ok-button" {
			return e
		}
	}
	panic("no OK button row")
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
	if row := okRow(a); row.disabled || row.vals[row.idx] != "Auto from MiSTer (B)" {
		t.Fatalf("row before any press: %+v", row)
	}
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
	if row := okRow(a); row.vals[0] != "Auto from MiSTer (A)" || row.vals[1] != "B" || row.vals[2] != "A" {
		t.Fatalf("xbox row values: %v", row.vals)
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
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "ok-button" {
			a.panel.cursor = i
		}
	}
	a.stepValue(1)
	if got := a.OKButtons(); len(got) != 1 || got["045e_028e"] != "a" {
		t.Fatalf("override saved as %v", got)
	}
	if row := okRow(a); row.idx != 1 || row.vals[1] != "A" || !strings.HasPrefix(row.help, "Microsoft X-B"+gfx.Ellipsis+": ") {
		t.Fatalf("row with override: %+v", row)
	}
	if a.legendSwapped() || a.padEvent(platform.Event{Key: platform.KeyEnter, Pressed: true, At: at, Source: pad}).Key != platform.KeyEnter {
		t.Fatal("override A did not stop the swap")
	}
	if lines := a.padLines(okTestPads()[1]); lines[len(lines)-1] != "  OK button: A (set in Options)" {
		t.Fatalf("tester pad lines with override %q", lines)
	}
	for i, e := range a.panel.entries {
		if e.kind == "ok-button" {
			a.panel.cursor = i
		}
	}
	a.stepValue(-1)
	if a.OKButtons() != nil || !a.legendSwapped() {
		t.Fatalf("auto again: %v swapped %v", a.OKButtons(), a.legendSwapped())
	}
	// the arcade board presses: it is the current pad now, and it has
	// nothing to choose, so the row mutes and the legends go straight
	a.padEvent(platform.Event{Key: platform.KeyStart, Code: 315, Pressed: true, At: at, Source: "USB Arcade Controller"})
	if row := okRow(a); !row.disabled || row.vals[0] != "via MiSTer" {
		t.Fatalf("row for the arcade board: %+v", row)
	}
	if a.legendSwapped() || a.btn("A") != "A" {
		t.Fatal("legends should follow the arcade board")
	}
}

// With no pad at all the row waits, muted; a pad defined with MENU OK
// elsewhere, or not at all, confirms with A and the hint says why; one
// button in both A and B is called out as the define mistake it is.
func TestOKButtonRowStates(t *testing.T) {
	a := okTestApp(nil)
	if row := okRow(a); !row.disabled || row.vals[0] != "no pad used yet" {
		t.Fatalf("row without pads: %+v", row)
	}
	if a.padEvent(platform.Event{Key: platform.KeyEnter, Pressed: true, Source: "some pad"}).Key != platform.KeyEnter {
		t.Fatal("an unknown source should pass through")
	}
	pad := okTestPads()[1]
	// one pad as two event nodes with one name (the DE10's Xbox 360 pad)
	// is still the only pad; two different pads are not
	twin := pad
	twin.Node = "event8"
	a = okTestApp(func() []support.Pad { return []support.Pad{pad, twin} })
	if row := okRow(a); row.disabled || row.vals[row.idx] != "Auto from MiSTer (B)" {
		t.Fatalf("row with the pad's two nodes: %+v", row)
	}
	other := pad
	other.Name, other.Product = "8BitDo M30", 0x0b12
	a = okTestApp(func() []support.Pad { return []support.Pad{pad, other} })
	if row := okRow(a); !row.disabled || row.vals[0] != "no pad used yet" {
		t.Fatalf("row with two different pads: %+v", row)
	}
	for _, c := range []struct {
		ok, val, note string
	}{
		{"A", "Auto from MiSTer (A)", "MiSTer's MENU OK"},
		{"", "Auto from MiSTer (A)", "MiSTer's MENU OK is not defined"},
		{"X", "Auto from MiSTer (A)", "MiSTer's MENU OK is on X"},
		{"btn 316", "Auto from MiSTer (A)", "MiSTer's MENU OK is on btn 316"},
	} {
		p := pad
		p.OK = c.ok
		a := okTestApp(func() []support.Pad { return []support.Pad{p} })
		row := okRow(a)
		if row.disabled || row.vals[row.idx] != c.val || a.okNote(p) != c.note || a.legendSwapped() {
			t.Errorf("OK %q: row %+v note %q", c.ok, row, a.okNote(p))
		}
		checkOKHint(t, a, row)
	}
	same := pad
	same.Slots = map[string]uint16{"A": 305, "B": 305, "Start": 315}
	a = okTestApp(func() []support.Pad { return []support.Pad{same} })
	if row := okRow(a); !row.disabled || row.vals[0] != "A = B" {
		t.Fatalf("row for A = B: %+v", row)
	}
	if lines := a.padLines(same); lines[len(lines)-1] != "  A and B are the same button in MiSTer: no back button here" {
		t.Fatalf("tester lines for A = B: %q", lines)
	}
	checkOKHint(t, a, okRow(a))
	arcade := okTestPads()[0]
	a = okTestApp(func() []support.Pad { return []support.Pad{arcade} })
	a.padEvent(platform.Event{Key: platform.KeyStart, Pressed: true, Source: arcade.Name})
	checkOKHint(t, a, okRow(a))
}

// checkOKHint holds the row's hint to the help box, as every Options hint
// is held (options_help_test.go): three lines horizontal, four in tate.
func checkOKHint(t *testing.T, a *App, row panelEntry) {
	t.Helper()
	for _, c := range []struct {
		rot   gfx.Rotation
		lines int
	}{{gfx.RotNone, 3}, {gfx.RotRight, 4}} {
		b := New(Config{PhysW: 320, PhysH: 240, Rotation: c.rot, SafeInsetX: 15, SafeInsetY: 15}, a.Data(), nil)
		cols := b.sm.Cols(b.lay.Body.Dx() - 6)
		if n := len(gfx.Wrap(row.help, cols, 99)); n > c.lines {
			t.Errorf("%d cols x %d lines: hint needs %d lines: %q", cols, c.lines, n, row.help)
		}
	}
}
