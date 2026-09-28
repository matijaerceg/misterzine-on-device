package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestFacetCountsRespectOtherFiltersAndSearch(t *testing.T) {
	rows := []data.Row{
		{K: "a", Title: "Game A", Base: "Arcade", Src: "one", Rot: "Vertical", Res: "15kHz"},
		{K: "b", Title: "Game B", Base: "Arcade", Src: "one", Rot: "Horizontal", Res: "31kHz"},
		{K: "c", Title: "Game C", Base: "Arcade", Src: "two", Rot: "Vertical", Res: "15kHz"},
		{K: "d", Title: "Game D", Base: "Arcade", Src: "one"},
		{K: "console", Title: "Console", Base: "Console", Src: "one"},
	}
	a := New(Config{ShowNonArcade: true, PhysW: 320, PhysH: 240, Favorites: map[string]bool{"a": true, "b": true}, Status: func(i int) data.Status {
		if i == 3 {
			return data.StatusNotFound
		}
		return data.StatusCurrent
	}}, data.Ingest(rows, "", time.Now()), nil)
	a.SetFilters(data.Filters{SrcOff: map[string]bool{"two": true}, RotOff: map[string]bool{"h": true}, Install: data.InstallFound})
	if got := a.facetCounts("rot"); !reflect.DeepEqual(got, map[string]int{"h": 1, "v": 1}) {
		t.Fatal(got)
	}
	if got := a.facetCounts("res"); !reflect.DeepEqual(got, map[string]int{"15kHz": 1}) {
		t.Fatal(got)
	}
	if got := a.facetCounts("src"); !reflect.DeepEqual(got, map[string]int{"one": 2, "two": 1}) {
		t.Fatal(got)
	}
	openExpandedFilters(a)
	foundZero := false
	for _, e := range a.panel.entries {
		if e.kind == "res" && e.value == "31kHz" && !e.header {
			foundZero = e.count == 0 && e.showCount
		}
	}
	if !foundZero {
		t.Fatal("zero-count choice disappeared")
	}
	a.setSearch("Game B")
	if got := a.facetCounts("rot"); !reflect.DeepEqual(got, map[string]int{"h": 1}) {
		t.Fatal(got)
	}
	a.setSearch("")
	a.SetFilters(data.Filters{FavOnly: true})
	if got := a.facetCounts("rot"); !reflect.DeepEqual(got, map[string]int{"h": 1, "v": 1}) {
		t.Fatal(got)
	}
	a.seen = data.InitSeen(&data.SeenRecord{Cur: map[string]string{"a": "", "b": "", "console": ""}}, rows, time.Now(), false)
	a.SetFilters(data.Filters{Since: true})
	if got := a.facetCounts("rot"); !reflect.DeepEqual(got, map[string]int{"v": 1, "": 1}) {
		t.Fatal(got)
	}
	a.SetFilters(data.Filters{BaseOff: map[string]bool{"Arcade": true}, RotOff: map[string]bool{"": true}})
	if len(a.view) != 1 || a.CursorKey() != "console" {
		t.Fatal("arcade choices filtered a system core")
	}
	for _, e := range a.filterEntries() {
		if e.kind == "rot" || e.kind == "res" || e.text == "Arcade game filters:" {
			t.Fatal("arcade section visible with Arcade excluded")
		}
	}
}

// The On the card and ROM check sections count within the other filters and
// the search, as the facet sections do: with beta games hidden, a beta game
// missing jtbeta.zip no longer counts as a ROM problem (reported on Discord).
func TestCardAndROMCountsFollowOtherFilters(t *testing.T) {
	rows := []data.Row{
		{K: "ok", Title: "Stable OK", Base: "Arcade", MRA: "_Arcade/ok.mra", Core: "ok"},
		{K: "bad", Title: "Stable Bad", Base: "Arcade", MRA: "_Arcade/bad.mra", Core: "bad"},
		{K: "beta", Title: "Beta Game", Base: "Arcade", MRA: "_Arcade/beta.mra", Core: "beta", Beta: true},
		{K: "gone", Title: "Stable Gone", Base: "Arcade", MRA: "_Arcade/gone.mra", Core: "gone"},
	}
	answers := map[string]string{"_Arcade/ok.mra": "", "_Arcade/bad.mra": "Missing game ROM: bad.zip", "_Arcade/beta.mra": "Missing game ROM: jtbeta.zip"}
	ds := data.Ingest(rows, "", time.Now())
	a := New(Config{PhysW: 320, PhysH: 240,
		Status: func(i int) data.Status {
			if ds.Rows[i].K == "gone" {
				return data.StatusNotFound
			}
			return data.StatusCurrent
		},
		ROMKnown: func(rel string) (string, bool, bool) {
			text, ok := answers[rel]
			return text, text != "", ok
		},
	}, ds, nil)
	// a count shows once the scan and the check have answered, zeros
	// included: an empty subset reads (0) rather than no number
	count := func(kind, value string) int {
		t.Helper()
		openExpandedFilters(a)
		for _, e := range a.panel.entries {
			if !e.header && e.kind == kind && e.value == value {
				if !e.showCount {
					t.Errorf("%s/%s: count hidden", kind, value)
				}
				return e.count
			}
		}
		t.Fatalf("no %s/%s entry", kind, value)
		return -1
	}
	check := func(when string, want map[[2]string]int) {
		t.Helper()
		for k, n := range want {
			if got := count(k[0], k[1]); got != n {
				t.Errorf("%s: %s/%s = %d, want %d", when, k[0], k[1], got, n)
			}
		}
	}
	check("no filters", map[[2]string]int{
		{"install", data.InstallAll}: 4, {"install", data.InstallFound}: 3, {"install", data.InstallMissing}: 1,
		{"rom", data.ROMAll}: 4, {"rom", data.ROMLaunch}: 2, {"rom", data.ROMNone}: 1,
	})
	a.SetFilters(data.Filters{BetaOff: map[string]bool{"beta": true}})
	check("beta off", map[[2]string]int{
		{"install", data.InstallAll}: 3, {"install", data.InstallFound}: 2,
		{"rom", data.ROMAll}: 3, {"rom", data.ROMLaunch}: 1, {"rom", data.ROMAny}: 1,
	})
	// a section's own choice does not narrow its counts, the other's does:
	// ROM launch leaves Stable Bad for On the card, not found leaves Stable
	// Gone (never checked) for ROM check
	a.SetFilters(data.Filters{BetaOff: map[string]bool{"beta": true}, Install: data.InstallMissing, ROM: data.ROMLaunch})
	check("own choices", map[[2]string]int{
		{"install", data.InstallAll}: 1, {"install", data.InstallFound}: 1, {"install", data.InstallMissing}: 0,
		{"rom", data.ROMAll}: 1, {"rom", data.ROMLaunch}: 0,
	})
	a.SetFilters(data.Filters{})
	a.setSearch("stable")
	check("search", map[[2]string]int{
		{"install", data.InstallAll}: 3, {"rom", data.ROMAll}: 3, {"rom", data.ROMLaunch}: 1,
	})
	a.setSearch("nothing like it")
	check("search matching nothing", map[[2]string]int{
		{"install", data.InstallAll}: 0, {"rom", data.ROMAll}: 0, {"rom", data.ROMLaunch}: 0,
	})
}

func TestAlphabeticalSortCyclePreservesSelectionAndFilters(t *testing.T) {
	rows := []data.Row{{Base: "Arcade", K: "z", Title: "Zulu", Updated: "2026-09-09", Date: "2026-09-07"}, {Base: "Arcade", K: "a", Title: "alpha", Updated: "2026-09-08", Date: "2026-09-09"}, {Base: "Arcade", K: "n", Title: "Game 10"}, {Base: "Arcade", K: "m", Title: "Game 2"}}
	a := New(Config{ShowNonArcade: true, PhysW: 320, PhysH: 240, ViewsOff: []string{"recents"}, Favorites: map[string]bool{"z": true, "a": true, "n": true, "m": true}}, data.Ingest(rows, "", time.Now()), &data.SeenRecord{Cur: map[string]string{}})
	a.SetFilters(data.Filters{FavOnly: true})
	a.MoveToKey("z")
	for _, mode := range []data.SortMode{data.SortDebut, data.SortYear, data.SortAlphabetical, data.SortMaker, data.SortCore, data.SortFavorites, data.SortUpdated} {
		a.actList(platform.KeySpace)
		if a.Sort() != mode || a.CursorKey() != "z" || !a.filters.FavOnly {
			t.Fatal("sort cycle changed selection or filters")
		}
		if mode == data.SortAlphabetical {
			var got []string
			for _, i := range a.view {
				got = append(got, a.ds.Rows[i].K)
			}
			if !reflect.DeepEqual(got, []string{"a", "m", "n", "z"}) {
				t.Fatal(got)
			}
			if !a.marker || a.marks[0] != 0 {
				t.Fatal("status row missing in alphabetical order")
			}
		}
	}
	a.setSearch("Game")
	a.actList(platform.KeySpace)
	a.actList(platform.KeySpace)
	a.actList(platform.KeySpace)
	if a.Sort() != data.SortAlphabetical || a.Search() != "Game" || len(a.view) != 2 {
		t.Fatal("sort lost search")
	}
}

func TestProvisionalDetailsExplainPresentValues(t *testing.T) {
	r := data.Row{Base: "Arcade", Rot: "Horizontal", Ctl: "4-way · 0 buttons", Prov: []string{"rot", "ctl"}}
	a := New(Config{ShowNonArcade: true, PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{r}, "", time.Now()), nil)
	var text string
	for _, line := range a.detailLines(&a.ds.Rows[0], &a.ds.Der[0], 0) {
		text += line.text + "\n"
	}
	if !strings.Contains(text, "Horizontal (provisional)") || !strings.Contains(text, "0 buttons (provisional)") {
		t.Fatal(text)
	}
}

func TestOpenFilterCountsFollowCardScan(t *testing.T) {
	status := data.StatusNotFound
	a := New(Config{ShowNonArcade: true, PhysW: 320, PhysH: 240, Status: func(int) data.Status { return status }}, data.Ingest([]data.Row{{K: "a", Base: "Arcade", Rot: "Horizontal"}}, "", time.Now()), nil)
	a.SetFilters(data.Filters{Install: data.InstallFound})
	openExpandedFilters(a)
	for i, e := range a.panel.entries {
		if e.kind == "rot" && e.value == "h" {
			a.panel.cursor = i
		}
	}
	if a.panel.entries[a.panel.cursor].count != 0 {
		t.Fatal("missing game counted as on card")
	}
	status = data.StatusCurrent
	a.Refilter()
	e := a.panel.entries[a.panel.cursor]
	if e.kind != "rot" || e.value != "h" || e.count != 1 {
		t.Fatalf("scan left stale count or moved selection: %+v", e)
	}
}
