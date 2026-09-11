package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"testing"
	"time"
)

func TestStrictRotationFilterAndManualRestoration(t *testing.T) {
	rows := []data.Row{
		{K: "h", Title: "Horizontal", Base: "Arcade", Rot: "Horizontal"},
		{K: "v", Title: "Vertical", Base: "Arcade", Rot: "Vertical"},
		{K: "unknown", Title: "Unknown", Base: "Arcade"},
		{K: "system", Title: "System", Base: "Console"},
	}
	for _, tc := range []struct {
		rot  gfx.Rotation
		want string
	}{{gfx.RotNone, "h"}, {gfx.RotLeft, "v"}, {gfx.RotRight, "v"}} {
		want := tc.want
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: tc.rot,
			FilterRotation: true, FollowRotation: false, Favorites: map[string]bool{"h": true, "v": true, "unknown": true}},
			data.Ingest(rows, "", time.Now()), nil)
		// Even a contradictory manual rotation filter is preserved and superseded.
		a.SetFilters(data.Filters{RotOff: map[string]bool{want: true}, SrcOff: map[string]bool{"excluded": true}})
		if len(a.view) != 1 || a.CursorKey() != want {
			t.Fatal("not strict", want, a.view)
		}
		a.SetSort(data.SortFavorites)
		if len(a.view) != 1 || a.CursorKey() != want {
			t.Fatal("favorites escaped strict rule")
		}
		counts := a.facetCounts("base")
		if counts["Console"] != 0 || counts["Arcade"] != 1 {
			t.Fatal("counts ignored rotation", counts)
		}
		a.openPanel(ScreenOptions)
		for i, e := range a.panel.entries {
			if e.kind == "filter-rotation" {
				a.panel.cursor = i
				break
			}
		}
		a.stepValue(-1)
		if a.FilterRotation() || !a.filters.RotOff[want] || !a.filters.SrcOff["excluded"] {
			t.Fatal("lost manual filters")
		}
		if len(a.view) != 2 {
			t.Fatal("off did not restore manual view", a.view)
		}
		a.stepValue(1)
		if len(a.view) != 1 || a.CursorKey() != want {
			t.Fatal("on did not apply immediately")
		}
		a.SetFilters(data.Filters{})
		if len(a.view) != 1 {
			t.Fatal("clearing manual filters bypassed strict rule")
		}
	}
}

func TestRotationFilterFollowsRotationChanges(t *testing.T) {
	rows := []data.Row{
		{K: "h", Title: "Horizontal", Base: "Arcade", Rot: "Horizontal"},
		{K: "v", Title: "Vertical", Base: "Arcade", Rot: "Vertical"},
	}
	a := New(Config{PhysW: 320, PhysH: 240, Rotation: gfx.RotNone, FilterRotation: true},
		data.Ingest(rows, "", time.Now()), nil)
	if len(a.view) != 1 || a.CursorKey() != "h" || a.rotationFilterLabel() != "Rotation: horizontal only" {
		t.Fatal("horizontal start", a.view)
	}
	a.SetRotation(gfx.RotLeft)
	if len(a.view) != 1 || a.CursorKey() != "v" || a.rotationFilterLabel() != "Rotation: vertical only" {
		t.Fatal("manual rotation did not move the filter", a.view)
	}
	a.SetRotation(gfx.RotRight)
	if len(a.view) != 1 || a.CursorKey() != "v" {
		t.Fatal("same orientation must keep the view")
	}
	a.SetRotation(gfx.RotNone)
	if len(a.view) != 1 || a.CursorKey() != "h" {
		t.Fatal("back to horizontal")
	}
}
