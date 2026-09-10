package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestLetterJumpUsesVisibleCollationGroups(t *testing.T) {
	rows := []data.Row{}
	for _, title := range []string{"!Game", "2 Game", "10 Game", "Alpha", "alpha 2", "Beta", "beta 2", "Éclair", "Elephant", "Zoo"} {
		rows = append(rows, data.Row{K: title, Title: title})
	}
	a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortAlphabetical}, data.Ingest(rows, "test", time.Now()), nil)
	step := func(key platform.Key, want string) {
		t.Helper()
		a.actList(key)
		if a.CursorKey() != want {
			t.Fatalf("%v selected %q; want %q", key, a.CursorKey(), want)
		}
	}
	step(platform.KeyPageDown, "Alpha") // numbers and symbols share #
	step(platform.KeyDown, "alpha 2")
	step(platform.KeyPageDown, "Beta")
	step(platform.KeyDown, "beta 2")
	step(platform.KeyPageUp, "Alpha") // first item of previous group
	step(platform.KeyPageUp, "!Game")
	step(platform.KeyPageUp, "!Game")  // no wrapping
	step(platform.KeyEnd, "Zoo")       // keyboard end remains end
	step(platform.KeyPageUp, "Éclair") // accent uses E, like sorting
	step(platform.KeyPageDown, "Zoo")
	step(platform.KeyPageDown, "Zoo")
	step(platform.KeyHome, "!Game")

	a.cfg.Favorites = map[string]bool{"Alpha": true, "Éclair": true, "Zoo": true}
	a.SetFilters(data.Filters{FavOnly: true})
	step(platform.KeyHome, "Alpha")
	step(platform.KeyPageDown, "Éclair") // B filtered out
	a.setSearch("o")
	step(platform.KeyPageUp, "Zoo") // search restricts the same view
	a.setSearch("no matches")
	step(platform.KeyPageDown, "")
	step(platform.KeyPageUp, "")
}

func TestShouldersKeepEndpointsInDateSorts(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "new", Title: "Alpha", Updated: "2026-09-10", Date: "2026-09-10"},
		{K: "old", Title: "Beta", Updated: "2020-01-01", Date: "2020-01-01"},
	}, "test", time.Now()), nil)
	for _, mode := range []data.SortMode{data.SortUpdated, data.SortDebut} {
		a.SetSort(mode)
		a.actList(platform.KeyPageDown)
		if a.CursorKey() != "old" {
			t.Fatal("R no longer goes to end in date order")
		}
		a.actList(platform.KeyPageUp)
		if a.CursorKey() != "new" {
			t.Fatal("L no longer goes to top in date order")
		}
	}
}
