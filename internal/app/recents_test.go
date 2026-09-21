package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestRecentsViewLifecycle(t *testing.T) {
	rows := []data.Row{{Base: "Arcade", K: "z", Title: "Zulu", MRA: "_Arcade/z.mra"}, {Base: "Arcade", K: "a", Title: "Alpha", MRA: "_Arcade/a.mra"}, {Base: "Arcade", K: "b", Title: "Beta", MRA: "_Arcade/b.mra"}}
	clock := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	launched, changed := 0, 0
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock }, ViewsOff: []string{"recents"},
		Launch: func(string) { launched++ }, RecentsChanged: func() { changed++ },
		Exists: func(string) bool { return true }},
		data.Ingest(rows, "", clock), nil)

	// off (a fresh install's Views): Y skips Recents and SetSort refuses it
	a.SetSort(data.SortFavorites)
	a.actList(platform.KeySpace)
	if a.mode != data.SortUpdated {
		t.Fatal("recents view joined the cycle while off")
	}
	a.SetSort(data.SortRecents)
	if a.mode != data.SortUpdated {
		t.Fatal("recents view selectable while off")
	}

	// launches are recorded either way, newest first, each key once
	a.moveToKey("a")
	a.actList(platform.KeyStart)
	clock = clock.Add(time.Hour)
	a.moveToKey("z")
	a.actList(platform.KeyStart)
	clock = clock.Add(time.Hour)
	a.moveToKey("a")
	a.actList(platform.KeyStart)
	if launched != 3 || changed != 3 {
		t.Fatalf("launched %d changed %d", launched, changed)
	}
	if r := a.Recents(); len(r) != 2 || r[0].K != "a" || r[1].K != "z" || r[0].At != "2026-09-11T14:00:00Z" {
		t.Fatalf("history %+v", r)
	}

	// on: Favorites -> Recents -> Updated, launch order, launch date, count
	a.setViewOn(data.SortRecents, true)
	a.SetSort(data.SortFavorites)
	a.actList(platform.KeySpace)
	if a.mode != data.SortRecents || len(a.view) != 2 || a.CursorKey() != "a" || a.ds.Rows[a.view[1]].K != "z" {
		t.Fatalf("recents view: mode %v view %v", a.mode, a.view)
	}
	if a.launchedAt("a") != "2026-09-11" || a.launchedAt("b") != "" || a.jumpGroupKey(0) != "2026-09" {
		t.Fatal("launch dates")
	}
	a.actList(platform.KeySpace)
	if a.mode != data.SortUpdated {
		t.Fatal("recents did not wrap to updated")
	}

	// taking the view out on the Views page while in it moves on to the next view on
	a.SetSort(data.SortRecents)
	a.openViews()
	for i, e := range a.panel.entries {
		if e.kind == "view" && e.value == "recents" {
			a.panel.cursor = i
		}
	}
	a.togglePanel()
	if a.viewOn(data.SortRecents) || a.mode != data.SortUpdated {
		t.Fatalf("view off left the recents view: on %v mode %v", a.viewOn(data.SortRecents), a.mode)
	}

	// empty history message and the cap
	b := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortRecents, Now: func() time.Time { return clock }}, data.Ingest(rows, "", clock), nil)
	if b.mode != data.SortRecents || b.emptyListMessage() != "No launches yet" {
		t.Fatal("empty recents view")
	}
	for i := 0; i < maxRecents+5; i++ {
		b.cfg.RecentLaunches = append(b.cfg.RecentLaunches, data.Recent{K: "old" + itoa(i), At: "2026-01-01T00:00:00Z"})
	}
	b.recordLaunch(&rows[2])
	if len(b.Recents()) != maxRecents || b.Recents()[0].K != "b" {
		t.Fatal("history cap")
	}
	// a remembered Recents sort is ignored while the view is off
	c := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortRecents, ViewsOff: []string{"recents"}}, data.Ingest(rows, "", clock), nil)
	if c.mode != data.SortUpdated {
		t.Fatal("remembered recents restored while off")
	}
}

// Reopened by the launcher after a game (--resume), the app lands on that
// game: in the view it starts in when that view lists it, otherwise back
// in the view the launch was made from, which a launch records. A launch
// from before views were recorded, or one the filters hide, leaves the
// list at the top.
func TestResumeLandsOnTheLaunchedGameInItsView(t *testing.T) {
	rows := []data.Row{{Base: "Arcade", K: "z", Title: "Zulu", MRA: "_Arcade/z.mra", Updated: "2026-01-03"}, {Base: "Arcade", K: "a", Title: "Alpha", MRA: "_Arcade/a.mra", Updated: "2026-01-01"}, {Base: "Arcade", K: "b", Title: "Beta", MRA: "_Arcade/b.mra", Updated: "2026-01-02"}}
	clock := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock }, Launch: func(string) {},
		Exists: func(string) bool { return true }, Status: func(int) data.Status { return data.StatusCurrent }},
		data.Ingest(rows, "", clock), nil)
	a.SetSort(data.SortAlphabetical)
	a.moveToKey("b")
	a.actList(platform.KeyStart)
	if r := a.Recents()[0]; r.K != "b" || r.View != "alphabetical" {
		t.Fatalf("launch recorded as %+v", r)
	}
	// the Recents view lists only what was launched: b is there, so no view change
	a.SetSort(data.SortRecents)
	a.cursor = 0
	if !a.ResumeAt(a.Recents()[0]) || a.CursorKey() != "b" || a.Sort() != data.SortRecents {
		t.Fatalf("resume in a view that lists the game: cursor %q, view %v", a.CursorKey(), a.Sort())
	}
	// a game in Recents but starting in Favorites (empty): back to the launch view
	a.SetSort(data.SortFavorites)
	if !a.ResumeAt(data.Recent{K: "b", View: "alphabetical"}) || a.Sort() != data.SortAlphabetical || a.CursorKey() != "b" {
		t.Fatalf("resume from an empty view: view %v, cursor %q", a.Sort(), a.CursorKey())
	}
	// the launch view since turned off, or unrecorded: the list stays put
	a.SetSort(data.SortFavorites)
	if a.ResumeAt(data.Recent{K: "b"}) || a.Sort() != data.SortFavorites {
		t.Fatalf("an unrecorded launch view moved the list: view %v", a.Sort())
	}
	a.cfg.ViewsOff = []string{"alphabetical"}
	a.viewsOff = parseViewsOff(a.cfg.ViewsOff)
	if a.ResumeAt(data.Recent{K: "b", View: "alphabetical"}) || a.Sort() != data.SortFavorites {
		t.Fatalf("a launch view turned off was reopened: view %v", a.Sort())
	}
	a.viewsOff = nil
	// hidden by a filter: found nowhere, the list stays at the top
	a.SetSort(data.SortAlphabetical)
	a.SetFilters(data.Filters{FavOnly: true})
	if a.ResumeAt(data.Recent{K: "b", View: "alphabetical"}) {
		t.Fatal("a filtered-out game was reported found")
	}
}
