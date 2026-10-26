package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// genreViewsApp has two shoot 'em ups (one on two cores), a fighting game,
// a beat 'em up and a game of no genre; the genre views given are on.
func genreViewsApp(genresOn ...string) (*App, func(platform.Key)) {
	clock := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "zeta", Title: "Zeta Wing", Base: "Arcade", Genre: "Shoot 'em up", Updated: "2026-01-05", Date: "2020-01-01"},
		{K: "sf", Title: "Street Brawl", Base: "Arcade", Genre: "Fighting", Updated: "2026-01-04", Date: "2020-01-02"},
		{K: "aero-a", G: "aero", Title: "Aero Blaster (Core A)", Base: "Arcade", Core: "corea", Genre: "Shoot 'em up", Updated: "2026-01-03", Date: "2020-01-03"},
		{K: "aero", G: "aero", Title: "Aero Blaster (Core B)", Base: "Arcade", Core: "coreb", Genre: "Shoot 'em up", Updated: "2026-01-02", Date: "2020-01-04"},
		{K: "ff", Title: "Final Brawl", Base: "Arcade", Genre: "Beat 'em up", Updated: "2026-01-01", Date: "2020-01-05"},
		{K: "none", Title: "Mystery", Base: "Arcade", Updated: "2025-12-31", Date: "2020-01-06"},
	}
	a := New(Config{PhysW: 320, PhysH: 240, ShowBetaFeatures: true, ClockTrusted: true, ViewsOff: []string{"recents"}, GenreViewsOn: genresOn, Now: func() time.Time { return clock }}, data.Ingest(rows, "h", clock), nil)
	tap := func(k platform.Key) {
		a.Handle(platform.Event{Key: k, Pressed: true, At: clock})
		a.Handle(platform.Event{Key: k, Pressed: false, At: clock})
	}
	return a, tap
}

// The genre views join the Y cycle after Favorites and Recents, in their
// own order, only when on; a fresh install has them all off.
func TestGenreViewsJoinTheCycleOnlyWhenOn(t *testing.T) {
	cycle := func(a *App, tap func(platform.Key), n int) []data.SortMode {
		var seen []data.SortMode
		for i := 0; i < n; i++ {
			seen = append(seen, a.mode)
			chooseNextView(a)
		}
		return seen
	}
	a, tap := genreViewsApp()
	want := []data.SortMode{data.SortUpdated, data.SortDebut, data.SortYear, data.SortAlphabetical, data.SortMaker, data.SortCore, data.SortFavorites}
	if got := cycle(a, tap, 7); !reflect.DeepEqual(got, want) || a.mode != data.SortUpdated || len(a.GenreViewsOn()) != 0 {
		t.Fatalf("fresh cycle %v then %v, genre views on %v", got, a.mode, a.GenreViewsOn())
	}
	a.SetSort(data.SortShmups)
	if a.mode != data.SortUpdated {
		t.Fatal("a genre view that is off must be refused")
	}
	a, tap = genreViewsApp("runnguns", "shmups")
	a.setViewOn(data.SortRecents, true)
	want = append(want, data.SortRecents, data.SortShmups, data.SortRunNGuns)
	if got := cycle(a, tap, 10); !reflect.DeepEqual(got, want) || a.mode != data.SortUpdated {
		t.Fatalf("cycle with two genre views on %v then %v", got, a.mode)
	}
	if !reflect.DeepEqual(a.GenreViewsOn(), []string{"shmups", "runnguns"}) || !reflect.DeepEqual(a.ViewsOff(), []string{}) {
		t.Fatalf("persisted: genre on %v, off %v", a.GenreViewsOn(), a.ViewsOff())
	}
	// an unknown or non-genre name in the saved list is ignored
	a, _ = genreViewsApp("bogus", "alphabetical", "fighting")
	if !reflect.DeepEqual(a.GenreViewsOn(), []string{"fighting"}) || !a.viewOn(data.SortAlphabetical) {
		t.Fatalf("parsed genre views on %v", a.GenreViewsOn())
	}
	// Views off and the genre views off together leave nothing: the
	// choice is ignored, the genre views staying off
	b := New(Config{PhysW: 320, PhysH: 240, ViewsOff: []string{"updated", "debut", "year", "alphabetical", "maker", "core", "favorites", "recents"}}, a.ds, nil)
	if b.viewsOnCount() != 8 || len(b.GenreViewsOn()) != 0 || b.mode != data.SortUpdated {
		t.Fatalf("nothing on: %d on, genre %v, mode %v", b.viewsOnCount(), b.GenreViewsOn(), b.mode)
	}
	// only a genre view on is a valid choice
	c := New(Config{PhysW: 320, PhysH: 240, ViewsOff: []string{"updated", "debut", "year", "alphabetical", "maker", "core", "favorites", "recents"}, GenreViewsOn: []string{"fighting"}}, a.ds, nil)
	if c.viewsOnCount() != 1 || c.mode != data.SortFighting {
		t.Fatalf("only Fighting on: %d on, mode %v", c.viewsOnCount(), c.mode)
	}
}

// A genre view shows the rows of its genre alone, A-Z with letter headers
// and jumps like the A-Z view, the same-game umbrellas included; search
// and filters narrow it further.
func TestGenreViewListsItsGenreAZ(t *testing.T) {
	a, tap := genreViewsApp("shmups", "fighting", "beatemups", "runnguns")
	a.SetSort(data.SortShmups)
	if got := strings.Join(viewKeys(a), " "); got != "g:aero zeta" {
		t.Fatalf("shoot 'em ups %q", got)
	}
	if a.shown != 3 || a.statusBar().view != "Shoot 'em ups A-Z" || !a.statusBar().narrow || a.statusBar().shown != 3 {
		t.Fatalf("status %+v shown %d", a.statusBar(), a.shown)
	}
	// the status line, then a header per letter
	if !a.groupHeaders() || !reflect.DeepEqual(a.marks, []int{0, 0, 1}) || a.markText(1) != "A" || a.markText(2) != "Z" {
		t.Fatalf("letter headers: %v marks %v %q %q", a.groupHeaders(), a.marks, a.markText(1), a.markText(2))
	}
	// the umbrella opens like under A-Z, and L/R jump letters
	a.cursor = 0
	a.act(platform.KeyEnter)
	if got := strings.Join(viewKeys(a), " "); got != "g:aero aero-a aero zeta" {
		t.Fatalf("open umbrella %q", got)
	}
	tap(platform.KeyPageDown)
	if a.CursorKey() != "zeta" || a.notice != "" {
		t.Fatalf("letter jump: cursor %q notice %q", a.CursorKey(), a.notice)
	}
	tap(platform.KeyPageUp)
	if a.CursorKey() != "g:aero" {
		t.Fatalf("letter jump back: cursor %q", a.CursorKey())
	}
	// the A-Z view keeps every genre, in the same order
	a.SetSort(data.SortAlphabetical)
	if got := strings.Join(viewKeys(a), " "); got != "g:aero aero-a aero ff none sf zeta" {
		t.Fatalf("A-Z %q", got)
	}
	a.SetSort(data.SortFighting)
	if got := strings.Join(viewKeys(a), " "); got != "sf" || a.statusBar().view != "Fighting A-Z" {
		t.Fatalf("fighting %q, status %q", got, a.statusBar().view)
	}
	a.SetSort(data.SortBeatEmUps)
	if got := strings.Join(viewKeys(a), " "); got != "ff" || a.statusBar().view != "Beat 'em ups A-Z" {
		t.Fatalf("beat 'em ups %q, status %q", got, a.statusBar().view)
	}
	a.SetSort(data.SortRunNGuns)
	if len(a.view) != 0 || a.statusBar().view != "Run 'n guns A-Z" || a.emptyListMessage() != "No Run 'n guns listed" {
		t.Fatalf("run 'n guns %v, status %q, empty %q", viewKeys(a), a.statusBar().view, a.emptyListMessage())
	}
	// search and filters apply on top of the genre
	a.SetSort(data.SortShmups)
	a.setSearch("zeta")
	if got := strings.Join(viewKeys(a), " "); got != "zeta" || a.shown != 1 {
		t.Fatalf("search %q shown %d", got, a.shown)
	}
	a.setSearch("")
	a.SetFilters(data.Filters{SrcOff: map[string]bool{"": true}})
	if len(a.view) != 0 || a.emptyListMessage() == "No Shoot 'em ups listed" {
		t.Fatalf("filtered %v, empty %q", viewKeys(a), a.emptyListMessage())
	}
	a.SetFilters(data.Filters{})
	// the Genre filter counts within the view
	if counts := a.facetCounts("plr"); counts[""] != 3 {
		t.Fatalf("facet counts in the view %v", counts)
	}
}

// genreViewLabels is the Genre views page top to bottom, which is also the
// order Y walks the views on after Recents.
var genreViewLabels = []string{"Shoot 'em ups", "Fighting", "Beat 'em ups", "Run 'n guns", "Platformers", "Arena shooters", "Fixed shooters", "Rail shooters",
	"Light gun", "Racing", "Maze", "Puzzle", "Sports", "Tabletop & quiz", "Other"}

// Every catalogue genre has a view: its settings name, the genre label it
// matches exactly, its page label and its top-bar name, and the view lists
// that genre alone.
func TestEveryGenreHasAView(t *testing.T) {
	table := []struct{ name, genre, label string }{
		{"shmups", "Shoot 'em up", "Shoot 'em ups"}, {"fighting", "Fighting", "Fighting"}, {"beatemups", "Beat 'em up", "Beat 'em ups"}, {"runnguns", "Run 'n gun", "Run 'n guns"},
		{"platformers", "Platformer", "Platformers"}, {"arenashooters", "Arena shooter", "Arena shooters"}, {"fixedshooters", "Fixed shooter", "Fixed shooters"}, {"railshooters", "Rail shooter", "Rail shooters"},
		{"lightgun", "Light gun", "Light gun"}, {"racing", "Racing", "Racing"}, {"maze", "Maze", "Maze"}, {"puzzle", "Puzzle", "Puzzle"}, {"sports", "Sports", "Sports"},
		{"tabletopquiz", "Tabletop & quiz", "Tabletop & quiz"}, {"other", "Other", "Other"},
	}
	if len(data.GenreViews) != len(table) || len(genreViewLabels) != len(table) {
		t.Fatalf("%d genre views, want %d", len(data.GenreViews), len(table))
	}
	clock := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	var rows []data.Row
	var names []string
	for i, tc := range table {
		rows = append(rows, data.Row{K: tc.name, Title: string(rune('A'+i)) + " " + tc.genre, Base: "Arcade", Genre: tc.genre, Updated: "2026-01-01", Date: "2020-01-01"})
		names = append(names, tc.name)
	}
	rows = append(rows, data.Row{K: "none", Title: "No genre", Base: "Arcade", Updated: "2026-01-01", Date: "2020-01-01"})
	a := New(Config{PhysW: 320, PhysH: 240, ViewsOff: []string{"recents"}, GenreViewsOn: names, Now: func() time.Time { return clock }}, data.Ingest(rows, "h", clock), nil)
	for i, tc := range table {
		m := data.GenreViews[i]
		if m.Name() != tc.name || m.Genre() != tc.genre || m.String() != tc.label || viewLabels[m] != tc.label || genreViewLabels[i] != tc.label || !m.Alphabetical() || !m.Valid() {
			t.Errorf("%d: name %q genre %q label %q (page %q)", i, m.Name(), m.Genre(), m.String(), viewLabels[m])
		}
		if back, ok := data.ParseSort(tc.name); !ok || back != m {
			t.Errorf("%q parses to %v %v", tc.name, back, ok)
		}
		a.SetSort(m)
		if got := strings.Join(viewKeys(a), " "); got != tc.name || a.statusBar().view != tc.label+" A-Z" || statusViewsShort[m] != tc.label+" A-Z" {
			t.Errorf("%s view lists %q, top bar %q", tc.name, got, a.statusBar().view)
		}
	}
	// the Y cycle walks them in page order after Favorites
	a.SetSort(data.SortFavorites)
	for _, m := range data.GenreViews {
		chooseNextView(a)
		if a.mode != m {
			t.Fatalf("Y reached %v, want %v", a.mode, m)
		}
	}
	// the first four keep their saved values; the rest follow in order
	if int(data.SortShmups) != 8 || int(data.SortRunNGuns) != 11 || int(data.SortPlatformers) != 12 || int(data.SortOther) != 22 || data.SortMode(23).Valid() {
		t.Fatalf("values: shmups %d runnguns %d platformers %d other %d", data.SortShmups, data.SortRunNGuns, data.SortPlatformers, data.SortOther)
	}
}

// Both families of views share one editor and keep their existing saved names.
func TestGenreViewsPage(t *testing.T) {
	a, tap := genreViewsApp()
	tap(platform.KeyBack)
	expandOptionsForTest(a)
	row := -1
	for i, e := range a.panel.entries {
		if e.kind == "genre-views" {
			t.Fatal("separate genre editor remains")
		}
		if e.kind == "views" {
			row = i
		}
	}
	if row < 0 || a.panel.entries[row].text != "Available views (7 of 23)" {
		t.Fatal("missing consolidated entry")
	}
	a.panel.cursor = row
	tap(platform.KeyEnter)
	if a.screen != ScreenViews || a.viewsTitle() != "Available views" || len(a.panel.entries) != 25 {
		t.Fatal("missing combined editor")
	}
	for i, want := range genreViewLabels {
		if e := a.panel.entries[i+10]; e.text != want || e.checked {
			t.Fatalf("genre %d: %+v", i, e)
		}
	}
	a.panel.cursor = 10
	changed := 0
	a.cfg.SettingsChanged = func() { changed++ }
	tap(platform.KeyEnter)
	tap(platform.KeyDown)
	tap(platform.KeyDown)
	tap(platform.KeyEnter)
	if !reflect.DeepEqual(a.GenreViewsOn(), []string{"shmups", "beatemups"}) || changed != 2 {
		t.Fatal("genre choices were not saved")
	}
	tap(platform.KeyBack)
	if a.screen != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "views" || a.panel.entries[a.panel.cursor].text != "Available views (9 of 23)" {
		t.Fatal("return to Options")
	}
	b := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortBeatEmUps, ViewsOff: a.ViewsOff(), GenreViewsOn: a.GenreViewsOn()}, a.ds, nil)
	if b.mode != data.SortBeatEmUps || !b.viewOn(data.SortShmups) || b.viewOn(data.SortFighting) {
		t.Fatal("saved settings did not round trip")
	}
	a.SetSort(data.SortShmups)
	a.openViews("views")
	a.panel.cursor = 10
	tap(platform.KeyEnter)
	if a.mode != data.SortBeatEmUps || a.viewOn(data.SortShmups) {
		t.Fatal("disabling current genre did not advance")
	}
	for _, m := range a.viewsPageOrder() {
		a.setViewOn(m, false)
	}
	a.buildPanel()
	a.panel.cursor = 12
	if !a.panel.entries[12].disabled {
		t.Fatal("last genre view is not protected")
	}
	tap(platform.KeyEnter)
	if a.viewsOnCount() != 1 || a.notice != "Keep at least one view on" {
		t.Fatal("last view disabled")
	}
}
