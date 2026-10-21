package data

import (
	"slices"
	"testing"
	"time"
)

// sameGameRows are a pair on two cores, a game on one, and a standin for
// the first member's setname.
func sameGameRows() []Row {
	cat := []Row{
		{K: "bub-a", G: "bub", Title: "Bubble (Core A)", Base: "Arcade", Core: "corea", SN: "buba", Updated: "2026-05-01", Date: "2025-01-01", DateKind: "debut"},
		{K: "solo", Title: "Solo", Base: "Arcade", Core: "solo", SN: "solo", Updated: "2026-05-15", Date: "2025-06-01"},
		{K: "bub", G: "bub", Title: "Bubble (Core B)", Base: "Arcade", Core: "coreb", SN: "bub", Updated: "2026-06-01", Date: "2024-01-01", Img: "bub"},
	}
	local := []Row{{K: LocalKey("buba"), Title: "Bubble (card)", Base: "Arcade", Src: SrcLocal, SN: "buba", Core: "corec", Standin: true}}
	return MergeLocal(cat, local)
}

func sameGameSet() *Dataset { return Ingest(sameGameRows(), "h", time.Time{}) }

func TestFoldUmbrellaTakesFirstMembersPlace(t *testing.T) {
	ds := sameGameSet()
	view := ds.Order(SortUpdated) // bub, solo, bub-a, standin
	out, groups := ds.Fold(view, SortUpdated, FoldOpts{Groups: true})
	if len(groups) != 1 || groups[0].G != "bub" || groups[0].Versions != 2 || len(groups[0].Members) != 3 {
		t.Fatalf("groups %+v", groups)
	}
	if want := []int{len(ds.Rows), ds.Index("solo")}; !slices.Equal(out, want) {
		t.Fatalf("folded view %v, want %v", out, want)
	}
	if g := &groups[0]; g.Title(ds) != "Bubble" || g.Updated(ds) != "2026-06-01" || g.Date(ds) != "2024-01-01" || g.Preview(ds) != ds.Index("bub") {
		t.Fatalf("umbrella values %q %q %q %d", g.Title(ds), g.Updated(ds), g.Date(ds), g.Preview(ds))
	}
	// open, the members follow the umbrella in the order's own sequence
	out, _ = ds.Fold(view, SortUpdated, FoldOpts{Groups: true, Open: func(g string) bool { return g == "bub" }})
	want := []int{len(ds.Rows), ds.Index("bub"), ds.Index("bub-a"), ds.Index(LocalKey("buba")), ds.Index("solo")}
	if !slices.Equal(out, want) {
		t.Fatalf("open view %v, want %v", out, want)
	}
}

func TestFoldDebutPlacesUmbrellaAtFirstDebut(t *testing.T) {
	ds := sameGameSet()
	out, groups := ds.Fold(ds.Order(SortDebut), SortDebut, FoldOpts{Groups: true})
	if groups[0].Anchor != ds.Index("bub") {
		t.Fatalf("anchor %d, want the 2024 debut", groups[0].Anchor)
	}
	if want := []int{ds.Index("solo"), len(ds.Rows)}; !slices.Equal(out, want) {
		t.Fatalf("debut view %v, want %v", out, want)
	}
}

func TestFoldSkipsCoreFavoritesRecents(t *testing.T) {
	ds := sameGameSet()
	for _, m := range []SortMode{SortCore, SortFavorites, SortRecents} {
		view := ds.Order(m)
		if out, groups := ds.Fold(view, m, FoldOpts{Groups: true}); groups != nil || !slices.Equal(out, view) {
			t.Errorf("%v folded: %v", m, groups)
		}
	}
}

func TestFoldPromotesALoneMember(t *testing.T) {
	ds := sameGameSet()
	// a member and its own standin are one core: no umbrella
	view := []int{ds.Index("bub-a"), ds.Index(LocalKey("buba")), ds.Index("solo")}
	if out, groups := ds.Fold(view, SortAlphabetical, FoldOpts{Groups: true}); groups != nil || !slices.Equal(out, view) {
		t.Fatalf("lone member folded: %v %v", out, groups)
	}
}

func TestStandinFoldsWithItsCatalogueRow(t *testing.T) {
	ds := sameGameSet()
	if g := ds.GroupOf(ds.Index(LocalKey("buba"))); g != "bub" {
		t.Fatalf("standin group %q", g)
	}
	_, groups := ds.Fold(ds.Order(SortAlphabetical), SortAlphabetical, FoldOpts{Groups: true})
	if len(groups) != 1 || !slices.Contains(groups[0].Members, ds.Index(LocalKey("buba"))) {
		t.Fatalf("standin not folded: %+v", groups)
	}
}

func TestFixtureCarriesGroups(t *testing.T) {
	ds, _ := loadFixture(t)
	_, groups := ds.Fold(ds.Order(SortUpdated), SortUpdated, FoldOpts{Groups: true})
	if len(groups) < 4 {
		t.Fatalf("fixture folds into %d groups", len(groups))
	}
	for _, g := range groups {
		if g.Versions < 2 || g.Title(ds) == "" {
			t.Errorf("group %s: %d versions, title %q", g.G, g.Versions, g.Title(ds))
		}
	}
}
