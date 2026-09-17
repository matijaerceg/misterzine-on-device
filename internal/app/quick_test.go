package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Select held turns Y and X into the layout and shot toggles; released,
// they sort and open Filters again.
func TestSelectChordsToggleLayoutAndShots(t *testing.T) {
	a, clock, _, _ := menuApp() // three rows, so Select + A has something to star
	changed := 0
	a.cfg.SettingsChanged = func() { changed++ }
	key := func(k platform.Key, down bool) bool {
		*clock = clock.Add(30 * time.Millisecond)
		return a.Handle(platform.Event{Key: k, Pressed: down, At: *clock})
	}
	tap := func(k platform.Key) { key(k, true); key(k, false) }
	sort := a.Sort()
	if !key(platform.KeySelect, true) {
		t.Fatal("pressing Select should repaint for the chord legend")
	}
	tap(platform.KeySpace)
	if a.ListLayout() != "split" || a.Sort() != sort {
		t.Fatalf("Select+Y: layout %q, sort %v -> %v", a.ListLayout(), sort, a.Sort())
	}
	tap(platform.KeySpace)
	tap(platform.KeySpace)
	if a.ListLayout() != "text" {
		t.Fatalf("Select+Y should reach text, got %q", a.ListLayout())
	}
	tap(platform.KeySpace)
	if a.ListLayout() != "list" {
		t.Fatalf("Select+Y should cycle back to list, got %q", a.ListLayout())
	}
	tap(platform.KeyTab)
	if a.ListShot() != "title" || a.screen != ScreenList {
		t.Fatalf("Select+X: shot %q, screen %v", a.ListShot(), a.screen)
	}
	if changed != 5 {
		t.Fatalf("settings changed %d times, want 5", changed)
	}
	// Select + A stars the row; the other buttons wait while Select is down
	if a.cursor != 0 {
		t.Fatalf("cursor %d", a.cursor)
	}
	k0 := a.CursorKey()
	tap(platform.KeyEnter)
	if !a.cfg.Favorites[k0] || a.screen != ScreenList || a.notice != "favorite added" {
		t.Fatalf("Select+A: favorite %v, screen %v, notice %q", a.cfg.Favorites[k0], a.screen, a.notice)
	}
	tap(platform.KeyEnter)
	if a.cfg.Favorites[k0] || a.notice != "favorite removed" {
		t.Fatalf("Select+A again: favorite %v, notice %q", a.cfg.Favorites[k0], a.notice)
	}
	tap(platform.KeyDown)
	tap(platform.KeyRight)
	tap(platform.KeyPageDown)
	tap(platform.KeyEnd)
	tap(platform.KeyBack)
	if a.cursor != 0 || a.screen != ScreenList {
		t.Fatalf("with Select held nothing should move or open: cursor %d, screen %v", a.cursor, a.screen)
	}
	if !key(platform.KeySelect, false) {
		t.Fatal("releasing Select should repaint for the ordinary legend")
	}
	tap(platform.KeyDown)
	if a.cursor != 1 {
		t.Fatalf("Down after the release: cursor %d", a.cursor)
	}
	tap(platform.KeyUp)
	tap(platform.KeySpace)
	if a.Sort() == sort || a.ListLayout() != "list" {
		t.Fatalf("Y without Select: sort %v -> %v, layout %q", sort, a.Sort(), a.ListLayout())
	}
	tap(platform.KeyTab)
	if a.screen != ScreenFilter {
		t.Fatalf("X without Select should open Filters, screen %v", a.screen)
	}
}

// The text layout is rows alone, so the Select+X chord is left out of the
// legend and changes nothing; it comes back with a layout that has a
// picture. Options keeps the Art type row either way.
func TestSelectShotChordIdleInTextLayout(t *testing.T) {
	a, clock, _, _ := menuApp()
	changed := 0
	a.cfg.SettingsChanged = func() { changed++ }
	key := func(k platform.Key, down bool) bool {
		*clock = clock.Add(30 * time.Millisecond)
		return a.Handle(platform.Event{Key: k, Pressed: down, At: *clock})
	}
	a.cfg.ListLayout = "text"
	a.setRotation(a.rot) // the pane, and the picture with it, goes away
	key(platform.KeySelect, true)
	if hint := a.listHint(); strings.Contains(hint, "X Shots") || !strings.Contains(hint, "Y Layout") {
		t.Fatalf("text layout: the chord legend should skip X, got %q", hint)
	}
	if key(platform.KeyTab, true) || a.ListShot() != "gameplay" || a.notice != "" || changed != 0 {
		t.Fatalf("Select+X in the text layout: shot %q, notice %q, saved %d times", a.ListShot(), a.notice, changed)
	}
	key(platform.KeyTab, false)

	a.cfg.ListLayout = "list"
	a.setRotation(a.rot)
	if hint := a.listHint(); !strings.Contains(hint, "X Shots") {
		t.Fatalf("list layout: the chord legend should name X, got %q", hint)
	}
	key(platform.KeyTab, true)
	key(platform.KeyTab, false)
	if a.ListShot() != "title" || changed != 1 {
		t.Fatalf("Select+X with a picture: shot %q, saved %d times", a.ListShot(), changed)
	}
	// Options offers it in either layout: the choice applies when you go back
	a.cfg.ListLayout = "text"
	a.setRotation(a.rot)
	a.screen = ScreenOptions
	a.buildPanel()
	expandOptionsForTest(a)
	row := -1
	for i, e := range a.panel.entries {
		if e.kind == "list-shot" {
			row = i
		}
	}
	if row < 0 || a.panel.entries[row].disabled {
		t.Fatal("Options should keep the Art type row live in the text layout")
	}
	a.panel.cursor = row
	if !a.stepValue(-1) || a.ListShot() != "gameplay" {
		t.Fatalf("Options should still set Art type in the text layout, got %q", a.ListShot())
	}
}
