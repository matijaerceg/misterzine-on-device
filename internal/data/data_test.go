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

func TestAlphabeticalOrderMatchesBrowser(t *testing.T) {
	ds, g := loadFixture(t)
	firstDiff(t, "alphabetical order", keysOf(ds, ds.Order(SortAlphabetical)), g.Title, ds)
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

	first := InitSeen(nil, rows, fixedNow, true)
	if first.BaseRows != nil {
		t.Fatal("first visit must have no baseline")
	}
	if a, u := first.Since(ds, nil); a != 0 || u != 0 {
		t.Fatal("no counts on the first visit")
	}
	if got := first.Status(fixedNow, true, 0, 0); got[0] != "First visit. Next visit, changed rows get a green date." {
		t.Fatalf("first-visit status = %q", got)
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
	if next.BaseRows == nil {
		t.Fatal("second visit must carry a baseline")
	}
	if !next.Unseen(&rows2[1]) || !next.Unseen(&rows2[3]) || next.Unseen(&rows2[0]) {
		t.Fatal("unseen detection wrong")
	}
	order2 := ds2.Order(SortUpdated) // n, b, c, a
	if a, u := next.Since(ds2, nil); a != 1 || u != 1 {
		t.Fatalf("Since = %d added, %d updated; want 1, 1", a, u)
	}
	// A row catalogued late carries an old stamp: it counts as added like
	// any other, wherever it sorts.
	rows3 := append(rows2, Row{K: "old", Updated: "2020-01-01"})
	ds3 := Ingest(rows3, "", time.Time{})
	order3 := ds3.Order(SortUpdated) // n, b, c, a, old
	if a, u := next.Since(ds3, func(r *Row) bool { return r.K != "c" }); a != 2 || u != 1 {
		t.Fatalf("Since with a backfilled row = %d added, %d updated; want 2, 1", a, u)
	}
	if !next.AnyUnseen(ds3, order3) || next.AnyUnseen(ds2, order2[2:]) {
		t.Fatal("AnyUnseen must scan the whole order")
	}
	if got := next.Status(later, true, 2, 1); got[0] != "2 added, 1 updated since your last visit, 2 days ago" || got[1] != "2 added, 1 updated since visit, 2 days ago" || got[3] != "2 added, 1 updated since your last visit" || got[len(got)-1] != "2 added, 1 updated" {
		t.Fatalf("status = %q", got)
	}
	if got := next.Status(later, true, 0, 3); got[0] != "3 updated since your last visit, 2 days ago" {
		t.Fatalf("status = %q", got)
	}
	if got := next.Status(later, true, 0, 0); got[0] != "nothing added or updated since your last visit, 2 days ago" || got[1] != "nothing since your last visit, 2 days ago" || got[len(got)-1] != "nothing since visit" {
		t.Fatalf("quiet status = %q", got)
	}
	// Untrusted clock: baseline kept, no age.
	nt := InitSeen(&next.State, rows2, time.Time{}, false)
	if nt.BaseRows == nil || nt.Status(time.Time{}, false, 1, 0)[0] != "1 added since your last visit" {
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
	if got := Apply(ds, order, none, nil, nil, nil, nil); len(got) != 2 {
		t.Fatal("no filters must pass everything")
	}
	f := &Filters{RotOff: map[string]bool{"": true}}
	if got := Apply(ds, order, f, nil, nil, nil, nil); len(got) != 2 {
		t.Fatalf("rot filter = %v", got)
	}
	f = &Filters{Install: InstallFound}
	st := func(i int) Status {
		if i == 1 {
			return StatusCurrent
		}
		return StatusNotFound
	}
	if got := Apply(ds, order, f, st, nil, nil, nil); len(got) != 1 || got[0] != 1 {
		t.Fatalf("install filter = %v", got)
	}
	f = &Filters{Since: true}
	if got := Apply(ds, order, f, nil, nil, func(i int) bool { return i == 1 }, nil); len(got) != 1 || got[0] != 1 {
		t.Fatalf("since filter = %v", got)
	}
	f = &Filters{FavOnly: true}
	if got := Apply(ds, order, f, nil, func(k string) bool { return k == "a" }, nil, nil); len(got) != 1 || got[0] != 0 {
		t.Fatalf("fav filter = %v", got)
	}
}
