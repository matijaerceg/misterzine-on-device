package data

import (
	"slices"
	"testing"
	"time"
)

// batchRows: three games one core build shipped together, a pair on
// another core, a same-game pair (one member on the batch's core and
// build) and a game alone.
func batchSet() *Dataset {
	rows := []Row{
		{K: "a", Title: "Alpha", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-06"},
		{K: "b", Title: "Bravo", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-06"},
		{K: "c", Title: "Charlie", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-06"},
		{K: "p1", Title: "Pair One", Base: "Arcade", Core: "pair", Updated: "2026-10-14", Date: "2026-01-01"},
		{K: "p2", Title: "Pair Two", Base: "Arcade", Core: "pair", Updated: "2026-10-14", Date: "2026-01-02"},
		{K: "s1", G: "same", Title: "Same (gnet)", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-06"},
		{K: "s2", G: "same", Title: "Same (other)", Base: "Arcade", Core: "other", Updated: "2026-09-01", Date: "2025-01-01"},
		{K: "solo", Title: "Solo", Base: "Arcade", Core: "solo", Updated: "2026-08-01", Date: "2025-06-01"},
	}
	return Ingest(rows, "h", time.Time{})
}

func memberKeys(ds *Dataset, g Group) []string {
	var ks []string
	for _, i := range g.Members {
		ks = append(ks, ds.Rows[i].K)
	}
	slices.Sort(ks)
	return ks
}

func TestBatchFoldsThreeGamesOnOneBuild(t *testing.T) {
	ds := batchSet()
	out, groups := ds.Fold(ds.Order(SortUpdated), SortUpdated, FoldOpts{Groups: true, Batches: true})
	var batch, same *Group
	for n := range groups {
		if groups[n].Batch() {
			batch = &groups[n]
		} else {
			same = &groups[n]
		}
	}
	if batch == nil || same == nil || len(groups) != 2 {
		t.Fatalf("groups %+v", groups)
	}
	// the same-game member on the batch's core and build stays in its group
	if got := memberKeys(ds, *batch); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("batch members %v", got)
	}
	if batch.G != BatchG("gnet", "2026-10-15") || batch.Key() != "b:gnet@2026-10-15" || batch.Day != "2026-10-15" || batch.Versions != 3 || batch.Total != 3 {
		t.Fatalf("batch %+v key %q", *batch, batch.Key())
	}
	if batch.Title(ds) != ds.Der[ds.Index("a")].CoreLabel {
		t.Fatalf("batch title %q", batch.Title(ds))
	}
	// the pair is too few: two plain rows
	if !slices.Contains(out, ds.Index("p1")) || !slices.Contains(out, ds.Index("p2")) {
		t.Fatalf("pair folded: %v", out)
	}
	// the batch heads where its first game sorted, closed
	if out[slices.Index(out, len(ds.Rows)+slices.IndexFunc(groups, func(g Group) bool { return g.Batch() }))+1] == batch.Members[0] {
		t.Fatal("closed batch shows its members")
	}
}

func TestBatchTakesALoneGroupMember(t *testing.T) {
	ds := batchSet()
	// the same-game partner filtered away: s1 is a plain row and joins the batch
	view := slices.DeleteFunc(slices.Clone(ds.Order(SortUpdated)), func(i int) bool { return ds.Rows[i].K == "s2" })
	_, groups := ds.Fold(view, SortUpdated, FoldOpts{Groups: true, Batches: true})
	if len(groups) != 1 || !slices.Equal(memberKeys(ds, groups[0]), []string{"a", "b", "c", "s1"}) {
		t.Fatalf("groups %+v", groups)
	}
	// with same-game groups off it joins as well
	_, groups = ds.Fold(ds.Order(SortUpdated), SortUpdated, FoldOpts{Batches: true})
	if len(groups) != 1 || len(groups[0].Members) != 4 {
		t.Fatalf("batches only: %+v", groups)
	}
}

func TestBatchUnderDebutKeysOnTheDebut(t *testing.T) {
	ds := batchSet()
	_, groups := ds.Fold(ds.Order(SortDebut), SortDebut, FoldOpts{Batches: true})
	if len(groups) != 1 || groups[0].G != BatchG("gnet", "2026-10-06") || groups[0].Day != "2026-10-06" {
		t.Fatalf("debut groups %+v", groups)
	}
	for _, m := range []SortMode{SortAlphabetical, SortYear, SortMaker, SortCore} {
		if _, groups := ds.Fold(ds.Order(m), m, FoldOpts{Batches: true}); groups != nil {
			t.Errorf("%v batched: %+v", m, groups)
		}
	}
}

func TestBatchKeepsItsHeaderOverASearchMatch(t *testing.T) {
	ds := batchSet()
	view := []int{ds.Index("b"), ds.Index("solo")}
	out, groups := ds.Fold(view, SortUpdated, FoldOpts{Groups: true, Batches: true, Search: true, Open: IsBatch})
	if len(groups) != 1 || groups[0].Versions != 1 || groups[0].Total != 3 {
		t.Fatalf("search groups %+v", groups)
	}
	if want := []int{len(ds.Rows), ds.Index("b"), ds.Index("solo")}; !slices.Equal(out, want) {
		t.Fatalf("search view %v, want %v", out, want)
	}
	// without a search one match is a plain row
	if _, groups := ds.Fold(view, SortUpdated, FoldOpts{Groups: true, Batches: true}); groups != nil {
		t.Fatalf("lone match folded: %+v", groups)
	}
	// a same-game member never joins a batch during a search
	if _, groups := ds.Fold([]int{ds.Index("s1")}, SortUpdated, FoldOpts{Groups: true, Batches: true, Search: true}); groups != nil {
		t.Fatalf("group member batched: %+v", groups)
	}
}

func TestFixtureCarriesBatches(t *testing.T) {
	ds, _ := loadFixture(t)
	for _, m := range []SortMode{SortUpdated, SortDebut} {
		_, groups := ds.Fold(ds.Order(m), m, FoldOpts{Groups: true, Batches: true})
		n := 0
		for _, g := range groups {
			if !g.Batch() {
				continue
			}
			n++
			if g.Versions < BatchMin || g.Title(ds) == "" || IsBatch(g.Core) {
				t.Errorf("%v batch %s: %d games, title %q", m, g.G, g.Versions, g.Title(ds))
			}
		}
		if n == 0 {
			t.Errorf("%v: the fixture folds no batch", m)
		}
	}
}
