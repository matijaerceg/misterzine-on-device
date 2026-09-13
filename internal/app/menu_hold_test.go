package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// menuApp is a list of three games with a Quit counter and a tap helper.
func menuApp() (*App, *time.Time, *int, func(platform.Key) bool) {
	a, clock := saverApp()
	quits := 0
	a.cfg.Quit = func() { quits++ }
	a.cfg.Status = func(int) data.Status { return data.StatusCurrent }
	a.cfg.Exists = func(string) bool { return true }
	a.SetData(data.Ingest([]data.Row{
		{K: "a", Title: "Alpha", Base: "Arcade", MRA: "_Arcade/Alpha.mra", Updated: "2026-01-01"},
		{K: "b", Title: "Beta", Base: "Console", Updated: "2026-01-02"},
		{K: "c", Title: "Gamma", Base: "Arcade", MRA: "_Arcade/Gamma.mra", Updated: "2026-01-03"},
	}, "test", *clock), nil)
	tap := func(k platform.Key) bool {
		*clock = clock.Add(30 * time.Millisecond)
		r := a.Handle(platform.Event{Key: k, Pressed: true, At: *clock})
		*clock = clock.Add(30 * time.Millisecond)
		a.Handle(platform.Event{Key: k, Pressed: false, At: *clock})
		return r
	}
	return a, clock, &quits, tap
}

func TestMenuHeldLeavesInOptionsMode(t *testing.T) {
	a, clock, quits, _ := menuApp()
	t0 := *clock
	if !a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: true, At: t0}) || a.screen != ScreenOptions {
		t.Fatalf("the press should open Options at once: screen %v", a.screen)
	}
	if next := a.NextTick(); !next.Equal(t0.Add(menuHint)) {
		t.Fatalf("NextTick %v, want the hint at %v", next, t0.Add(menuHint))
	}
	if a.Tick(t0.Add(200 * time.Millisecond)); a.notice != "" {
		t.Fatalf("a tap must not hint: %q", a.notice)
	}
	if !a.Tick(t0.Add(350*time.Millisecond)) || a.notice != menuHoldNotice {
		t.Fatalf("no hint after %v: %q", 350*time.Millisecond, a.notice)
	}
	if next := a.NextTick(); !next.Equal(t0.Add(menuHold)) {
		t.Fatalf("NextTick %v, want the leave at %v", next, t0.Add(menuHold))
	}
	a.Tick(t0.Add(1900 * time.Millisecond))
	if *quits != 0 {
		t.Fatal("left before the hold length")
	}
	if !a.Tick(t0.Add(menuHold)) || *quits != 1 || a.notice != "" {
		t.Fatalf("hold: quits %d, notice %q", *quits, a.notice)
	}
	a.Tick(t0.Add(5 * time.Second))
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: false, At: t0.Add(5 * time.Second)})
	a.Tick(t0.Add(9 * time.Second))
	if *quits != 1 || !a.NextTick().IsZero() && a.NextTick().Before(t0.Add(time.Minute)) {
		t.Fatalf("one hold leaves once: quits %d, NextTick %v", *quits, a.NextTick())
	}
}

func TestMenuReleasedBeforeHoldStays(t *testing.T) {
	a, clock, quits, _ := menuApp()
	t0 := *clock
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: true, At: t0})
	a.Tick(t0.Add(700 * time.Millisecond))
	if a.notice != menuHoldNotice {
		t.Fatalf("notice %q", a.notice)
	}
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: false, At: t0.Add(time.Second)})
	if a.notice != "" {
		t.Fatalf("the release should take the hint away: %q", a.notice)
	}
	a.Tick(t0.Add(3 * time.Second))
	if *quits != 0 || a.screen != ScreenOptions {
		t.Fatalf("quits %d, screen %v", *quits, a.screen)
	}
	// the vsync loop's Frame counts too
	t1 := t0.Add(10 * time.Second)
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: true, At: t1})
	a.Frame(t1.Add(menuHold))
	if *quits != 1 {
		t.Fatalf("Frame at the hold length: quits %d", *quits)
	}
}

func TestMenuHoldIgnoredInLeaveModeAndPadTester(t *testing.T) {
	a, clock, quits, _ := menuApp()
	a.cfg.MenuButton = "leave"
	t0 := *clock
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: true, At: t0})
	if *quits != 1 || !a.menuAt.IsZero() {
		t.Fatalf("leave: quits %d, timing %v", *quits, a.menuAt)
	}
	a.Tick(t0.Add(3 * time.Second))
	if *quits != 1 {
		t.Fatalf("leave must not quit twice: %d", *quits)
	}
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: false, At: t0.Add(3 * time.Second)})

	a.cfg.MenuButton = "options"
	a.screen, a.support.mode = ScreenTroubleshooting, "pad"
	t1 := t0.Add(10 * time.Second)
	a.Handle(platform.Event{Key: platform.KeyMenu, Pressed: true, At: t1})
	a.Tick(t1.Add(3 * time.Second))
	if *quits != 1 || !a.menuAt.IsZero() {
		t.Fatalf("the pad tester only logs Menu: quits %d", *quits)
	}
}

func TestOptionsReturnsToTheScreenItWasOpenedOver(t *testing.T) {
	a, _, quits, tap := menuApp()
	// Details, by Menu and by B
	tap(platform.KeyEnter)
	if a.screen != ScreenDetails {
		t.Fatalf("screen %v", a.screen)
	}
	tap(platform.KeyMenu)
	if a.screen != ScreenOptions {
		t.Fatalf("screen %v", a.screen)
	}
	tap(platform.KeyMenu)
	if a.screen != ScreenDetails {
		t.Fatalf("Menu should return to Details: %v", a.screen)
	}
	tap(platform.KeyMenu)
	tap(platform.KeyBack)
	if a.screen != ScreenDetails {
		t.Fatalf("B should return to Details: %v", a.screen)
	}
	// a trip through a screen Options owns keeps the origin
	tap(platform.KeyMenu)
	a.OpenScan()
	tap(platform.KeyBack) // scan -> Options
	if a.screen != ScreenOptions {
		t.Fatalf("screen %v", a.screen)
	}
	tap(platform.KeyBack)
	if a.screen != ScreenDetails {
		t.Fatalf("Options after a scan should still return to Details: %v", a.screen)
	}
	// the artwork
	tap(platform.KeyEnter)
	if a.screen != ScreenShot {
		t.Fatalf("screen %v", a.screen)
	}
	tap(platform.KeyMenu)
	tap(platform.KeyBack)
	if a.screen != ScreenShot {
		t.Fatalf("B should return to the artwork: %v", a.screen)
	}
	// a row that is no longer under the cursor lands on the list
	tap(platform.KeyMenu)
	a.cursor = 1
	tap(platform.KeyBack)
	if a.screen != ScreenList {
		t.Fatalf("another row under the cursor: %v", a.screen)
	}
	// Filters keep their browsing state across Options
	tap(platform.KeyTab)
	if a.screen != ScreenFilter {
		t.Fatalf("screen %v", a.screen)
	}
	tap(platform.KeyDown)
	tap(platform.KeyDown)
	row := a.panel.cursor
	if row == 0 {
		t.Fatal("the cursor should have moved down the filters")
	}
	tap(platform.KeyMenu)
	if a.screen != ScreenOptions || a.panel.cursor == row && len(a.panel.entries) == 0 {
		t.Fatalf("screen %v", a.screen)
	}
	tap(platform.KeyDown)
	optionsRow := a.panel.cursor
	tap(platform.KeyMenu)
	if a.screen != ScreenFilter || a.panel.cursor != row {
		t.Fatalf("Filters should come back on row %d: screen %v cursor %d", row, a.screen, a.panel.cursor)
	}
	tap(platform.KeyBack)
	if a.screen != ScreenList {
		t.Fatalf("B on Filters: %v", a.screen)
	}
	tap(platform.KeyBack)
	if a.screen != ScreenOptions || a.panel.cursor != optionsRow {
		t.Fatalf("Options should reopen on row %d: screen %v cursor %d", optionsRow, a.screen, a.panel.cursor)
	}
	tap(platform.KeyBack)
	if a.screen != ScreenList || *quits != 0 {
		t.Fatalf("screen %v quits %d", a.screen, *quits)
	}
}
