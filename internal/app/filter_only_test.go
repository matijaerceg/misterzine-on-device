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

func TestSinceVisitStatusCountsWholeCatalogue(t *testing.T) {
	now := time.Now()
	rows := make([]data.Row, 300)
	base := map[string]string{}
	for i := range rows {
		rows[i] = data.Row{Base: "Arcade", K: fmt.Sprint(i), Title: fmt.Sprint(i), Updated: "2026-09-10"}
		base[rows[i].K] = rows[i].Updated
	}
	// a row catalogued late: unseen, but dated years before the last visit
	rows[299].Updated = "2020-01-01"
	base[rows[299].K] = "2019-01-01"
	stored := &data.SeenRecord{T: now.Add(-2 * time.Hour).Format(time.RFC3339), Cur: base}
	a := New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), stored)
	if !a.marker || a.sinceAdded != 0 || a.sinceUpdated != 1 || !reflect.DeepEqual(a.marks, []int{0}) {
		t.Fatalf("backfilled row: added=%d updated=%d marks=%v", a.sinceAdded, a.sinceUpdated, a.marks)
	}
	if got := a.markText(0); !strings.HasPrefix(got, "1 updated") || !strings.HasSuffix(got, " ago") {
		t.Fatalf("a deep unseen row must count: %q", got)
	}
	a.SetFilters(data.Filters{Since: true})
	if len(a.view) != 1 {
		t.Fatal("Since filter must still scan full catalogue")
	}
	if got := a.markText(0); !strings.HasPrefix(got, "1 updated") {
		t.Fatalf("the status counts the catalogue, not the filtered view: %q", got)
	}
	a.SetFilters(data.Filters{})

	// the same row, once seen: nothing changed anywhere, and the line says so
	base[rows[299].K] = "2020-01-01"
	a = New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), stored)
	if a.sinceAdded != 0 || a.sinceUpdated != 0 || !reflect.DeepEqual(a.marks, []int{0}) {
		t.Fatalf("seen view: added=%d updated=%d marks=%v", a.sinceAdded, a.sinceUpdated, a.marks)
	}
	if got := a.markText(0); !strings.HasPrefix(got, "nothing added or updated") || !strings.HasSuffix(got, " ago") {
		t.Fatalf("top line = %q", got)
	}

	// two rows rebuilt and one new: the status row still heads the list,
	// there is no divider, and every order carries the same line
	rows[5].Updated = "2026-09-12"
	rows[7].Updated = "2026-09-11"
	rows = append(rows, data.Row{Base: "Arcade", K: "fresh", Title: "Fresh", Updated: "2026-09-12"})
	base[rows[299].K] = "2019-01-01"
	a = New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), stored)
	if a.sinceAdded != 1 || a.sinceUpdated != 3 || !reflect.DeepEqual(a.marks, []int{0}) {
		t.Fatalf("fresh rows: added=%d updated=%d marks=%v", a.sinceAdded, a.sinceUpdated, a.marks)
	}
	want := "1 added, 3 updated"
	// the full sentence with the age only fits a wide line; a narrow one
	// keeps the counts and the age
	if forms := a.seen.Status(now, true, 1, 3); forms[0] != "1 added, 3 updated since your last visit, 2 hours ago" || !strings.HasPrefix(a.markText(0), "1 added, 3 updated, ") {
		t.Fatalf("status forms %q, shown %q", forms, a.markText(0))
	}
	if got := a.markText(0); !strings.HasPrefix(got, want) {
		t.Fatalf("line = %q", got)
	}
	a.SetSort(data.SortDebut)
	if got := a.markText(0); !strings.HasPrefix(got, want) || !reflect.DeepEqual(a.marks, []int{0}) {
		t.Fatalf("debut order: %q marks %v", got, a.marks)
	}
	a.SetSort(data.SortAlphabetical)
	if got := a.markText(0); !strings.HasPrefix(got, want) || len(a.marks) < 2 || a.marks[0] != 0 || a.marks[1] != 0 {
		t.Fatalf("A-Z order: %q marks %v", got, a.marks)
	}
	if a.markText(1) != "0-9 and symbols" || a.screenLine(0) != 2 {
		t.Fatalf("A-Z headers follow the status row: %q line %d", a.markText(1), a.screenLine(0))
	}

	// first visit: the line explains the mark to come
	a = New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true}, data.Ingest(rows, "", now), nil)
	if !reflect.DeepEqual(a.marks, []int{0}) || !strings.HasPrefix(a.markText(0), "First visit") {
		t.Fatalf("first visit: marks %v %q", a.marks, a.markText(0))
	}
}
