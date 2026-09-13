package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func padTestApp() (*App, *time.Time) {
	a, clock := supportApp()
	a.cfg.Support = &SupportHooks{Pads: func() []support.Pad {
		return []support.Pad{
			{Node: "event3", Name: "Microsoft X-Box 360 pad", Vendor: 0x045e, Product: 0x028e, Mapped: true, Direct: true, Menu: "316", Map: "/media/fat/config/inputs/input_045e_028e_v3.map",
				Slots: map[string]uint16{"A": 305, "B": 304, "X": 308, "Y": 307, "L": 310, "R": 773, "Start": 315, "Up": 802}},
			{Node: "event0", Name: "Brook ZERO-Pi Fighting Board", Map: "Linux default", Slots: map[string]uint16{"Start": 315}},
		}
	}}
	a.OpenTroubleshooting()
	a.actSupport(platform.KeyDown)
	a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyEnter, At: *clock})
	return a, clock
}

func TestPadTesterListsPressesAndLeavesOnHeldB(t *testing.T) {
	a, clock := padTestApp()
	if a.support.mode != "pad" || len(a.support.pads) != 2 {
		t.Fatalf("mode %q pads %d", a.support.mode, len(a.support.pads))
	}
	launches := 0
	a.cfg.Launch = func(string) { launches++ }
	at := *clock
	press := func(k platform.Key, code uint16, source string, gap time.Duration) {
		at = at.Add(gap)
		a.Handle(platform.Event{Key: k, Code: code, Source: source, Pressed: true, At: at})
		a.Handle(platform.Event{Key: k, Code: code, Source: source, At: at.Add(10 * time.Millisecond)})
	}
	// the 8BitDo defined by position: 305 is the button called A
	press(platform.KeyEnter, 305, "Microsoft X-Box 360 pad", time.Second)
	press(platform.KeyStart, 315, "Brook ZERO-Pi Fighting Board", 8*time.Millisecond)
	press(platform.KeyOther, 310, "Microsoft X-Box 360 pad", 500*time.Millisecond)
	press(platform.KeyUp, keyUpCode, "MiSTer virtual input", 100*time.Millisecond)
	press(platform.KeyEnter, 0, "script", 100*time.Millisecond)
	if launches != 0 || a.Screen() != ScreenTroubleshooting || a.support.mode != "pad" {
		t.Fatal("tester let a press act")
	}
	if len(a.support.presses) != 5 {
		t.Fatalf("%d presses", len(a.support.presses))
	}
	lines := make([]string, len(a.support.presses))
	for i, p := range a.support.presses {
		var prev *padPress
		if i+1 < len(a.support.presses) {
			prev = &a.support.presses[i+1]
		}
		lines[i] = a.padPressLine(p, prev)
	}
	press(platform.KeyPageDown, 773, "Microsoft X-Box 360 pad", 90*time.Millisecond)
	if l := a.padPressLine(a.support.presses[0], nil); !strings.Contains(l, "ax2+ = R: page down (R)") {
		t.Fatalf("axis press line %q", l)
	}
	a.support.presses = a.support.presses[1:]
	for i, want := range []string{"script: details / confirm +100ms", "MiSTer translation: up +100ms", "btn 310 = L: no action +500ms", "btn 315 = Start: launch +8ms", "btn 305 = A: details / confirm"} {
		if !strings.Contains(lines[i], want) {
			t.Fatalf("line %d %q lacks %q", i, lines[i], want)
		}
	}
	// the label set applies to slot names
	a.cfg.ButtonLabels = "playstation"
	if l := a.padPressLine(a.support.presses[4], nil); !strings.Contains(l, "= "+gfx.Circle+":") {
		t.Fatalf("label set ignored: %q", l)
	}
	if l := a.padLines(a.support.pads[0]); len(l) < 3 || !strings.Contains(l[1], gfx.Circle+" 305") || !strings.Contains(l[2], "input_045e_028e_v3.map") {
		t.Fatalf("pad lines %q", l)
	}
	if l := a.padLines(a.support.pads[1]); !strings.Contains(strings.Join(l, " "), "no MiSTer map") {
		t.Fatalf("unmapped pad lines %q", l)
	}
	a.cfg.ButtonLabels = "mister"
	for i := 0; i < padPressKeep+3; i++ {
		press(platform.KeyTab, 308, "Microsoft X-Box 360 pad", 200*time.Millisecond)
	}
	if len(a.support.presses) != padPressKeep {
		t.Fatalf("kept %d presses", len(a.support.presses))
	}
	// a tap of B is listed and does not leave; a two second hold does
	at = at.Add(time.Second)
	a.Handle(platform.Event{Key: platform.KeyBack, Code: 304, Source: "Microsoft X-Box 360 pad", Pressed: true, At: at})
	if a.NextTick().IsZero() {
		t.Fatal("no leave deadline scheduled")
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Code: 304, Source: "Microsoft X-Box 360 pad", At: at.Add(300 * time.Millisecond)})
	a.Tick(at.Add(3 * time.Second))
	if a.support.mode != "pad" || !strings.Contains(a.padPressLine(a.support.presses[0], nil), "btn 304 = B: back / Options") {
		t.Fatalf("tap of B: mode %q first %q", a.support.mode, a.padPressLine(a.support.presses[0], nil))
	}
	at = at.Add(4 * time.Second)
	a.Handle(platform.Event{Key: platform.KeyBack, Code: 304, Source: "Microsoft X-Box 360 pad", Pressed: true, At: at})
	// the hold fills the line under the top bar a pixel at a time
	w := a.lay.Status.Dx()
	if next := a.NextTick(); !next.After(at) || !next.Before(at.Add(100*time.Millisecond)) {
		t.Fatalf("NextTick %v, want the first pixel shortly after %v", next, at)
	}
	if !a.Tick(at.Add(time.Second)) || a.support.mode != "pad" {
		t.Fatal("left too early, or no repaint for the line")
	}
	if bar := a.support.holdBar; bar < w/2-1 || bar > w/2+1 || a.holdBar() != bar {
		t.Fatalf("one second in, the line is %d px of %d (painted %d)", bar, w, a.holdBar())
	}
	a.Tick(at.Add(padTestLeave))
	if a.support.mode != "menu" || a.Screen() != ScreenTroubleshooting || a.support.holdBar != 0 {
		t.Fatalf("hold did not leave cleanly: %q, line %d px", a.support.mode, a.support.holdBar)
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Code: 304, Source: "Microsoft X-Box 360 pad", At: at.Add(padTestLeave + 50*time.Millisecond)})
	if a.Screen() != ScreenTroubleshooting || a.support.mode != "menu" {
		t.Fatal("the release after leaving acted")
	}
	a.Invalidate()
	a.Paint()
}

const keyUpCode = 103
