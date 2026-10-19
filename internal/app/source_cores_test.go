package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestSourceOpensIntoCores(t *testing.T) {
	rows := []data.Row{
		{K: "a1", Title: "Alpha One", Base: "Arcade", Src: "ongo", Core: "Seta"},
		{K: "a2", Title: "Alpha Two", Base: "Arcade", Src: "ongo", Core: "Seta"},
		{K: "b", Title: "Bravo", Base: "Arcade", Src: "ongo", Core: "BallySente"},
		{K: "p", Title: "Papa", Base: "Arcade", Src: "ppriest", Core: "Seta"},
		{K: "j", Title: "Juliet", Base: "Arcade", Src: "jtbindb", Core: "jtcps1"},
	}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
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
	if !ongo.checked || ongo.partial || ongo.count != 3 || ongo.text != gfx.ArrowRight+" Ongo" {
		t.Fatalf("Ongo starts closed and fully on: %+v", ongo)
	}
	// a source with one core has nothing to open
	if e := a.panel.entries[find("src", "ppriest")]; e.text != data.SrcShort("ppriest") {
		t.Fatalf("single-core source: %+v", e)
	}
	a.panel.cursor = find("src", "ppriest")
	if a.selectedDecade() != "" {
		t.Fatal("a single-core source must not open")
	}
	for _, e := range a.panel.entries {
		if e.kind == "core" {
			t.Fatal("cores listed while closed")
		}
	}
	a.panel.cursor = find("src", "ongo")
	a.actPanel(platform.KeyRight)
	bally, seta := a.panel.entries[find("core", "ongo/BallySente")], a.panel.entries[find("core", "ongo/Seta")]
	if find("core", "ongo/BallySente") > find("core", "ongo/Seta") || !bally.checked || bally.count != 1 || seta.count != 2 || bally.text != "  Bally/Sente SAC-1" || seta.text != "  Seta 1st gen" {
		t.Fatalf("cores after opening, by label: %+v %+v", bally, seta)
	}
	// A on one core hides it in this source alone; the source turns partial
	a.panel.cursor = find("core", "ongo/Seta")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 3 || !a.filters.CoreOff["ongo/Seta"] || a.filters.SrcOff["ongo"] {
		t.Fatalf("Seta off in Ongo: view %d filters %+v", len(a.view), a.filters)
	}
	if e := a.panel.entries[find("src", "ongo")]; !e.partial || e.checked || e.count != 3 {
		t.Fatalf("source must show partial with the full tally: %+v", e)
	}
	if why := a.filters.Why(&a.ds.Rows[0], &a.ds.Der[0], data.StatusUnknown, false, false, data.ROMUnknown); why != "Filters -> Source (ongo: Seta 1st gen off)" {
		t.Fatalf("why %q", why)
	}
	saved, _ := json.Marshal(a.filters)
	var restored data.Filters
	if err := json.Unmarshal(saved, &restored); err != nil || !restored.CoreOff["ongo/Seta"] {
		t.Fatalf("core choice must round-trip through JSON: %s", saved)
	}
	// the other core off too collapses into the source off
	a.panel.cursor = find("core", "ongo/BallySente")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 2 || !a.filters.SrcOff["ongo"] || len(a.filters.CoreOff) != 0 {
		t.Fatalf("every core off must mean the source off: %+v", a.filters)
	}
	// a core back on while its source is off: that core alone
	a.panel.cursor = find("core", "ongo/BallySente")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 3 || a.filters.SrcOff["ongo"] || !a.filters.CoreOff["ongo/Seta"] || a.filters.CoreOff["ongo/BallySente"] {
		t.Fatalf("BallySente alone: %+v", a.filters)
	}
	// A on the source restores all of it, then hides all of it
	a.panel.cursor = find("src", "ongo")
	a.actPanel(platform.KeyEnter)
	if a.filters.Active() || len(a.view) != 5 {
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
	if len(a.view) != 2 || !a.filters.SrcOff["ppriest"] || !a.filters.SrcOff["jtbindb"] || !a.filters.CoreOff["ongo/BallySente"] {
		t.Fatalf("only Ongo's Seta: view %d %+v", len(a.view), a.filters)
	}
	a.actPanel(platform.KeySpace)
	if a.filters.Active() {
		t.Fatalf("second Y must restore the section: %+v", a.filters)
	}
	// Y on a source after a core choice: that source whole, nothing else
	a.SetFilters(data.Filters{CoreOff: map[string]bool{"ongo/Seta": true}})
	a.buildPanel()
	a.panel.cursor = find("src", "ongo")
	a.actPanel(platform.KeySpace)
	if len(a.view) != 3 || len(a.filters.CoreOff) != 0 || len(a.filters.SrcOff) != 2 {
		t.Fatalf("only Ongo: %+v", a.filters)
	}
	a.SetFilters(data.Filters{})
	// Left closes the cores and lands on the source; a core choice reopens
	// them on the next visit
	a.buildPanel()
	a.panel.cursor = find("core", "ongo/BallySente")
	a.actPanel(platform.KeyLeft)
	if e := a.panel.entries[a.panel.cursor]; e.kind != "src" || e.value != "ongo" || a.panel.yearOpen[srcGroup("ongo")] {
		t.Fatalf("Left must close onto the source: %+v", e)
	}
	a.SetFilters(data.Filters{CoreOff: map[string]bool{"ongo/Seta": true}})
	a.openPanel(ScreenFilter)
	if !a.panel.yearOpen[srcGroup("ongo")] || a.panel.sectionClosed["src"] {
		t.Fatal("an active core choice must open Source and its source")
	}
	if !a.filterSectionActive("src") {
		t.Fatal("the Source heading must show the core choice as active")
	}
}
