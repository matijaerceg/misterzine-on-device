package app

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Options -> Sources: installed only drops every source whose Downloader
// database the card lacks from the list, search, Favorites, the card
// counts and the Filters panel, without reading as an active filter.
func TestInstalledSourcesLeaveEveryView(t *testing.T) {
	rows := []data.Row{
		{K: "m", Title: "MiSTer game", Base: "Arcade", Src: "distribution_mister"},
		{K: "j", Title: "Jotego game", Base: "Arcade", Src: "jtbindb"},
		{K: "c", Title: "Coin-Op game", Base: "Arcade", Src: "coinop"},
		{K: "x", Title: "Meathax game", Base: "Arcade", Src: "meathax"},
		{K: "r", Title: "rmCores game", Base: "Arcade", Src: "rmcores"},
		{K: "n", Title: "New source game", Base: "Arcade", Src: "newsource"},
	}
	changes := 0
	a := New(Config{PhysW: 320, PhysH: 240, InstalledOnly: true, Favorites: map[string]bool{"x": true, "m": true},
		Status:          func(i int) data.Status { return data.StatusNotFound },
		SettingsChanged: func() { changes++ }}, data.Ingest(rows, "", time.Now()), nil)
	keys := func() string {
		var ks []string
		for _, i := range a.view {
			ks = append(ks, a.ds.Rows[i].K)
		}
		sort.Strings(ks)
		return strings.Join(ks, " ")
	}
	if keys() != "c j m n r x" || a.total != 6 {
		t.Fatalf("nothing is hidden before the card scan reports: %q total %d", keys(), a.total)
	}
	a.SetHiddenSources(data.HiddenSources([]data.DB{{ID: "distribution_mister"}, {ID: "jtcores"}}), true)
	a.Refilter()
	if keys() != "j m n" || a.total != 3 {
		t.Fatalf("installed only: %q total %d", keys(), a.total)
	}
	if a.filtersActive() || a.filters.Active() {
		t.Fatal("the sources rule must not read as a filter")
	}
	if got := a.cardCounts(); got[data.StatusNotFound] != 3 {
		t.Fatalf("card counts include hidden sources: %v", got)
	}
	a.SetSort(data.SortFavorites)
	if keys() != "m" {
		t.Fatalf("favorites: %q", keys())
	}
	a.SetSort(data.SortAlphabetical)
	a.query = "meathax"
	a.rebuild()
	if len(a.view) != 0 {
		t.Fatalf("search reached a hidden source: %q", keys())
	}
	a.query = ""
	openExpandedFilters(a)
	var sources []string
	for _, e := range a.panel.entries {
		if e.kind == "src" && !e.header {
			sources = append(sources, e.value)
		}
	}
	if got := strings.Join(sources, " "); got != "distribution_mister jtbindb newsource" {
		t.Fatalf("Source section: %q", got)
	}
	a.screen = ScreenList
	a.openPanel(ScreenOptions)
	row := -1
	for i, e := range a.panel.entries {
		if e.kind == "sources" {
			row = i
		}
	}
	if row < 0 || a.panel.entries[row].idx != 1 || a.panel.entries[row].vals[1] != "installed only" {
		t.Fatal("Sources option missing or not on installed only")
	}
	a.panel.cursor = row
	a.stepValue(-1)
	if a.InstalledOnly() || changes != 3 || keys() != "c j m n r x" || a.total != 6 {
		t.Fatalf("all: %q total %d changes %d", keys(), a.total, changes)
	}
	a.stepValue(1)
	if !a.InstalledOnly() || keys() != "j m n" {
		t.Fatalf("installed again: %q", keys())
	}
	if help := a.panel.entries[row].help; strings.Contains(help, "No downloader.ini") {
		t.Fatalf("help reports a missing ini while one was read: %q", help)
	}
	a.SetHiddenSources(nil, false)
	a.Refilter()
	if keys() != "c j m n r x" {
		t.Fatalf("no downloader.ini must hide nothing: %q", keys())
	}
	if help := a.panel.entries[row].help; !strings.Contains(help, "No downloader.ini was found") {
		t.Fatalf("help must say why nothing is hidden: %q", help)
	}
}
