package app

import (
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

func TestOnlyFilterKeepsOtherSectionsAndSearch(t *testing.T) {
	saves := 0
	rows := []data.Row{{K: "a", Title: "Game A", Base: "Arcade", Genre: "Shooter"}, {K: "b", Title: "Game B", Base: "Arcade", Genre: "Puzzle"}, {K: "c", Title: "Other", Base: "Arcade", Genre: "Shooter"}}
	a := New(Config{PhysW: 320, PhysH: 240, FiltersChanged: func() { saves++ }}, data.Ingest(rows, "", time.Now()), nil)
	a.SetFilters(data.Filters{SrcOff: map[string]bool{"excluded": true}, GenreOff: map[string]bool{"Shooter": true}})
	a.setSearch("Game")
	openExpandedFilters(a)
	for i, e := range a.panel.entries {
		if e.kind == "genre" && !e.header && e.value == "Shooter" {
			a.panel.cursor = i
			break
		}
	}
	if !a.canOnlyFilter() {
		t.Fatal("missing shortcut")
	}
	beforeFirst := saves
	a.actPanel(platform.KeySpace)
	if saves != beforeFirst+1 {
		t.Fatal("Y did not schedule filter persistence")
	}
	if a.filters.GenreOff["Shooter"] || !a.filters.GenreOff["Puzzle"] || !a.filters.SrcOff["excluded"] || a.query != "Game" || len(a.view) != 1 || a.CursorKey() != "a" {
		t.Fatal("isolation changed other filters/search or failed to include unchecked value")
	}
	a.actPanel(platform.KeySpace)
	if len(a.view) != 2 || len(a.filters.GenreOff) != 0 || !a.filters.SrcOff["excluded"] {
		t.Fatal("second Y did not enable entire section while preserving other filters")
	}
	a.actPanel(platform.KeySpace) // isolate again before testing A
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 0 {
		t.Fatal("A no longer toggles")
	}
	for i, e := range a.panel.entries {
		if e.kind == "genre" && e.header {
			a.panel.cursor = i
			break
		}
	}
	if a.canOnlyFilter() || a.actPanel(platform.KeySpace) {
		t.Fatal("Y acted on header")
	}
}

func TestOnlyToggleAfterRestartAndOtherSectionEdits(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade", Genre: "Shooter"}, {K: "b", Base: "Arcade", Genre: "Puzzle"},
	}, "", time.Now()), nil)
	choose := func(kind, value string) {
		t.Helper()
		for i, e := range a.panel.entries {
			if !e.header && e.kind == kind && e.value == value {
				a.panel.cursor = i
				return
			}
		}
		t.Fatal("missing choice", kind, value)
	}
	a.SetFilters(data.Filters{GenreOff: map[string]bool{"Puzzle": true}})
	openExpandedFilters(a)
	choose("genre", "Shooter")
	a.onlyFilter()
	if len(a.filters.GenreOff) != 0 {
		t.Fatal("saved isolation without history did not toggle back to all")
	}
	choose("genre", "Shooter")
	a.onlyFilter()
	// Editing another section must not make Y restore that other section.
	choose("install", data.InstallFound)
	a.togglePanel()
	choose("genre", "Shooter")
	a.onlyFilter()
	if a.filters.Install != data.InstallFound || len(a.filters.GenreOff) != 0 {
		t.Fatal("restoration changed unrelated filters")
	}
}

func TestNoChangesLabelHonorsBoundedWindow(t *testing.T) {
	now := time.Now()
	rows := make([]data.Row, data.SplitScan+1)
	base := map[string]string{}
	for i := range rows {
		rows[i] = data.Row{K: fmt.Sprint(i), Title: fmt.Sprint(i), Updated: "2026-09-10"}
		base[rows[i].K] = rows[i].Updated
	}
	rows[data.SplitScan].Updated = "2020-01-01"
	base[rows[data.SplitScan].K] = "2019-01-01"
	a := New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), &data.SeenRecord{T: now.Add(-2 * time.Hour).Format(time.RFC3339), Cur: base})
	if !a.topMark || a.noChangesLabel() != "No changes in top 200" {
		t.Fatal("unseen row beyond marker window misrepresented")
	}
	a.SetFilters(data.Filters{Since: true})
	if len(a.view) != 1 {
		t.Fatal("Since filter must still scan full catalogue")
	}
}
