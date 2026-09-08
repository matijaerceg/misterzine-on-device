package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

type golden struct {
	Rows       int               `json:"rows"`
	Updated    []string          `json:"updated"`
	Debut      []string          `json:"debut"`
	Title      []string          `json:"title"`
	CoreLabels map[string]string `json:"coreLabels"`
}

func loadFixture(t *testing.T) (*Dataset, golden) {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "testdata", "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := DecodeRows(f)
	if err != nil {
		t.Fatal(err)
	}
	gb, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sort_golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var g golden
	if err := json.Unmarshal(gb, &g); err != nil {
		t.Fatal(err)
	}
	if len(rows) != g.Rows {
		t.Fatalf("fixture has %d rows, golden %d; rerun tools/sort_golden.js", len(rows), g.Rows)
	}
	return Ingest(rows, "test", fixedNow), g
}

func keysOf(ds *Dataset, order []int) []string {
	out := make([]string, len(order))
	for i, idx := range order {
		out[i] = ds.Rows[idx].K
	}
	return out
}

// firstDiff reports the first position where two key lists differ, with context.
func firstDiff(t *testing.T, name string, got, want []string, ds *Dataset) {
	t.Helper()
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			lo := i - 3
			if lo < 0 {
				lo = 0
			}
			hi := i + 4
			if hi > len(want) {
				hi = len(want)
			}
			var w, g []string
			for j := lo; j < hi; j++ {
				w = append(w, describe(ds, want[j]))
				if j < len(got) {
					g = append(g, describe(ds, got[j]))
				}
			}
			t.Fatalf("%s: first difference at %d\n want: %s\n  got: %s", name, i, strings.Join(w, " | "), strings.Join(g, " | "))
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%s: length %d, want %d", name, len(got), len(want))
	}
}

func describe(ds *Dataset, k string) string {
	i := ds.Index(k)
	if i < 0 {
		return k + "?"
	}
	r := &ds.Rows[i]
	return k + "(" + r.Title + " u=" + r.Updated + " d=" + r.Date + " c=" + ds.Der[i].CoreLabel + ")"
}

func TestCoreLabelsMatchSite(t *testing.T) {
	ds, g := loadFixture(t)
	for core, want := range g.CoreLabels {
		if got := CoreLabel(core, ds.Sole); got != want {
			t.Errorf("coreLabel(%q) = %q, want %q", core, got, want)
		}
	}
}

func TestTitleCollationMatchesBrowser(t *testing.T) {
	ds, g := loadFixture(t)
	idx := make([]int, len(ds.Rows))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		return CompareKeys(ds.Der[idx[a]].titleKey, ds.Der[idx[b]].titleKey) < 0
	})
	firstDiff(t, "title order", keysOf(ds, idx), g.Title, ds)
}

func TestUpdatedOrderMatchesSite(t *testing.T) {
	ds, g := loadFixture(t)
	firstDiff(t, "updated order", keysOf(ds, ds.Order(SortUpdated)), g.Updated, ds)
}

func TestDebutOrderMatchesSite(t *testing.T) {
	ds, g := loadFixture(t)
	firstDiff(t, "debut order", keysOf(ds, ds.Order(SortDebut)), g.Debut, ds)
}

func TestCompareBasics(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"19xx", "1941", -1},      // numeric runs by value
		{"3DO", "280-ZZZAP", -1},  // 3 < 280
		{"Ace", "ASO", -1},        // case-insensitive
		{"Pokemon", "Pokémon", 0}, // accent-insensitive
		{"Galaga '88", "Galaga 3", -1},
		{"a b", "ab", -1}, // space before letters
		{"abc", "abcd", -1},
		{"x", "x", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestClusterN(t *testing.T) {
	rows := []Row{
		{K: "a", Core: "x", Updated: "2026-01-02", Date: "2020-01-01"},
		{K: "b", Core: "x", Updated: "2026-01-02", Date: "2020-01-01"},
		{K: "c", Core: "x", Updated: "2026-01-02", Date: "2026-01-02"}, // debut-floored: excluded
		{K: "d", Core: "y", Updated: "2026-01-02", Date: "2020-01-01"},
	}
	ds := Ingest(rows, "", time.Time{})
	want := []int{2, 2, 0, 1}
	for i, w := range want {
		if got := ds.ClusterN(i); got != w {
			t.Errorf("row %d clusterN = %d, want %d", i, got, w)
		}
	}
}

func TestRelAge(t *testing.T) {
	cases := map[string]string{
		"2026-09-08": "today",
		"2026-09-07": "yesterday",
		"2026-09-03": "5 days ago",
		"2026-08-09": "30 days ago",
		"2026-08-08": "1 month ago", // 31 days
		"2025-09-18": "1 year ago",  // 355 days: month band rounds to 12, hands over
		"2024-09-08": "2 years ago", // 731 days
		"2026-09-09": "today",       // future clamps to 0
		"":           "",
	}
	for iso, want := range cases {
		if got := RelAge(fixedNow, iso); got != want {
			t.Errorf("RelAge(%q) = %q, want %q", iso, got, want)
		}
	}
	if got := YearAge(fixedNow, "1981"); got != "45 years old" {
		t.Errorf("YearAge 1981 = %q", got)
	}
	if got := YearAge(fixedNow, "2025"); got != "1 year old" {
		t.Errorf("YearAge 2025 = %q", got)
	}
	if got := YearAge(fixedNow, "2026"); got != "" {
		t.Errorf("YearAge 2026 = %q", got)
	}
	if got := RelUpdated(fixedNow, fixedNow.Add(-90*time.Minute)); got != "2 hours ago" {
		t.Errorf("RelUpdated 90m = %q", got)
	}
	if got := RelUpdated(fixedNow, fixedNow.Add(-20*time.Second)); got != "just now" {
		t.Errorf("RelUpdated 20s = %q", got)
	}
	if got := RelUpdated(fixedNow, fixedNow.Add(-50*time.Hour)); got != "Sep 6, 2026" {
		t.Errorf("RelUpdated 50h = %q", got)
	}
}

func TestSeen(t *testing.T) {
	rows := []Row{{K: "a", Updated: "2026-09-01"}, {K: "b", Updated: "2026-09-02"}, {K: "c", Updated: "2026-09-03"}}
	ds := Ingest(rows, "", time.Time{})
	order := ds.Order(SortUpdated) // c, b, a

	first := InitSeen(nil, rows, fixedNow, true)
	if first.BaseRows != nil || first.MarkerOn(SortUpdated) {
		t.Fatal("first visit must have no baseline")
	}
	if first.SplitAt(ds, order, SortUpdated) != -1 {
		t.Fatal("no marker on the first visit")
	}

	// A quick return keeps the (absent) baseline.
	same := InitSeen(&first.State, rows, fixedNow.Add(10*time.Minute), true)
	if same.BaseRows != nil {
		t.Fatal("same-visit return must not promote a baseline")
	}

	// A real return promotes the previous snapshot; row b then updates.
	rows2 := []Row{{K: "a", Updated: "2026-09-01"}, {K: "b", Updated: "2026-09-09"}, {K: "c", Updated: "2026-09-03"}, {K: "n", Updated: "2026-09-10"}}
	ds2 := Ingest(rows2, "", time.Time{})
	later := fixedNow.Add(2 * 24 * time.Hour)
	next := InitSeen(&first.State, rows2, later, true)
	if next.BaseRows == nil || !next.MarkerOn(SortUpdated) {
		t.Fatal("second visit must carry a baseline")
	}
	if !next.Unseen(&rows2[1]) || !next.Unseen(&rows2[3]) || next.Unseen(&rows2[0]) {
		t.Fatal("unseen detection wrong")
	}
	order2 := ds2.Order(SortUpdated) // n, b, c, a
	if got := next.SplitAt(ds2, order2, SortUpdated); got != 1 {
		t.Fatalf("SplitAt = %d, want 1 (after b)", got)
	}
	if got := next.Label(later, true); got != "your last look, 2 days ago" {
		t.Fatalf("label = %q", got)
	}
	if next.SplitAt(ds2, ds2.Order(SortDebut), SortDebut) != -1 {
		t.Fatal("marker must be off under the debut sort")
	}
	// Untrusted clock: baseline kept, no age.
	nt := InitSeen(&next.State, rows2, time.Time{}, false)
	if nt.BaseRows == nil || nt.Label(time.Time{}, false) != "your last look" {
		t.Fatal("untrusted clock handling wrong")
	}
	// Empty snapshot reads as no baseline.
	empty := InitSeen(&SeenRecord{T: "2026-09-01T00:00Z", Cur: map[string]string{}}, rows, fixedNow, true)
	if empty.BaseRows != nil {
		t.Fatal("empty snapshot must read as no baseline")
	}
}

func TestDiffNews(t *testing.T) {
	old := Ingest([]Row{{K: "a", Title: "A", Updated: "1"}, {K: "b", Title: "B", Updated: "1"}, {K: "c", Title: "C", Updated: "1"}}, "", time.Time{})
	cases := []struct {
		rows []Row
		want string
	}{
		{[]Row{{K: "a", Title: "A", Updated: "1"}}, ""},
		{[]Row{{K: "a", Title: "A", Updated: "2"}, {K: "z", Title: "Z"}}, "1 new, 1 updated"},
		{[]Row{{K: "a", Title: "A", Updated: "2"}, {K: "b", Title: "B", Updated: "2"}}, "2 updated: A, B"},
		{[]Row{{K: "v", Title: "V"}, {K: "w", Title: "W"}, {K: "x", Title: "X"}, {K: "y", Title: "Y"}, {K: "z", Title: "Z"}}, "5 new: V, W, X +2 more"},
	}
	for _, c := range cases {
		if got := DiffNews(old, c.rows); got != c.want {
			t.Errorf("DiffNews = %q, want %q", got, c.want)
		}
	}
}

func TestLabelsAndASCII(t *testing.T) {
	if got := CoreLabel("jtcps2", nil); got != "Capcom CPS-2" {
		t.Errorf("jtcps2 = %q", got)
	}
	if got := CoreLabel("BubSysROMx", nil); got != "Bub Sys ROMx" {
		t.Errorf("camelCase = %q", got)
	}
	if got := CoreLabel("solo", map[string]string{"solo": "Solo Game"}); got != "Solo Game" {
		t.Errorf("sole = %q", got)
	}
	if got := TypeLabel(&Row{Base: "Console", Deprecated: true}); got != "Console core (deprecated)" {
		t.Errorf("TypeLabel = %q", got)
	}
	if got := ASCII("8-way · 3 buttons"); got != "8-way / 3 buttons" {
		t.Errorf("ASCII middot = %q", got)
	}
	if got := ASCII("Pokémon – Red"); got != "Pokemon - Red" {
		t.Errorf("ASCII fold = %q", got)
	}
	if got := SrcShort("jtbindb"); got != "Jotego" {
		t.Errorf("SrcShort = %q", got)
	}
}

func TestFilters(t *testing.T) {
	rows := []Row{
		{K: "a", Base: "Arcade", Src: "jtbindb", Rot: "Vertical (CCW)", Plr: "2", Genre: "Shooter"},
		{K: "b", Base: "Console", Src: "distribution_mister", Rot: "", Plr: "", Genre: ""},
	}
	ds := Ingest(rows, "", time.Time{})
	order := []int{0, 1}
	none := &Filters{}
	if got := Apply(ds, order, none, nil, nil); len(got) != 2 {
		t.Fatal("no filters must pass everything")
	}
	f := &Filters{RotOff: map[string]bool{"": true}}
	if got := Apply(ds, order, f, nil, nil); len(got) != 1 || got[0] != 0 {
		t.Fatalf("rot filter = %v", got)
	}
	f = &Filters{Install: InstallFound}
	st := func(i int) Status {
		if i == 1 {
			return StatusCurrent
		}
		return StatusNotFound
	}
	if got := Apply(ds, order, f, st, nil); len(got) != 1 || got[0] != 1 {
		t.Fatalf("install filter = %v", got)
	}
	f = &Filters{FavOnly: true}
	if got := Apply(ds, order, f, nil, func(k string) bool { return k == "a" }); len(got) != 1 || got[0] != 0 {
		t.Fatalf("fav filter = %v", got)
	}
}
