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
	a.openPanel(ScreenFilter)
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
	if len(a.view) != 1 {
		t.Fatal("second Y toggled selection away")
	}
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
