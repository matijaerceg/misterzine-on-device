package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func TestFavoritesModeLifecycle(t *testing.T) {
	rows := []data.Row{{Base: "Arcade", K: "z", Title: "Zulu"}, {Base: "Arcade", K: "a", Title: "Alpha"}, {Base: "Arcade", K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortFavorites, ViewsOff: []string{"recents"},
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
	// no Downloader on this card: the app-only row is not offered, since
	// pressing it could not start anything
	if entries[2].kind != "update" || entries[2].text != "Update MisterZine + all" {
		t.Fatalf("without Downloader the app-only row showed: %q", entries[2].text)
	}
	a.cfg.CanUpdateApp = func() bool { return true }
	entries = a.optionsEntries()
	if entries[2].kind != "update-app" || entries[2].text != "Update MisterZine only" || entries[3].kind != "update" || entries[3].text != "Update MisterZine + all" {
		t.Fatalf("update actions with a new version out: %q, %q", entries[2].text, entries[3].text)
	}
	a.SetAppUpdate("")
	if entries = a.optionsEntries(); entries[2].text != "Run Update All" || entries[3].kind != "update-result" {
		t.Fatal("stale update notice")
	}
}

// With a new MisterZine announced, Options offers to fetch it alone: A on
// that row opens the update screen for a MisterZine update and asks the
// host for the app-only run; the ordinary row still asks for Update All.
func TestUpdateMisterzineOnlyRow(t *testing.T) {
	now := time.Unix(1788900000, 0)
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now },
		CanUpdateApp: func() bool { return true }}, data.Ingest(nil, "test", now), nil)
	var actions []string
	a.cfg.Action = func(kind, arg string) { actions = append(actions, kind+"="+arg) }
	a.SetAppUpdate("v1.0.6")
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "update-app" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyEnter)
	if a.screen != ScreenUpdate || a.update.Mode != updater.ModeApp || a.update.Label != "Starting MisterZine update" {
		t.Fatalf("screen %v, run %+v", a.screen, a.update)
	}
	if strings.Join(actions, " ") != "update=app" {
		t.Fatalf("actions %v", actions)
	}
	c := gfx.New(320, 240)
	a.paintUpdate(c)
	a.SetUpdate(updater.State{ID: "run", Mode: updater.ModeApp, Status: "completed", Started: now, Heartbeat: now}, true)
	if a.update.Summary() != "MisterZine update finished successfully" {
		t.Fatalf("summary %q", a.update.Summary())
	}
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "update" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyEnter)
	if a.update.Mode != updater.ModeAll || strings.Join(actions, " ") != "update=app update=all" {
		t.Fatalf("run %+v, actions %v", a.update, actions)
	}
}
