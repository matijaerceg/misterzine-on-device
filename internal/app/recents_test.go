package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestRecentsViewLifecycle(t *testing.T) {
	rows := []data.Row{{K: "z", Title: "Zulu", MRA: "_Arcade/z.mra"}, {K: "a", Title: "Alpha", MRA: "_Arcade/a.mra"}, {K: "b", Title: "Beta", MRA: "_Arcade/b.mra"}}
	clock := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	launched, changed := 0, 0
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock },
		Launch: func(string) { launched++ }, RecentsChanged: func() { changed++ },
		Exists: func(string) bool { return true }},
		data.Ingest(rows, "", clock), nil)

	// off by default: Y skips Recents and SetSort refuses it
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
	a.cfg.Recents = true
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

	// turning the option off while in the view falls back to updated
	a.SetSort(data.SortRecents)
	a.screen = ScreenOptions
	a.buildPanel()
	for i, e := range a.panel.entries {
		if e.kind == "recents" {
			a.panel.cursor = i
		}
	}
	a.stepValue(-1)
	if a.cfg.Recents || a.mode != data.SortUpdated {
		t.Fatal("option off left the recents view")
	}

	// empty history message and the cap
	a.cfg.Recents = true
	b := New(Config{PhysW: 320, PhysH: 240, Recents: true, RememberSort: true, LastSort: data.SortRecents, Now: func() time.Time { return clock }}, data.Ingest(rows, "", clock), nil)
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
	c := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortRecents}, data.Ingest(rows, "", clock), nil)
	if c.mode != data.SortUpdated {
		t.Fatal("remembered recents restored while off")
	}
}
