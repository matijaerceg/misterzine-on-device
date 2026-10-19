package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// sourceCoreRows: Ongo ships Seta (two games, a third under ppriest) and
// BallySente and Cave, one game each; ppriest ships only Seta; Jotego only
// its single-game jtcps1.
func sourceCoreRows() []data.Row {
	return []data.Row{
		{K: "a1", Title: "Alpha One", Base: "Arcade", Src: "ongo", Core: "Seta"},
		{K: "a2", Title: "Alpha Two", Base: "Arcade", Src: "ongo", Core: "Seta"},
		{K: "b", Title: "Bravo", Base: "Arcade", Src: "ongo", Core: "BallySente"},
		{K: "c", Title: "Charlie", Base: "Arcade", Src: "ongo", Core: "Cave"},
		{K: "p", Title: "Papa", Base: "Arcade", Src: "ppriest", Core: "Seta"},
		{K: "j", Title: "Juliet", Base: "Arcade", Src: "jtbindb", Core: "jtcps1"},
	}
}

func TestSourceOpensIntoCores(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(sourceCoreRows(), "", time.Now()), nil)
	openExpandedFilters(a)
	find := func(kind, value string) int {
		t.Helper()
		for i, e := range a.panel.entries {
			if e.kind == kind && e.value == value && !e.header {
				return i
			}
		}
		t.Fatalf("missing %s/%s", kind, value)
		return -1
	}
	ongo := a.panel.entries[find("src", "ongo")]
	if !ongo.checked || ongo.partial || ongo.count != 4 || ongo.text != gfx.ArrowRight+" Ongo" {
		t.Fatalf("Ongo starts closed and fully on: %+v", ongo)
	}
	// one core, or only single-game cores: nothing to open
	for _, src := range []string{"ppriest", "jtbindb"} {
		a.panel.cursor = find("src", src)
		if e := a.panel.entries[a.panel.cursor]; e.text != data.SrcShort(src) || a.selectedDecade() != "" {
			t.Fatalf("%s must not open: %+v", src, e)
		}
	}
	for _, e := range a.panel.entries {
		if e.kind == "core" || e.kind == "singles" {
			t.Fatal("cores listed while closed")
		}
	}
	a.panel.cursor = find("src", "ongo")
	a.actPanel(platform.KeyRight)
	singles, seta := a.panel.entries[find("singles", "ongo")], a.panel.entries[find("core", "ongo/Seta")]
	if find("singles", "ongo") != find("src", "ongo")+1 || !singles.checked || singles.count != 2 || singles.text != "  Single-game cores" || seta.count != 2 || seta.text != "  Seta 1st gen" {
		t.Fatalf("Single-game cores first, then the multi-game ones: %+v %+v", singles, seta)
	}
	for _, e := range a.panel.entries {
		if e.kind == "core" && e.value != "ongo/Seta" {
			t.Fatalf("a single-game core listed on its own: %+v", e)
		}
	}
	// A on one core hides it in this source alone; the source turns partial
	a.panel.cursor = find("core", "ongo/Seta")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 4 || !a.filters.CoreOff["ongo/Seta"] || a.filters.SrcOff["ongo"] {
		t.Fatalf("Seta off in Ongo: view %d filters %+v", len(a.view), a.filters)
	}
	if e := a.panel.entries[find("src", "ongo")]; !e.partial || e.checked || e.count != 4 {
		t.Fatalf("source must show partial with the full tally: %+v", e)
	}
	if why := a.filters.Why(&a.ds.Rows[0], &a.ds.Der[0], data.StatusUnknown, false, false, data.ROMUnknown); why != "Filters -> Source/core (ongo: Seta 1st gen off)" {
		t.Fatalf("why %q", why)
	}
	saved, _ := json.Marshal(a.filters)
	var restored data.Filters
	if err := json.Unmarshal(saved, &restored); err != nil || !restored.CoreOff["ongo/Seta"] {
		t.Fatalf("core choice must round-trip through JSON: %s", saved)
	}
	// the single-game cores off too collapses into the source off
	a.panel.cursor = find("singles", "ongo")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 2 || !a.filters.SrcOff["ongo"] || len(a.filters.CoreOff) != 0 || len(a.filters.SinglesOff) != 0 {
		t.Fatalf("every row off must mean the source off: %+v", a.filters)
	}
	// a row back on while its source is off: that row alone
	a.panel.cursor = find("singles", "ongo")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 4 || a.filters.SrcOff["ongo"] || !a.filters.CoreOff["ongo/Seta"] || a.filters.SinglesOff["ongo"] {
		t.Fatalf("single-game cores alone: %+v", a.filters)
	}
	// A on the source restores all of it, then hides all of it
	a.panel.cursor = find("src", "ongo")
	a.actPanel(platform.KeyEnter)
	if a.filters.Active() || len(a.view) != 6 {
		t.Fatalf("source must come back whole: %+v", a.filters)
	}
	a.actPanel(platform.KeyEnter)
	if !a.filters.SrcOff["ongo"] || len(a.filters.CoreOff) != 0 || len(a.view) != 2 {
		t.Fatalf("source again must hide all its games: %+v", a.filters)
	}
	a.actPanel(platform.KeyEnter)
	// Y on a core: only that core; Y again: the whole section back
	a.panel.cursor = find("core", "ongo/Seta")
	if !a.canOnlyFilter() {
		t.Fatal("Y must apply to a core")
	}
	a.actPanel(platform.KeySpace)
	if len(a.view) != 2 || !a.filters.SrcOff["ppriest"] || !a.filters.SrcOff["jtbindb"] || !a.filters.SinglesOff["ongo"] {
		t.Fatalf("only Ongo's Seta: view %d %+v", len(a.view), a.filters)
	}
	a.actPanel(platform.KeySpace)
	if a.filters.Active() {
		t.Fatalf("second Y must restore the section: %+v", a.filters)
	}
	// Y on the single-game cores: only those
	a.panel.cursor = find("singles", "ongo")
	a.actPanel(platform.KeySpace)
	if len(a.view) != 2 || !a.filters.CoreOff["ongo/Seta"] || a.filters.SinglesOff["ongo"] {
		t.Fatalf("only Ongo's single-game cores: view %d %+v", len(a.view), a.filters)
	}
	a.SetFilters(data.Filters{})
	// Left closes the cores and lands on the source; a core choice reopens
	// them on the next visit
	a.buildPanel()
	a.panel.cursor = find("core", "ongo/Seta")
	a.actPanel(platform.KeyLeft)
	if e := a.panel.entries[a.panel.cursor]; e.kind != "src" || e.value != "ongo" || a.panel.yearOpen[srcGroup("ongo")] {
		t.Fatalf("Left must close onto the source: %+v", e)
	}
	a.SetFilters(data.Filters{SinglesOff: map[string]bool{"ongo": true}})
	a.openPanel(ScreenFilter)
	if !a.panel.yearOpen[srcGroup("ongo")] || a.panel.sectionClosed["src"] {
		t.Fatal("an active core choice must open Source/core and its source")
	}
	if !a.filterSectionActive("src") {
		t.Fatal("the Source/core heading must show the core choice as active")
	}
}

// Single-game cores off hides the ones the catalogue gains later too, and a
// core that gains a second game leaves the group as it was: off.
func TestSingleGameCoresFollowTheCatalogue(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(sourceCoreRows(), "", time.Now()), nil)
	openExpandedFilters(a)
	a.panel.yearOpen = map[string]bool{srcGroup("ongo"): true}
	a.buildPanel()
	for i, e := range a.panel.entries {
		if e.kind == "singles" && e.value == "ongo" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyEnter)
	f := a.filters
	if !f.SinglesOff["ongo"] || !f.CoreOff["ongo/BallySente"] || !f.CoreOff["ongo/Cave"] || len(a.view) != 4 {
		t.Fatalf("single-game cores off: view %d %+v", len(a.view), f)
	}
	// next catalogue: Ongo gains Toaplan (one game) and a second Cave game
	rows := append(sourceCoreRows(),
		data.Row{K: "t", Title: "Tango", Base: "Arcade", Src: "ongo", Core: "Toaplan"},
		data.Row{K: "c2", Title: "Charlie Two", Base: "Arcade", Src: "ongo", Core: "Cave"})
	b := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	b.SetFilters(f)
	if len(b.view) != 4 {
		t.Fatalf("Toaplan, both Cave games and Bravo must stay hidden: view %d", len(b.view))
	}
	openExpandedFilters(b)
	cave := false
	for _, e := range b.panel.entries {
		if e.kind == "core" && e.value == "ongo/Cave" {
			cave = !e.checked
		}
		if e.kind == "singles" && e.value == "ongo" && (e.checked || e.partial) {
			t.Fatalf("single-game cores must read off: %+v", e)
		}
	}
	if !cave {
		t.Fatal("Cave must be listed on its own, still off")
	}
}
