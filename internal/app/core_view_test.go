package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func coreViewApp(cfg Config) (*App, func(platform.Key)) {
	clock := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "a", Title: "Alpha", Base: "Arcade", Core: "jtcps1", Year: "1989", Updated: "2026-01-03", Date: "2020-01-01"},
		{K: "b", Title: "Beta", Base: "Arcade", Core: "jtcps1", Year: "1991", Updated: "2026-01-02", Date: "2020-01-02"},
		{K: "c", Title: "Gamma", Base: "Arcade", Core: "galaga", Year: "1981", Updated: "2026-01-01", Date: "2020-01-03"},
		{K: "d", Title: "Delta", Base: "Console", Core: "SNES", Year: "1990", Updated: "2025-12-31", Date: "2020-01-04"},
	}
	cfg.ShowNonArcade, cfg.PhysW, cfg.PhysH = true, 320, 240
	cfg.Now = func() time.Time { return clock }
	a := New(cfg, data.Ingest(rows, "", clock), nil)
	tap := func(k platform.Key) {
		a.Handle(platform.Event{Key: k, Pressed: true, At: clock})
		a.Handle(platform.Event{Key: k, Pressed: false, At: clock})
	}
	return a, tap
}

// viewsRow opens Options and returns its Views row.
func viewsRow(t *testing.T, a *App, tap func(platform.Key)) panelEntry {
	t.Helper()
	tap(platform.KeyBack)
	expandOptionsForTest(a)
	for _, e := range a.panel.entries {
		if e.kind == "views" {
			return e
		}
	}
	t.Fatal("no Views row in Options")
	return panelEntry{}
}

func TestCoreView(t *testing.T) {
	a, tap := coreViewApp(Config{ViewsOff: []string{"recents"}})
	var seen []data.SortMode
	for i := 0; i < 7; i++ {
		seen = append(seen, a.mode)
		tap(platform.KeySpace)
	}
	want := []data.SortMode{data.SortUpdated, data.SortDebut, data.SortYear, data.SortAlphabetical, data.SortMaker, data.SortCore, data.SortFavorites}
	if !reflect.DeepEqual(seen, want) || a.mode != data.SortUpdated {
		t.Fatalf("Y walks %v then %v", seen, a.mode)
	}
	a.SetSort(data.SortCore)
	keys := func() []string {
		var out []string
		for _, i := range a.view {
			out = append(out, a.ds.Rows[i].K)
		}
		return out
	}
	// Capcom CPS-1: Alpha, Beta; Galaga's one game; the console core
	if got := keys(); !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("core order %v", got)
	}
	if !reflect.DeepEqual(a.marks, []int{0, 0, 2, 3}) {
		t.Fatalf("marks %v", a.marks)
	}
	if a.markText(1) != "Capcom CPS-1" || a.markText(2) != "Single-game cores" || a.markText(3) != "Console cores" {
		t.Fatalf("headers %q %q %q", a.markText(1), a.markText(2), a.markText(3))
	}
	// L/R jump cores, the row centered under its header, no notice
	centered := func(pos int) int { return centeredTop(a.screenLine(pos), a.totalLines(), a.lay.Lines) }
	a.cursor, a.top = 0, 0
	tap(platform.KeyPageDown)
	if a.cursor != 2 || a.top != centered(2) || a.shortPage || a.notice != "" {
		t.Fatalf("jump to the single-game cores: cursor %d top %d notice %q", a.cursor, a.top, a.notice)
	}
	tap(platform.KeyPageDown)
	tap(platform.KeyPageUp)
	if a.cursor != 2 {
		t.Fatalf("jump back: cursor %d", a.cursor)
	}
	a.Paint()
	// Options -> Views has the Core view after Manufacturer, on for everyone
	e := viewsRow(t, a, tap)
	if e.text != "Views (7 of 8)" || !strings.Contains(e.help, "manufacturer, core, Favorites") {
		t.Fatalf("Views row %q, help %q", e.text, e.help)
	}
	a.openViews()
	if len(a.panel.entries) != 8 || a.panel.entries[5].value != "core" || a.panel.entries[5].text != "Core" || !a.panel.entries[5].checked {
		t.Fatalf("Views page %+v", a.panel.entries)
	}
	// taking the current view out moves the list on to Favorites
	a.setViewOn(data.SortCore, false)
	if a.viewOn(data.SortCore) || a.mode != data.SortFavorites || !reflect.DeepEqual(a.ViewsOff(), []string{"core", "recents"}) {
		t.Fatalf("Core off: mode %v off %v", a.mode, a.ViewsOff())
	}
	// a remembered or default Core view opens in it
	if b, _ := coreViewApp(Config{RememberSort: true, LastSort: data.SortCore}); b.mode != data.SortCore {
		t.Fatalf("remembered Core view opens in %v", b.mode)
	}
	if b, _ := coreViewApp(Config{DefaultSort: data.SortCore}); b.mode != data.SortCore || b.DefaultView() != data.SortCore {
		t.Fatalf("default Core view opens in %v", b.mode)
	}
}
