package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestMenuButtonOpensOptionsOrLeaves(t *testing.T) {
	a, clock := saverApp()
	quits := 0
	a.cfg.Quit = func() { quits++ }
	tap := func(k platform.Key) bool {
		*clock = clock.Add(30 * time.Millisecond)
		r := a.Handle(platform.Event{Key: k, Pressed: true, At: *clock})
		*clock = clock.Add(30 * time.Millisecond)
		return a.Handle(platform.Event{Key: k, Pressed: false, At: *clock}) || r
	}
	if !tap(platform.KeyMenu) || a.screen != ScreenOptions {
		t.Fatalf("Menu on the list: screen %v", a.screen)
	}
	tap(platform.KeyDown)
	row := a.panel.cursor
	if !tap(platform.KeyMenu) || a.screen != ScreenList {
		t.Fatalf("Menu on Options should close it: screen %v", a.screen)
	}
	if !tap(platform.KeyMenu) || a.screen != ScreenOptions || a.panel.cursor != row {
		t.Fatalf("Menu should reopen Options on row %d: screen %v cursor %d", row, a.screen, a.panel.cursor)
	}
	tap(platform.KeyTab)
	if a.screen != ScreenOptions {
		t.Fatalf("X inside Options: screen %v", a.screen)
	}
	tap(platform.KeyBack)
	tap(platform.KeyTab) // Filters
	if !tap(platform.KeyMenu) || a.screen != ScreenOptions {
		t.Fatalf("Menu over Filters should open Options: screen %v", a.screen)
	}
	tap(platform.KeyBack)
	if a.screen != ScreenFilter {
		t.Fatalf("closing Options should return to Filters: screen %v", a.screen)
	}
	tap(platform.KeyBack)
	if quits != 0 || a.screen != ScreenList {
		t.Fatalf("Options mode must never quit: quits %d, screen %v", quits, a.screen)
	}
	a.cfg.MenuButton = "leave"
	if tap(platform.KeyMenu) || quits != 1 || a.screen != ScreenList {
		t.Fatalf("leave: quits %d, screen %v", quits, a.screen)
	}
}
