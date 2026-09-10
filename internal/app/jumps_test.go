package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestLetterJumpsPutGroupAtTopInBothDirections(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, groupSize := range []int{2, 30} {
			rows := []data.Row{}
			for _, initial := range []string{"A", "B", "Z"} {
				for i := 0; i < groupSize; i++ {
					title := fmt.Sprintf("%s %02d", initial, i)
					rows = append(rows, data.Row{K: title, Title: title})
				}
			}
			a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, RememberSort: true, LastSort: data.SortAlphabetical}, data.Ingest(rows, "test", time.Now()), nil)
			jump := func(key platform.Key, want string) {
				t.Helper()
				a.actList(key)
				a.Paint()
				if a.CursorKey() != want || a.top != a.screenLine(a.cursor) {
					t.Fatalf("rotation=%v size=%d: %v selected %q at line %d, top %d; want %q at top", rot, groupSize, key, a.CursorKey(), a.screenLine(a.cursor), a.top, want)
				}
			}
			jump(platform.KeyPageDown, "B 00")
			jump(platform.KeyPageDown, "Z 00")
			// A short final group remains at the top when walking within it.
			top := a.top
			a.actList(platform.KeyDown)
			a.Paint()
			if a.top != top {
				t.Fatal("walking within the last letter pulled earlier groups into view")
			}
			jump(platform.KeyPageUp, "B 00")
			jump(platform.KeyPageUp, "A 00")
			jump(platform.KeyPageDown, "B 00")
			a.actList(platform.KeyEnd)
			if a.top != max(0, len(a.view)-a.lay.Lines) {
				t.Fatal("End should still fill the final page")
			}
		}
	}
}

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

func TestMonthJumpsUseSortDateVisibleGroupsAndTopAlignment(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "new", Title: "Newest", Updated: "2026-09-10", Date: "2024-02-10"},
		{K: "same-month", Title: "Same update month", Updated: "2026-09-01", Date: "2024-01-01"},
		{K: "july", Title: "July 2026", Updated: "2026-07-15", Date: "2023-12-15"},
		{K: "older-july", Title: "July 2025", Updated: "2025-07-20", Date: "2023-01-20"},
		{K: "undated", Title: "Undated"},
	}
	base := map[string]string{}
	for _, row := range rows {
		base[row.K] = row.Updated
	}
	base["new"] = "2026-08-01"
	for _, mode := range []data.SortMode{data.SortUpdated, data.SortDebut} {
		a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: mode, Now: func() time.Time { return now }, ClockTrusted: true},
			data.Ingest(rows, "test", now), &data.SeenRecord{T: now.Format(time.RFC3339), Base: base})
		jump := func(key platform.Key, want, label string) {
			t.Helper()
			a.actList(key)
			a.Paint()
			if a.CursorKey() != want || a.top != a.screenLine(a.cursor) || a.notice != label {
				t.Fatalf("%v %v: got %q at top %d (row %d), notice %q; want %q / %q", mode, key, a.CursorKey(), a.top, a.screenLine(a.cursor), a.notice, want, label)
			}
		}
		if mode == data.SortUpdated {
			if a.split < 0 {
				t.Fatal("fixture must exercise the last-look divider")
			}
			jump(platform.KeyPageDown, "july", "July 2026")
			jump(platform.KeyPageDown, "older-july", "July 2025")
			jump(platform.KeyPageDown, "undated", "Date unknown")
			jump(platform.KeyPageDown, "undated", "Date unknown")
			jump(platform.KeyPageUp, "older-july", "July 2025")
			jump(platform.KeyPageUp, "july", "July 2026")
			jump(platform.KeyPageUp, "new", "September 2026")
		} else {
			jump(platform.KeyPageDown, "same-month", "January 2024")
			jump(platform.KeyPageDown, "july", "December 2023")
			jump(platform.KeyPageDown, "older-july", "January 2023")
			jump(platform.KeyPageUp, "july", "December 2023")
			jump(platform.KeyPageUp, "same-month", "January 2024")
			jump(platform.KeyPageUp, "new", "February 2024")
		}
		a.cfg.Favorites = map[string]bool{"new": true, "older-july": true, "undated": true}
		a.SetFilters(data.Filters{FavOnly: true})
		a.actList(platform.KeyHome)
		label := "July 2025"
		if mode == data.SortDebut {
			label = "January 2023"
		}
		jump(platform.KeyPageDown, "older-july", label)
		a.setSearch("July")
		a.actList(platform.KeyPageDown)
		if a.CursorKey() != "older-july" {
			t.Fatal("jump escaped search results")
		}
		a.setSearch("no matches")
		a.actList(platform.KeyPageUp)
		a.actList(platform.KeyPageDown)
		if a.CursorKey() != "" {
			t.Fatal("empty results acquired a selection")
		}
	}
}
