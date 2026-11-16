package app

import (
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
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
	// Y on the heading turns the whole section back on, never off
	if a.canOnlyFilter() || !a.canShowAllFilter() {
		t.Fatal("a heading with choices off offers All on, not Only")
	}
	beforeHeading := saves
	if !a.actPanel(platform.KeySpace) || saves != beforeHeading+1 {
		t.Fatal("Y on the heading did not apply and persist")
	}
	if len(a.filters.GenreOff) != 0 || !a.filters.SrcOff["excluded"] || a.query != "Game" || len(a.view) != 2 {
		t.Fatal("All on changed other filters/search or left genres off")
	}
	if e := a.panel.entries[a.panel.cursor]; !e.header || e.kind != "genre" {
		t.Fatal("All on moved the selection off the heading")
	}
	if a.canShowAllFilter() || a.actPanel(platform.KeySpace) || strings.Contains(a.filterHint(), "Y ") {
		t.Fatal("Y acted on, or was offered on, a heading with everything on")
	}
}

func TestHeadingAllOnAcrossSections(t *testing.T) {
	rows := []data.Row{
		{K: "a", Base: "Arcade", Genre: "Shooter", Year: "1985", Rot: "h"},
		{K: "b", Base: "Arcade", Genre: "Puzzle", Year: "1992", Rot: "v"},
	}
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, inset := range []int{0, 15, 40} {
			for _, size := range [][2]int{{320, 240}, {360, 270}, {480, 270}} {
				a := New(Config{PhysW: size[0], PhysH: size[1], Rotation: rot, SafeInsetX: inset, SafeInsetY: inset}, data.Ingest(rows, "", time.Now()), nil)
				a.SetFilters(data.Filters{
					Install:  data.InstallFound,
					YearOff:  map[string]bool{"1985": true},
					RotOff:   map[string]bool{"v": true},
					GenreOff: map[string]bool{"Puzzle": true},
				})
				a.openPanel(ScreenFilter)
				heading := func(kind string) {
					t.Helper()
					for i, e := range a.panel.entries {
						if e.header && e.kind == kind {
							a.panel.cursor = i
							return
						}
					}
					t.Fatal("missing heading", kind)
				}
				for _, kind := range []string{"install", "year", "rot", "genre"} {
					heading(kind)
					hint := a.filterHint()
					if !strings.HasSuffix(hint, "Y All on") || a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
						t.Fatalf("%v inset %d %v %s heading hint %q", rot, inset, size, kind, hint)
					}
					closed := a.panel.sectionClosed[kind]
					a.actPanel(platform.KeySpace)
					if a.filterSectionActive(kind) || a.panel.sectionClosed[kind] != closed {
						t.Fatalf("%s: section still filtered or its expansion changed", kind)
					}
					if strings.Contains(a.filterHint(), "Y ") {
						t.Fatalf("%s: All on still offered once everything is on", kind)
					}
				}
				if a.filters.Active() {
					t.Fatalf("every section was reset but filters remain: %+v", a.filters)
				}
			}
		}
	}
}

func TestHeadingAllOnKeepsHiddenTypeChoices(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade"}, {K: "c", Base: "Console"},
	}, "", time.Now()), nil)
	a.SetFilters(data.Filters{BaseOff: map[string]bool{"Console": true}, BetaOff: map[string]bool{"beta": true}})
	openExpandedFilters(a)
	for i, e := range a.panel.entries {
		if e.header && e.kind == "base" {
			a.panel.cursor = i
		}
	}
	if !a.actPanel(platform.KeySpace) {
		t.Fatal("Y on Type with Beta off must turn it back on")
	}
	if len(a.filters.BetaOff) != 0 || !reflect.DeepEqual(a.filters.BaseOff, map[string]bool{"Console": true}) {
		t.Fatalf("Type reset lost a hidden non-arcade choice or kept Beta off: %+v", a.filters)
	}
}

func TestHeadingAllOnLeavesTheRotationRule(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, FilterRotation: true}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade", Rot: "h"}, {K: "b", Base: "Arcade", Rot: "v"},
	}, "", time.Now()), nil)
	openExpandedFilters(a)
	for i, e := range a.panel.entries {
		if e.header && e.kind == "rot" {
			a.panel.cursor = i
		}
	}
	before := a.filters
	if a.canShowAllFilter() || a.actPanel(platform.KeySpace) || a.filterHint() != "B Back" || !reflect.DeepEqual(a.filters, before) {
		t.Fatal("Y on the rotation heading must leave the Options rule alone")
	}
}

func TestOnlyAllLegendKeepsAllBeforeBack(t *testing.T) {
	rows := []data.Row{{K: "a", Base: "Arcade", Genre: "Shooter", Year: "1985"}, {K: "b", Base: "Arcade", Genre: "Puzzle", Year: "1992"}}
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft} {
		for _, inset := range []int{0, 15, 40} {
			a := New(Config{PhysW: 320, PhysH: 240, Rotation: rot, SafeInsetX: inset, SafeInsetY: inset}, data.Ingest(rows, "", time.Now()), nil)
			openExpandedFilters(a)
			for i, e := range a.panel.entries {
				if e.header || e.info || (e.kind != "genre" && e.kind != "decade" && e.kind != "year") {
					continue
				}
				a.panel.cursor = i
				hint := a.filterHint()
				if !a.hintFits(hint) {
					t.Fatalf("%v inset %d: %q does not fit", rot, inset, hint)
				}
				if !strings.Contains(hint, "Y Only/all") && strings.Contains(hint, "B Back") {
					t.Fatalf("%v inset %d: shortened Y Only/all while keeping B Back: %q", rot, inset, hint)
				}
				// a value row without a subgroup to open has room for the
				// whole Y legend everywhere once B Back goes
				if e := a.panel.entries[i]; e.kind == "genre" && !strings.Contains(hint, "Y Only/all") {
					t.Fatalf("%v inset %d: genre row lost Y Only/all: %q", rot, inset, hint)
				}
			}
		}
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
