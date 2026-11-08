package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func neoGeoViewData() *data.Dataset {
	return data.Ingest([]data.Row{
		{K: "local:neogeo:mslug:one", Title: "Metal Slug", Src: data.SrcLocal, Base: "Arcade", Core: "NeoGeo", Year: "1996"},
		{K: "local:other", Title: "Other local arcade", Src: data.SrcLocal, Base: "Arcade", Core: "Other"},
		{K: "local:neogeo:aof:two", Title: "Art of Fighting", Src: data.SrcLocal, Base: "Arcade", Core: "NeoGeo_LLAPI_20260101", Year: "1992"},
		{K: "console", Title: "Neo Geo console core", Src: "mister", Base: "Console", Core: "NeoGeo"},
	}, "", time.Now())
}

func TestNeoGeoViewOptInAndPersistence(t *testing.T) {
	ds := neoGeoViewData()
	cfg := Config{PhysW: 320, PhysH: 240, NeoGeoSupport: true}
	a := New(cfg, ds, nil)
	if a.NeoGeoView() {
		t.Fatal("Neo Geo must default off")
	}
	a.SetSort(data.SortNeoGeo)
	if a.Sort() == data.SortNeoGeo {
		t.Fatal("disabled view was selected")
	}
	a.openViews("chooser")
	found := false
	for i, e := range a.panel.entries {
		if e.kind == "view" && e.value == "neogeo" {
			a.panel.cursor = i
			a.actPanel(platform.KeyEnter)
			found = true
			break
		}
	}
	if !found || !a.NeoGeoView() {
		t.Fatal("editor did not enable Neo Geo")
	}
	a.SetSort(data.SortNeoGeo)
	a.openViewChooser()
	if a.viewChooser.entries[a.viewChooser.cursor].value != "neogeo" {
		t.Fatal("chooser lost current view")
	}
	cfg.NeoGeoView, cfg.ViewsOff = a.NeoGeoView(), a.ViewsOff()
	cfg.LastSort, cfg.RememberSort = a.Sort(), true
	b := New(cfg, ds, nil)
	if b.Sort() != data.SortNeoGeo || !b.NeoGeoView() {
		t.Fatal("restart lost view")
	}
	// It can be the only view, and the editor protects the last enabled view.
	for _, m := range data.ViewOrder {
		if m != data.SortNeoGeo {
			b.setViewOn(m, false)
		}
	}
	cfg.ViewsOff = b.ViewsOff()
	c := New(cfg, ds, nil)
	if c.viewsOnCount() != 1 || c.setViewOn(data.SortNeoGeo, false) {
		t.Fatal("only-view protection failed")
	}
	// A host without shortcut support never offers the Neo Geo view, including when loading its settings.
	cfg.NeoGeoSupport = false
	d := New(cfg, ds, nil)
	if d.Sort() == data.SortNeoGeo || d.viewsOnCount() == 0 {
		t.Fatal("unsupported view did not fall back")
	}
	for _, e := range d.viewsEntries() {
		if e.value == "neogeo" {
			t.Fatal("unsupported host offers Neo Geo view")
		}
	}
}

func TestNeoGeoViewFiltersCountsSearchAndEmptyState(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, NeoGeoSupport: true, NeoGeoView: true}, neoGeoViewData(), nil)
	a.SetSort(data.SortNeoGeo)
	want := []string{"local:neogeo:aof:two", "local:neogeo:mslug:one"}
	if !reflect.DeepEqual(viewKeys(a), want) || a.statusBar().view != "Neo Geo A-Z" || a.statusBar().shown != 2 {
		t.Fatalf("view %v, status %+v", viewKeys(a), a.statusBar())
	}
	if a.markText(1) != "A" || a.markText(2) != "M" {
		t.Fatal("missing letter headers")
	}
	if got := a.facetCounts("src"); !reflect.DeepEqual(got, map[string]int{data.SrcLocal: 2}) {
		t.Fatalf("counts leaked other cores: %v", got)
	}
	a.setSearch("slug")
	if !reflect.DeepEqual(viewKeys(a), want[1:]) {
		t.Fatal("search did not narrow view")
	}
	a.setSearch("")
	for _, f := range []data.Filters{
		{CoreOff: map[string]bool{"local/NeoGeo": true}},
		{SrcOff: map[string]bool{data.SrcLocal: true}},
	} {
		a.SetFilters(f)
		if len(a.view) != 0 || a.emptyListMessage() != "Neo Geo hidden" {
			t.Fatalf("hidden message: %q", a.emptyListMessage())
		}
		a.SetSort(data.SortAlphabetical)
		a.SetSort(data.SortNeoGeo)
		if len(a.view) != 0 {
			t.Fatal("choosing view undid filters")
		}
	}
	a.SetFilters(data.Filters{YearOff: map[string]bool{"1992": true}})
	if !reflect.DeepEqual(viewKeys(a), want[1:]) {
		t.Fatal("ordinary filter did not narrow view")
	}
	a.SetFilters(data.Filters{})
	a.SetData(data.Ingest(nil, "", time.Now()), nil)
	if a.emptyListMessage() != "No Neo Geo shortcuts" {
		t.Fatalf("no shortcuts: %q", a.emptyListMessage())
	}
}

func TestFilterOrderingLocalFirstAndUnknownLast(t *testing.T) {
	rows := []data.Row{}
	players := []string{"", "10", "2", "1", "2"}
	for i, src := range []string{"", "jtbindb", "distribution_mister", "local", "ongo"} {
		rows = append(rows, data.Row{K: itoa(i), Title: "Game " + itoa(i), Base: "Arcade", Src: src, Plr: players[i]})
	}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	var sources, playerChoices []string
	for _, e := range a.filterEntries() {
		if e.header {
			continue
		}
		if e.kind == "src" {
			sources = append(sources, e.value)
		}
		if e.kind == "plr" {
			playerChoices = append(playerChoices, e.value)
		}
	}
	if want := []string{"local", "jtbindb", "distribution_mister", "ongo", ""}; !reflect.DeepEqual(sources, want) {
		t.Fatalf("source order %v", sources)
	}
	if want := []string{"1", "2", "10", ""}; !reflect.DeepEqual(playerChoices, want) {
		t.Fatalf("player order %v", playerChoices)
	}
}
