package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

func arcadeCatalogueApp() *App {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "s", Title: "Stable", Base: "Arcade", Src: "arc", Updated: "2026-09-10"},
		{K: "b", Title: "Beta", Base: "Arcade", Beta: true, Src: "arc", Updated: "2026-09-11"},
		{K: "c", Title: "Console", Base: "Console", Src: "sys", Updated: "2026-09-12"},
		{K: "p", Title: "Computer", Base: "Computer", Src: "sys", Updated: "2026-09-13"},
		{K: "o", Title: "Other", Base: "Other", Src: "sys", Updated: "2026-09-14"},
	}
	return New(Config{PhysW: 320, PhysH: 240, Favorites: map[string]bool{"s": true, "c": true}, RecentLaunches: []data.Recent{{K: "c", At: now.Format(time.RFC3339)}, {K: "s", At: now.Format(time.RFC3339)}}, Status: func(int) data.Status { return data.StatusCurrent }}, data.Ingest(rows, "", now), nil)
}
func TestArcadeCatalogueScope(t *testing.T) {
	a := arcadeCatalogueApp()
	if len(a.view) != 2 || a.total != 2 || a.filtersActive() || a.cardCounts()[data.StatusCurrent] != 2 {
		t.Fatal("default catalogue or counts")
	}
	for _, mode := range data.ViewOrder {
		a.SetSort(mode)
		for _, i := range a.view {
			if !a.ds.Rows[i].IsArcade() {
				t.Fatalf("system in %v", mode)
			}
		}
	}
	a.SetSort(data.SortAlphabetical)
	a.query = "Console"
	a.rebuild()
	if len(a.view) != 0 {
		t.Fatal("search exposed hidden core")
	}
	a.query = ""
	a.SetFilters(data.Filters{BaseOff: map[string]bool{"Console": true}})
	if a.filtersActive() || a.filterSectionActive("base") {
		t.Fatal("hidden type should not mark list or Type as filtered")
	}
	openExpandedFilters(a)
	a.panel.cursor = 0
	a.togglePanel()
	if a.cfg.ShowNonArcade || len(a.view) != 2 || !a.cfg.Favorites["c"] {
		t.Fatal("clear changed preference or favorite")
	}
	a.SetSort(data.SortFavorites)
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "show-non-arcade" {
			a.panel.cursor = i
		}
	}
	a.stepValue(1)
	if !a.ShowNonArcade() || len(a.view) != 2 || a.total != 5 || a.cardCounts()[data.StatusCurrent] != 5 {
		t.Fatal("opt-in failed to restore systems and favorites")
	}
	a.stepValue(-1)
	if len(a.view) != 1 || !a.cfg.Favorites["c"] {
		t.Fatal("favorite lost")
	}
}
func TestArcadeFlatTypeControls(t *testing.T) {
	a := arcadeCatalogueApp()
	a.SetFilters(data.Filters{BaseOff: map[string]bool{"Computer": true}})
	openExpandedFilters(a)
	choose := func(value string, key platform.Key) {
		t.Helper()
		for i, e := range a.panel.entries {
			if e.kind == "beta" && e.value == value {
				a.panel.cursor = i
				if a.selectedDecade() != "" {
					t.Fatal("flat Type exposes a nonexistent submenu")
				}
				a.actPanel(key)
				return
			}
		}
		t.Fatal("missing flat beta control")
	}
	for _, e := range a.panel.entries {
		if e.kind == "base" && !e.header {
			t.Fatal("non-arcade Type entry in default UI")
		}
	}
	if a.facetCounts("src")["sys"] != 0 || a.facetCounts("beta")["stable"] != 1 {
		t.Fatal("hidden systems counted")
	}
	choose("stable", platform.KeySpace)
	if len(a.view) != 1 || a.CursorKey() != "s" {
		t.Fatal("only stable")
	}
	choose("stable", platform.KeySpace)
	if len(a.view) != 2 || !a.filters.BaseOff["Computer"] || a.filters.BaseOff["Console"] {
		t.Fatal("only/all altered saved system selection")
	}
	choose("stable", platform.KeyEnter)
	choose("beta", platform.KeyEnter)
	if len(a.view) != 0 {
		t.Fatal("both off")
	}
	choose("beta", platform.KeyEnter)
	if len(a.view) != 1 || a.CursorKey() != "b" {
		t.Fatal("cannot recover from both off")
	}
}
func TestArcadeCatalogueNews(t *testing.T) {
	a := arcadeCatalogueApp()
	rows := append([]data.Row(nil), a.ds.Rows...)
	for i := range rows {
		if !rows[i].IsArcade() {
			rows[i].Updated = "2026-09-15"
		}
	}
	if a.CatalogueNews(a.ds, rows) != "" {
		t.Fatal("hidden update announced")
	}
	rows = append(rows, data.Row{K: "new", Title: "New game", Base: "Arcade"})
	if a.CatalogueNews(a.ds, rows) != "1 new: New game" {
		t.Fatal("visible change missing")
	}
	a.cfg.ShowNonArcade = true
	if a.CatalogueNews(a.ds, rows) != "1 new, 3 updated" {
		t.Fatal("opt-in news")
	}
}
func TestArcadeIntroRequiresFullHold(t *testing.T) {
	for _, frame := range []bool{false, true} {
		a := arcadeCatalogueApp()
		a.cfg.ArcadeIntro = true
		saves := 0
		a.cfg.SettingsChanged = func() { saves++ }
		now := time.Now()
		event := func(k platform.Key, pressed bool, at time.Time) {
			a.Handle(platform.Event{Key: k, Pressed: pressed, At: at})
		}
		tick := a.Tick
		if frame {
			tick = a.Frame
		}
		for _, k := range []platform.Key{platform.KeyStart, platform.KeyBack, platform.KeyMenu} {
			event(k, true, now)
			tick(now.Add(3 * time.Second))
			event(k, false, now.Add(3*time.Second))
		}
		if !a.ArcadeIntroPending() || saves != 0 {
			t.Fatal("another button dismissed notice")
		}
		event(platform.KeyEnter, true, now)
		next := a.NextTick()
		if !next.After(now) || !next.Before(now.Add(arcadeIntroHold)) {
			t.Fatal("hold has no progress tick")
		}
		tick(now.Add(time.Second))
		if a.holdBar() != a.lay.Status.Dx()/2 || !a.ArcadeIntroPending() {
			t.Fatal("hold progress missing or early dismissal")
		}
		event(platform.KeyEnter, true, now.Add(time.Second)) // repeated press must not reset timer
		tick(now.Add(arcadeIntroHold - time.Millisecond))
		if !a.ArcadeIntroPending() {
			t.Fatal("dismissed early")
		}
		event(platform.KeyEnter, false, now.Add(arcadeIntroHold-time.Millisecond))
		if !a.ArcadeIntroPending() || a.holdBar() != 0 || !a.nextArcadeIntroTick().IsZero() {
			t.Fatal("short hold did not reset")
		}
		now = now.Add(3 * time.Second)
		event(platform.KeyEnter, true, now)
		tick(now.Add(arcadeIntroHold))
		if a.ArcadeIntroPending() || saves != 1 || a.screen != ScreenList || a.holdBar() != 0 {
			t.Fatal("full hold failed")
		}
		event(platform.KeyEnter, true, now.Add(arcadeIntroHold+time.Millisecond))
		event(platform.KeyEnter, false, now.Add(arcadeIntroHold+2*time.Millisecond))
		if a.screen != ScreenList || len(a.down) != 0 {
			t.Fatal("held acknowledgement leaked into navigation")
		}
	}
}

func TestArcadeIntroReleaseAtDeadline(t *testing.T) {
	a := arcadeCatalogueApp()
	a.cfg.ArcadeIntro = true
	now := time.Now()
	a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: now})
	a.Handle(platform.Event{Key: platform.KeyEnter, At: now.Add(arcadeIntroHold)})
	if a.ArcadeIntroPending() || a.screen != ScreenList || len(a.down) != 0 {
		t.Fatal("deadline release did not complete cleanly")
	}
}

func TestArcadeLastLookIgnoresHiddenChanges(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{{K: "a", Title: "Arcade", Base: "Arcade", Updated: "2026-09-10"}, {K: "s", Title: "System", Base: "Computer", Updated: "2026-09-14"}}
	stored := &data.SeenRecord{T: now.Add(-24 * time.Hour).Format(time.RFC3339), Cur: map[string]string{"a": "2026-09-10", "s": "2026-09-10"}}
	a := New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true, Now: func() time.Time { return now }}, data.Ingest(rows, "", now), stored)
	if !a.topMark {
		t.Fatal("hidden update changed the last-look message")
	}
	a.SetFilters(data.Filters{Since: true})
	if len(a.view) != 0 {
		t.Fatal("hidden update appeared in Since last look")
	}
	a.cfg.ShowNonArcade = true
	a.Refilter()
	if len(a.view) != 1 || a.CursorKey() != "s" || a.topMark {
		t.Fatal("opt-in failed to reveal the system update")
	}
}
