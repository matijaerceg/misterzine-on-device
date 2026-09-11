package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestArcadeTypeSplitsIntoStableAndBeta(t *testing.T) {
	rows := []data.Row{
		{K: "s", Title: "Stable", Base: "Arcade"},
		{K: "b", Title: "Beta", Base: "Arcade", Beta: true},
		{K: "c", Title: "Console", Base: "Console"},
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
	arcade := a.panel.entries[find("base", "Arcade")]
	if !arcade.checked || arcade.partial || arcade.count != 2 || arcade.text != gfx.ArrowRight+" Arcade" {
		t.Fatalf("Arcade starts closed and fully on: %+v", arcade)
	}
	for _, e := range a.panel.entries {
		if e.kind == "beta" {
			t.Fatal("children listed while closed")
		}
	}
	a.panel.cursor = find("base", "Arcade")
	a.actPanel(platform.KeyRight)
	stable, beta := a.panel.entries[find("beta", "stable")], a.panel.entries[find("beta", "beta")]
	if !stable.checked || !beta.checked || stable.count != 1 || beta.count != 1 || stable.text != "  Stable" || beta.text != "  Beta" {
		t.Fatalf("children after opening: %+v %+v", stable, beta)
	}
	// A on Beta hides the beta core; Arcade turns partial
	a.panel.cursor = find("beta", "beta")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 2 || a.CursorKey() == "b" || !a.filters.BetaOff["beta"] || a.filters.BaseOff["Arcade"] {
		t.Fatalf("beta child off: view %d filters %+v", len(a.view), a.filters)
	}
	if e := a.panel.entries[find("base", "Arcade")]; !e.partial || e.checked || e.count != 2 {
		t.Fatalf("parent must show partial with the full tally: %+v", e)
	}
	saved, _ := json.Marshal(a.filters)
	var restored data.Filters
	if err := json.Unmarshal(saved, &restored); err != nil || !restored.BetaOff["beta"] {
		t.Fatal("beta choice must round-trip through JSON")
	}
	// the other child off too collapses into Arcade off
	a.panel.cursor = find("beta", "stable")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 1 || !a.filters.BaseOff["Arcade"] || len(a.filters.BetaOff) != 0 {
		t.Fatalf("both children off must mean Arcade off: %+v", a.filters)
	}
	if e := a.panel.entries[find("base", "Arcade")]; e.checked || e.partial {
		t.Fatalf("parent must show off: %+v", e)
	}
	// a child back on while the type is off: that child alone
	a.panel.cursor = find("beta", "beta")
	a.actPanel(platform.KeyEnter)
	if len(a.view) != 2 || a.filters.BaseOff["Arcade"] || !a.filters.BetaOff["stable"] || a.filters.BetaOff["beta"] {
		t.Fatalf("beta alone: %+v", a.filters)
	}
	// A on the parent restores everything
	a.panel.cursor = find("base", "Arcade")
	a.actPanel(platform.KeyEnter)
	if a.filters.Active() || len(a.view) != 3 {
		t.Fatalf("parent must restore the type: %+v", a.filters)
	}
	a.actPanel(platform.KeyEnter)
	if !a.filters.BaseOff["Arcade"] || len(a.view) != 1 {
		t.Fatal("parent again must hide every arcade row")
	}
	a.actPanel(platform.KeyEnter)
	// Y on a child: only that child; Y again: the whole section back
	a.panel.cursor = find("beta", "beta")
	if !a.canOnlyFilter() {
		t.Fatal("Y must apply to a child")
	}
	a.actPanel(platform.KeySpace)
	if len(a.view) != 1 || a.CursorKey() != "b" || !a.filters.BaseOff["Console"] || !a.filters.BetaOff["stable"] {
		t.Fatalf("only beta: %+v", a.filters)
	}
	a.actPanel(platform.KeySpace)
	if a.filters.Active() {
		t.Fatalf("second Y must restore the section: %+v", a.filters)
	}
	// Left closes the children and lands on the parent; a beta choice
	// reopens them on the next visit
	a.panel.cursor = find("beta", "stable")
	a.actPanel(platform.KeyLeft)
	if e := a.panel.entries[a.panel.cursor]; e.kind != "base" || e.value != "Arcade" || a.panel.yearOpen["Arcade"] {
		t.Fatalf("Left must close onto the parent: %+v", e)
	}
	a.SetFilters(data.Filters{BetaOff: map[string]bool{"beta": true}})
	a.openPanel(ScreenFilter)
	if !a.panel.yearOpen["Arcade"] || a.panel.sectionClosed["base"] {
		t.Fatal("an active beta choice must open Type and Arcade")
	}
	if !a.filterSectionActive("base") {
		t.Fatal("the Type heading must show the beta choice as active")
	}
}
