package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func viewsApp(off ...string) (*App, func(platform.Key)) {
	clock := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "a", Title: "Alpha", Base: "Arcade", Manufacturer: "Sega", Year: "1985", Updated: "2026-01-03", Date: "2020-01-01"},
		{K: "b", Title: "Beta", Base: "Arcade", Manufacturer: "Capcom", Year: "1991", Updated: "2026-01-02", Date: "2020-01-02"},
		{K: "c", Title: "Gamma", Base: "Arcade", Manufacturer: "Sega Enterprises", Year: "1988", Updated: "2026-01-01", Date: "2020-01-03"},
		{K: "d", Title: "Delta", Base: "Console", Updated: "2025-12-31", Date: "2020-01-04"},
	}
	a := New(Config{PhysW: 320, PhysH: 240, ViewsOff: off, Now: func() time.Time { return clock }}, data.Ingest(rows, "", clock), nil)
	tap := func(k platform.Key) {
		a.Handle(platform.Event{Key: k, Pressed: true, At: clock})
		a.Handle(platform.Event{Key: k, Pressed: false, At: clock})
	}
	return a, tap
}

func TestViewsCycleSkipsTheOnesOff(t *testing.T) {
	a, tap := viewsApp("recents")
	var seen []data.SortMode
	for i := 0; i < 6; i++ {
		seen = append(seen, a.mode)
		tap(platform.KeySpace)
	}
	want := []data.SortMode{data.SortUpdated, data.SortDebut, data.SortYear, data.SortAlphabetical, data.SortMaker, data.SortFavorites}
	if !reflect.DeepEqual(seen, want) || a.mode != data.SortUpdated {
		t.Fatalf("cycle %v then %v", seen, a.mode)
	}
	a, tap = viewsApp("recents", "debut", "year", "favorites")
	seen = nil
	for i := 0; i < 3; i++ {
		seen = append(seen, a.mode)
		tap(platform.KeySpace)
	}
	if want := []data.SortMode{data.SortUpdated, data.SortAlphabetical, data.SortMaker}; !reflect.DeepEqual(seen, want) || a.mode != data.SortUpdated {
		t.Fatalf("cycle with views off %v then %v", seen, a.mode)
	}
	// a visit starts on the first view that is on when Updated is off
	a, _ = viewsApp("updated", "recents")
	if a.mode != data.SortDebut {
		t.Fatalf("first view %v", a.mode)
	}
	a.SetSort(data.SortUpdated)
	if a.mode != data.SortDebut {
		t.Fatal("a view that is off must be refused")
	}
	// leaving nothing on is ignored
	a, _ = viewsApp("updated", "debut", "year", "alphabetical", "maker", "favorites", "recents")
	if a.viewsOnCount() != 7 || a.mode != data.SortUpdated {
		t.Fatalf("every view off: %d on, mode %v", a.viewsOnCount(), a.mode)
	}
}

func TestViewsPageTogglesAndKeepsTheLastOne(t *testing.T) {
	a, tap := viewsApp("recents")
	tap(platform.KeyBack) // Options
	row := -1
	for i, e := range a.panel.entries {
		if e.kind == "views" {
			row = i
		}
	}
	if row < 0 || a.panel.entries[row].text != "Views (6 of 7)" {
		t.Fatalf("Options row: %+v", a.panel.entries)
	}
	a.panel.cursor = row
	tap(platform.KeyEnter)
	if a.screen != ScreenViews || len(a.panel.entries) != len(data.ViewOrder) {
		t.Fatalf("Views page: screen %v entries %d", a.screen, len(a.panel.entries))
	}
	if e := a.panel.entries[a.panel.cursor]; e.kind != "view" || e.value != "updated" || !e.checked {
		t.Fatalf("cursor on %+v", e)
	}
	// take Updated out while in it: the list moves on to Debut
	tap(platform.KeyEnter)
	if a.viewOn(data.SortUpdated) || a.mode != data.SortDebut || !reflect.DeepEqual(a.ViewsOff(), []string{"updated", "recents"}) {
		t.Fatalf("after Updated off: on %v mode %v off %v", a.viewOn(data.SortUpdated), a.mode, a.ViewsOff())
	}
	// switch everything else off: the last one on is greyed and stays
	for _, m := range []data.SortMode{data.SortDebut, data.SortYear, data.SortAlphabetical, data.SortMaker} {
		tap(platform.KeyDown)
		if e := a.panel.entries[a.panel.cursor]; e.value != m.Name() {
			t.Fatalf("cursor on %q, want %v", e.value, m)
		}
		tap(platform.KeyEnter)
	}
	tap(platform.KeyDown)
	if e := a.panel.entries[a.panel.cursor]; e.value != "favorites" || !e.checked || !e.disabled {
		t.Fatalf("last view on: %+v", e)
	}
	tap(platform.KeyEnter)
	if a.viewsOnCount() != 1 || !a.viewOn(data.SortFavorites) || a.mode != data.SortFavorites || a.notice != "Keep at least one view on" {
		t.Fatalf("the last view must stay on: %d on, mode %v, notice %q", a.viewsOnCount(), a.mode, a.notice)
	}
	tap(platform.KeySpace)
	if a.mode != data.SortFavorites {
		t.Fatal("Y with one view stays put")
	}
	// bring Recents back, then leave: Options on the Views row, then the list
	tap(platform.KeyDown)
	tap(platform.KeyEnter)
	if !a.viewOn(data.SortRecents) || a.panel.entries[a.panel.cursor].disabled {
		t.Fatal("Recents back on")
	}
	tap(platform.KeyBack)
	if a.screen != ScreenOptions || a.panel.entries[a.panel.cursor].kind != "views" || a.panel.entries[a.panel.cursor].text != "Views (2 of 7)" {
		t.Fatalf("back to Options: screen %v row %+v", a.screen, a.panel.entries[a.panel.cursor])
	}
	tap(platform.KeyBack)
	if a.screen != ScreenList {
		t.Fatalf("screen %v", a.screen)
	}
	// a remembered view that is off falls back to the first one on
	b := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortYear, ViewsOff: []string{"updated", "year"}}, a.ds, nil)
	if b.mode != data.SortDebut {
		t.Fatalf("remembered off view: %v", b.mode)
	}
}

func TestMakerHeadersInTheList(t *testing.T) {
	a, tap := viewsApp("recents")
	a.SetSort(data.SortMaker)
	keys := func() []string {
		var out []string
		for _, i := range a.view {
			out = append(out, a.ds.Rows[i].K)
		}
		return out
	}
	// Capcom: Beta; Sega: Alpha, Gamma; Unknown maker: Delta
	if got := keys(); !reflect.DeepEqual(got, []string{"b", "a", "c", "d"}) {
		t.Fatalf("maker order %v", got)
	}
	if !reflect.DeepEqual(a.marks, []int{0, 1, 3}) || a.totalLines() != 7 {
		t.Fatalf("marks %v, %d lines", a.marks, a.totalLines())
	}
	for pos, line := range []int{1, 3, 4, 6} {
		if a.screenLine(pos) != line {
			t.Fatalf("screenLine(%d) = %d, want %d", pos, a.screenLine(pos), line)
		}
	}
	if a.markText(0) != "Capcom" || a.markText(1) != "Sega" || a.markText(2) != "Unknown maker" {
		t.Fatalf("headers %q %q %q", a.markText(0), a.markText(1), a.markText(2))
	}
	// L/R jump makers, the row centered like a step (no top alignment), no notice
	centered := func(pos int) int { return centeredTop(a.screenLine(pos), a.totalLines(), a.lay.Lines) }
	a.cursor, a.top = 0, 0
	tap(platform.KeyPageDown)
	if a.cursor != 1 || a.top != centered(1) || a.shortPage || a.notice != "" {
		t.Fatalf("jump to Sega: cursor %d top %d notice %q", a.cursor, a.top, a.notice)
	}
	tap(platform.KeyPageDown)
	if a.cursor != 3 || a.top != centered(3) {
		t.Fatalf("jump to the unknown maker: cursor %d top %d", a.cursor, a.top)
	}
	tap(platform.KeyPageUp)
	if a.cursor != 1 || a.top != centered(1) {
		t.Fatalf("jump back: cursor %d top %d", a.cursor, a.top)
	}
	// the date column shows the year, the top bar names the order
	a.Paint()
	// a search narrows the groups with the rows
	a.setSearch("gam")
	if !reflect.DeepEqual(keys(), []string{"c"}) || !reflect.DeepEqual(a.marks, []int{0}) || a.markText(0) != "Sega" {
		t.Fatalf("search: %v marks %v", keys(), a.marks)
	}
	a.setSearch("")
	// the header scrolls into view with its first row when stepping up
	a.cursor = 3
	a.top = 6
	a.ensureVisible()
	if a.top > 5 {
		t.Fatalf("header hidden above the cursor: top %d", a.top)
	}
	a.Paint()
	// a long list: the landing row sits mid-screen with its header above it
	var many []data.Row
	for i := 0; i < 60; i++ {
		many = append(many, data.Row{K: "m" + itoa(i), Title: "Game " + itoa(i), Base: "Arcade", Manufacturer: []string{"Atari", "Konami", "Sega"}[i/20]})
	}
	a.SetData(data.Ingest(many, "", a.cfg.Now()), nil)
	a.cursor, a.top = 0, 0
	tap(platform.KeyPageDown)
	line := a.screenLine(a.cursor)
	if a.cursor != 20 || a.top != line-a.lay.Lines/2 || !a.markAt(a.cursor) {
		t.Fatalf("long jump: cursor %d line %d top %d lines %d", a.cursor, line, a.top, a.lay.Lines)
	}
	// the top line shows Atari's rows, so Atari is pinned there; Konami's
	// header is on screen below the top, so Konami is not pinned yet
	if a.pinnedHeader() != "Atari" {
		t.Fatalf("pinned %q at top %d", a.pinnedHeader(), a.top)
	}
	for a.top < a.screenLine(20)-1 {
		tap(platform.KeyDown)
	}
	if a.pinnedHeader() != "" { // Konami's own header is the top line
		t.Fatalf("pinned %q with the header on the top line (top %d)", a.pinnedHeader(), a.top)
	}
	tap(platform.KeyDown)
	if a.pinnedHeader() != "Konami" {
		t.Fatalf("pinned %q past Konami's header", a.pinnedHeader())
	}
	a.cursor, a.top = 0, 0
	if a.pinnedHeader() != "" { // the list's own first header is showing
		t.Fatal("pinned at the top of the list")
	}
	a.Paint()
}
