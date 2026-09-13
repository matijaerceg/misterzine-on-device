package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Unstarring from the list in the Favorites view drops the row and lands
// the cursor on its neighbour; Start does not launch while Select is down.
func TestSelectAFavoritesFromTheList(t *testing.T) {
	a, clock, _, tap := menuApp()
	launches := 0
	a.cfg.Launch = func(string) { launches++ }
	saved := 0
	a.cfg.FavChanged = func() { saved++ }
	key := func(k platform.Key, down bool) {
		*clock = clock.Add(30 * time.Millisecond)
		a.Handle(platform.Event{Key: k, Pressed: down, At: *clock})
	}
	// star the first two rows from the list (Start waits while Select is down)
	first := a.CursorKey()
	key(platform.KeySelect, true)
	tap(platform.KeyEnter)
	tap(platform.KeyStart)
	key(platform.KeySelect, false)
	tap(platform.KeyDown)
	second := a.CursorKey()
	key(platform.KeySelect, true)
	tap(platform.KeyEnter)
	key(platform.KeySelect, false)
	if !a.cfg.Favorites[first] || !a.cfg.Favorites[second] || len(a.cfg.Favorites) != 2 || saved != 2 || launches != 0 {
		t.Fatalf("favorites %v, saved %d, launches %d", a.cfg.Favorites, saved, launches)
	}
	a.SetSort(data.SortFavorites)
	if len(a.view) != 2 {
		t.Fatalf("Favorites view has %d rows", len(a.view))
	}
	a.cursor = 0
	gone, stays := a.CursorKey(), a.ds.Rows[a.view[1]].K
	key(platform.KeySelect, true)
	tap(platform.KeyEnter)
	key(platform.KeySelect, false)
	if a.cfg.Favorites[gone] || len(a.view) != 1 || a.CursorKey() != stays || a.screen != ScreenList {
		t.Fatalf("after unstarring %s: favorites %v, view %d rows, cursor on %q, screen %v", gone, a.cfg.Favorites, len(a.view), a.CursorKey(), a.screen)
	}
	key(platform.KeySelect, true)
	tap(platform.KeyEnter)
	key(platform.KeySelect, false)
	if len(a.view) != 0 || a.cursor != 0 {
		t.Fatalf("after unstarring the last row: %d rows, cursor %d", len(a.view), a.cursor)
	}
	tap(platform.KeyEnter) // A on an empty view stays put
	if a.screen != ScreenList {
		t.Fatalf("screen %v", a.screen)
	}
}
