package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

func TestFavoritesModeLifecycle(t *testing.T) {
	rows := []data.Row{{K: "z", Title: "Zulu"}, {K: "a", Title: "Alpha"}, {K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortFavorites,
		Favorites: map[string]bool{"z": true, "a": true}}, data.Ingest(rows, "", time.Now()), nil)
	if len(a.view) != 2 || a.CursorKey() != "a" || a.filters.FavOnly {
		t.Fatal("favorites mode not independent/alphabetical")
	}
	a.actList(platform.KeyPageDown)
	if a.CursorKey() != "z" || a.top != a.screenLine(a.cursor) || a.notice != "" {
		t.Fatal("favorites letter jump")
	}
	a.actList(platform.KeyEnter)
	a.actDetails(platform.KeySpace)
	if a.screen != ScreenList || len(a.view) != 1 {
		t.Fatal("removing visible favorite")
	}
	a.actList(platform.KeyEnter)
	a.actDetails(platform.KeySpace)
	if len(a.view) != 0 || a.screen != ScreenList || a.emptyListMessage() != "No favorites yet" {
		t.Fatal("removing last favorite did not return to empty favorites view")
	}
	a.actList(platform.KeySpace)
	if a.mode != data.SortUpdated || len(a.view) != 3 {
		t.Fatal("favorites leaked into next mode")
	}
	a.SetSort(data.SortFavorites)
	a.setSearch("Beta")
	if len(a.view) != 0 {
		t.Fatal("search escaped favorites")
	}
}

func TestManualScanDismissAndUpdateNotice(t *testing.T) {
	calls := 0
	a := New(Config{PhysW: 320, PhysH: 240, Action: func(k, arg string) {
		if k == "rescan" {
			calls++
		}
	}},
		data.Ingest(nil, "", time.Now()), nil)
	a.OpenScan()
	if calls != 1 || a.screen != ScreenScan || a.scanReady {
		t.Fatal("scan start")
	}
	a.act(platform.KeyBack)
	a.FinishScan("", true)
	if a.screen != ScreenOptions {
		t.Fatal("scan completion stole focus")
	}
	a.SetAppUpdate("v1.0.6")
	entries := a.optionsEntries()
	if entries[2].kind != "update" || entries[2].text == "Run Update All" {
		t.Fatal("missing update action")
	}
	a.SetAppUpdate("")
	if a.optionsEntries()[2].text != "Run Update All" {
		t.Fatal("stale update notice")
	}
}
