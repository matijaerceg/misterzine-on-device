package app

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestYearDecadeFilteringAndPersistence(t *testing.T) {
	rows := []data.Row{
		{K: "a", Title: "Game A", Base: "Arcade", Year: "1980", Rot: "Vertical"},
		{K: "b", Title: "Game B", Base: "Arcade", Year: "1989", Rot: "Horizontal"},
		{K: "c", Title: "Game C", Base: "Arcade", Year: "1990", Rot: "Vertical"},
		{K: "u", Title: "Game U", Base: "Arcade", Year: "198?"},
		{K: "s", Title: "System", Base: "Console", Year: "1980"},
	}
	saves := 0
	a := New(Config{PhysW: 320, PhysH: 240, FiltersChanged: func() { saves++ }}, data.Ingest(rows, "", time.Now()), nil)
	openExpandedFilters(a)
	choose := func(kind, value string, header bool) {
		t.Helper()
		for i, e := range a.panel.entries {
			if e.kind == kind && e.value == value && e.header == header {
				a.panel.cursor = i
				return
			}
		}
		t.Fatalf("missing %s/%s", kind, value)
	}
	choose("decade", "1980s", false)
	before := a.panel.cursor
	a.actPanel(platform.KeyRight)
	a.actPanel(platform.KeyRight)
	if a.panel.cursor != before || !a.panel.yearOpen["1980s"] {
		t.Fatal("Right must expand, never page or toggle closed")
	}
	a.actPanel(platform.KeyLeft)
	if a.panel.cursor != before || a.panel.yearOpen["1980s"] {
		t.Fatal("Left must collapse, never page or toggle open")
	}
	choose("decade", "1980s", false)
	a.actPanel(platform.KeySpace)
	if len(a.view) != 3 || a.filters.YearOff["1980"] || !a.filters.YearOff["1990"] || !a.filters.YearOff[""] {
		t.Fatal("decade isolation or system exemption failed")
	}
	a.actPanel(platform.KeySpace)
	if len(a.view) != 5 || len(a.filters.YearOff) != 0 {
		t.Fatal("second Y must enable all years")
	}
	a.actPanel(platform.KeyTab)
	choose("year", "1989", false)
	a.actPanel(platform.KeyLeft)
	if a.panel.entries[a.panel.cursor].kind != "decade" || a.panel.yearOpen["1980s"] {
		t.Fatal("Left must collapse to parent decade")
	}
	a.actPanel(platform.KeyRight)
	choose("year", "1989", false)
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 4 {
		t.Fatal("single year toggle failed")
	}
	choose("decade", "1980s", false)
	if !a.panel.entries[a.panel.cursor].partial {
		t.Fatal("missing mixed decade marker")
	}
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 5 {
		t.Fatal("mixed decade toggle must enable whole decade")
	}
	choose("year", "1980", false)
	a.actPanel(platform.KeySpace)
	if len(a.view) != 2 {
		t.Fatal("exact year only failed")
	}
	encoded, err := json.Marshal(a.filters)
	if err != nil {
		t.Fatal(err)
	}
	var restored data.Filters
	if err = json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	a.SetFilters(restored)
	choose("year", "1980", false)
	a.actPanel(platform.KeyTab)
	if e := a.panel.entries[a.panel.cursor]; e.kind != "decade" || e.value != "1980s" {
		t.Fatal("collapse lost parent cursor", e)
	}
	if len(a.view) != 2 {
		t.Fatal("collapse changed selection")
	}
	a.actPanel(platform.KeyTab)
	choose("year", "1980", false)
	a.actPanel(platform.KeySpace)
	if len(a.view) != 5 {
		t.Fatal("second Y after restore must enable all")
	}
	choose("year", "", true)
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 1 || a.CursorKey() != "s" {
		t.Fatal("header must hide arcade years only")
	}
	choose("year", "", false)
	a.actPanel(platform.KeySpace)
	if len(a.view) != 2 {
		t.Fatal("unknown only failed")
	}
	choose("clear", "", false)
	a.actPanel(platform.KeyEnter)
	if a.filters.Active() || len(a.view) != 5 || saves < 8 {
		t.Fatal("clear/persistence hook failed")
	}
	a.SetFilters(data.Filters{YearOff: map[string]bool{"1980": true}, RotOff: map[string]bool{"h": true}})
	a.setSearch("Game")
	if got := a.facetCounts("year"); !reflect.DeepEqual(got, map[string]int{"1980": 1, "1990": 1, "": 1}) {
		t.Fatal("counts must ignore year selection, respect other filters/search", got)
	}
	a.cfg.FilterRotation = true
	a.SetRotation(gfx.RotLeft) // refilters on the orientation change
	if len(a.view) != 1 || a.CursorKey() != "c" {
		t.Fatal("year filter must combine with the strict rotation filter")
	}
}
