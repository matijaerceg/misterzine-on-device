package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"testing"
	"time"
)

func TestDeprecatedCataloguePreference(t *testing.T) {
	changes := 0
	a := New(Config{ShowNonArcade: true, PhysW: 320, PhysH: 240, Favorites: map[string]bool{"old": true}, SettingsChanged: func() { changes++ }}, data.Ingest([]data.Row{
		{K: "old", Title: "Old", Base: "Console", Deprecated: true},
		{K: "new", Title: "New", Base: "Console"},
	}, "", time.Now()), nil)
	if len(a.view) != 1 || a.total != 1 || a.filtersActive() {
		t.Fatal("default catalogue should quietly hide deprecated cores")
	}
	a.query = "Old"
	a.rebuild()
	if len(a.view) != 0 {
		t.Fatal("search exposed deprecated core")
	}
	a.query = ""
	a.SetSort(data.SortFavorites)
	if len(a.view) != 0 || !a.cfg.Favorites["old"] {
		t.Fatal("hidden favourite must be preserved")
	}
	a.openPanel(ScreenOptions)
	expandOptionsForTest(a)
	for i, e := range a.panel.entries {
		if e.kind == "show-deprecated" {
			a.panel.cursor = i
			break
		}
	}
	a.stepValue(1)
	if !a.ShowDeprecated() || len(a.view) != 1 || a.total != 2 || changes == 0 {
		t.Fatal("option failed to restore favourite")
	}
	a.stepValue(-1)
	a.filters = data.Filters{BaseOff: map[string]bool{"Console": true}}
	a.openPanel(ScreenFilter)
	expandOptionsForTest(a)
	for i, e := range a.panel.entries {
		if e.kind == "clear" {
			a.panel.cursor = i
			break
		}
	}
	a.togglePanel()
	if a.ShowDeprecated() || len(a.view) != 0 || !a.cfg.Favorites["old"] {
		t.Fatal("clearing filters changed catalogue preference or favourite")
	}
	a.SetSort(data.SortAlphabetical)
	if len(a.view) != 1 || a.cardCounts()[data.StatusUnknown] != 1 {
		t.Fatal("counts include deprecated core")
	}
}
