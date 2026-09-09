package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestKeyboardFindCombinesWithFiltersAndClears(t *testing.T) {
	now := time.Now()
	rows := []data.Row{{K: "one", Title: "Space Invaders", Base: "Arcade"}, {K: "two", Title: "SpaceInvaders II", Base: "Console"}, {K: "three", Title: "Invaders", Base: "Arcade"}}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(rows, "test", now), nil)
	tap := func(k platform.Key) {
		a.Handle(platform.Event{Key: k, Pressed: true, At: now})
		a.Handle(platform.Event{Key: k, At: now})
	}
	for _, ch := range "SPACE inv" {
		a.Handle(platform.Event{Text: ch, Pressed: true, At: now})
	}
	if len(a.view) != 2 || a.Search() != "SPACE inv" {
		t.Fatalf("case/space matching failed: %q, %d", a.Search(), len(a.view))
	}
	a.SetFilters(data.Filters{BaseOff: map[string]bool{"Console": true}})
	if len(a.view) != 1 || a.CursorKey() != "one" {
		t.Fatal("find ignored Filters")
	}
	// Gamepad Y remains sorting even with a query; keyboard Space appends.
	before := a.Sort()
	tap(platform.KeySpace)
	if a.Sort() == before || a.Search() != "SPACE inv" {
		t.Fatal("Y stopped sorting")
	}
	a.Handle(platform.Event{Key: platform.KeySpace, Text: ' ', Pressed: true, At: now})
	if a.Sort() == before || a.Search() != "SPACE inv " {
		t.Fatal("keyboard space sorted")
	}
	tap(platform.KeyBackspace)
	if a.Search() != "SPACE inv" {
		t.Fatal("Backspace did not edit")
	}
	tap(platform.KeyEnter)
	if a.Screen() != ScreenDetails {
		t.Fatal("could not browse a match")
	}
	a.Handle(platform.Event{Text: 'x', Pressed: true, At: now})
	tap(platform.KeyBack)
	if a.Search() != "SPACE inv" {
		t.Fatal("details typing changed search")
	}
	a.Handle(platform.Event{Text: 'z', Pressed: true, At: now})
	if len(a.view) != 0 || a.emptyListMessage() != "no matches, B: clear find" {
		t.Fatal("missing empty-query feedback")
	}
	tap(platform.KeyBack)
	if a.Screen() != ScreenList || a.Search() != "" || len(a.view) != 2 || !a.Filters().BaseOff["Console"] {
		t.Fatal("B did not clear just the query")
	}
	tap(platform.KeyBack)
	if a.Screen() != ScreenOptions {
		t.Fatal("second B did not open Options")
	}
}

func TestFreshArtworkAlwaysStartsAtFirstShot(t *testing.T) {
	now := time.Now()
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest([]data.Row{
		{K: "one", Title: "One", Img: "one", ImgSlots: []string{"title", "snap"}},
		{K: "two", Title: "Two", Img: "two", ImgSlots: []string{"title", "snap"}},
	}, "test", now), nil)
	tap := func(k platform.Key) {
		a.Handle(platform.Event{Key: k, Pressed: true, At: now})
		a.Handle(platform.Event{Key: k, At: now})
		a.Paint()
		now = now.Add(10 * time.Millisecond)
	}
	a.MoveToKey("one")
	tap(platform.KeyEnter)
	tap(platform.KeyEnter)
	if a.Screen() != ScreenShot || a.slot != 0 {
		t.Fatal("rapid A presses did not open first artwork")
	}
	tap(platform.KeyRight)
	if a.slot != 1 {
		t.Fatal("fixture did not select second artwork")
	}
	tap(platform.KeyBack)
	tap(platform.KeyBack)
	a.MoveToKey("two")
	tap(platform.KeyEnter)
	tap(platform.KeyEnter)
	if a.Screen() != ScreenShot || a.slot != 0 {
		t.Fatal("another game inherited the previous shot")
	}
}
