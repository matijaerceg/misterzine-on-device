package app

import (
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"reflect"
	"strings"
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

func TestLastLookLineFollowsDatesNotDepth(t *testing.T) {
	now := time.Now()
	rows := make([]data.Row, 300)
	base := map[string]string{}
	for i := range rows {
		rows[i] = data.Row{Base: "Arcade", K: fmt.Sprint(i), Title: fmt.Sprint(i), Updated: "2026-09-10"}
		base[rows[i].K] = rows[i].Updated
	}
	// a row catalogued late: unseen, but dated years before the last look
	rows[299].Updated = "2020-01-01"
	base[rows[299].K] = "2019-01-01"
	stored := &data.SeenRecord{T: now.Add(-2 * time.Hour).Format(time.RFC3339), Cur: base}
	a := New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), stored)
	if a.topMark || a.split != -1 || !reflect.DeepEqual(a.marks, []int{0}) {
		t.Fatalf("backfilled row: topMark=%v split=%d marks=%v", a.topMark, a.split, a.marks)
	}
	if got := a.markText(0); !strings.HasPrefix(got, "your last look") {
		t.Fatalf("a deep unseen row must not read as nothing new: %q", got)
	}
	a.SetFilters(data.Filters{Since: true})
	if len(a.view) != 1 {
		t.Fatal("Since filter must still scan full catalogue")
	}
	a.SetFilters(data.Filters{})

	// the same row, once seen: nothing new anywhere, and the line says so
	base[rows[299].K] = "2020-01-01"
	a = New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), stored)
	if !a.topMark || !reflect.DeepEqual(a.marks, []int{0}) {
		t.Fatalf("seen view: topMark=%v marks=%v", a.topMark, a.marks)
	}
	if got := a.markText(0); !strings.HasPrefix(got, "Nothing new since ") || !strings.HasSuffix(got, " ago") {
		t.Fatalf("top line = %q", got)
	}

	// two rows shipped since: the line sits after the lower one, and the
	// late-catalogued row deep below never drags it down
	rows[5].Updated = "2026-09-12"
	rows[7].Updated = "2026-09-11"
	base[rows[299].K] = "2019-01-01"
	a = New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), stored)
	if a.topMark || a.split != 1 || !reflect.DeepEqual(a.marks, []int{2}) {
		t.Fatalf("fresh rows: topMark=%v split=%d marks=%v", a.topMark, a.split, a.marks)
	}
	if got := a.markText(0); !strings.HasPrefix(got, "your last look") {
		t.Fatalf("line = %q", got)
	}
}
