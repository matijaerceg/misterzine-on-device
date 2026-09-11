package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"testing"
	"time"
)

func TestStrictINIFilterAndManualRestoration(t *testing.T) {
	rows := []data.Row{
		{K: "h", Title: "Horizontal", Base: "Arcade", Rot: "Horizontal"},
		{K: "v", Title: "Vertical", Base: "Arcade", Rot: "Vertical"},
		{K: "unknown", Title: "Unknown", Base: "Arcade"},
		{K: "system", Title: "System", Base: "Console"},
	}
	for _, ini := range []string{"h", "v"} {
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotNone, IniOrientation: ini,
			FilterRotation: true, FollowRotation: false, Favorites: map[string]bool{"h": true, "v": true, "unknown": true}},
			data.Ingest(rows, "", time.Now()), nil)
		// Even a contradictory manual rotation filter is preserved and superseded.
		a.SetFilters(data.Filters{RotOff: map[string]bool{ini: true}, SrcOff: map[string]bool{"excluded": true}})
		if len(a.view) != 1 || a.CursorKey() != ini {
			t.Fatal("not strict", ini, a.view)
		}
		a.SetSort(data.SortFavorites)
		if len(a.view) != 1 || a.CursorKey() != ini {
			t.Fatal("favorites escaped strict rule")
		}
		a.SetRotation(gfx.RotRight)
		if len(a.view) != 1 || a.CursorKey() != ini {
			t.Fatal("UI rotation changed INI filter")
		}
		counts := a.facetCounts("base")
		if counts["Console"] != 0 || counts["Arcade"] != 1 {
			t.Fatal("counts ignored INI", counts)
		}
		a.openPanel(ScreenOptions)
		for i, e := range a.panel.entries {
			if e.kind == "filter-rotation" {
				a.panel.cursor = i
				break
			}
		}
		a.stepValue(-1)
		if a.FilterRotation() || !a.filters.RotOff[ini] || !a.filters.SrcOff["excluded"] {
			t.Fatal("lost manual filters")
		}
		if len(a.view) != 2 {
			t.Fatal("off did not restore manual view", a.view)
		}
		a.stepValue(1)
		if len(a.view) != 1 || a.CursorKey() != ini {
			t.Fatal("on did not apply immediately")
		}
		a.SetFilters(data.Filters{})
		if len(a.view) != 1 {
			t.Fatal("clearing manual filters bypassed strict rule")
		}
	}
}

func TestMissingINIRotationDoesNotGuess(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, FilterRotation: true}, data.Ingest([]data.Row{{K: "unknown"}}, "", time.Now()), nil)
	if a.iniFilter() != "" || len(a.view) != 1 {
		t.Fatal("missing INI guessed a rotation")
	}
}
