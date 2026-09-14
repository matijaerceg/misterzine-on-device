package app

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

func (a *App) resetFilterExpansion() {
	a.panel.sectionClosed = map[string]bool{}
	a.panel.yearOpen = map[string]bool{}
	for _, e := range a.rawFilterEntries() {
		if !e.header || e.info || e.kind == "" {
			continue
		}
		a.panel.sectionClosed[e.kind] = !a.filterSectionActive(e.kind)
	}
	if a.rotationFilter() != "" {
		a.panel.sectionClosed["rot"] = false
	}
	if len(a.filters.BetaOff) > 0 {
		a.panel.yearOpen["Arcade"] = true // reveal the Stable/Beta choice
	}
	// Reveal individual years only for partially selected decades.
	total, hidden := map[string]int{}, map[string]int{}
	for year := range a.ds.Facets.Year {
		decade := data.Decade(year)
		if decade == "" {
			continue
		}
		total[decade]++
		if a.filters.YearOff[year] {
			hidden[decade]++
		}
	}
	for decade, n := range hidden {
		a.panel.yearOpen[decade] = n > 0 && n < total[decade]
	}
}

// onFilterSectionHeader reports whether the cursor sits on a section
// heading of the Filters page (not a decade, which has its own checkbox).
func (a *App) onFilterSectionHeader() bool {
	if a.screen != ScreenFilter || a.panel.cursor >= len(a.panel.entries) {
		return false
	}
	e := a.panel.entries[a.panel.cursor]
	return e.header && !e.info && e.kind != ""
}

// toggleFilterSection opens a closed section heading or closes an open one.
func (a *App) toggleFilterSection() bool {
	if !a.onFilterSectionHeader() {
		return false
	}
	kind := a.panel.entries[a.panel.cursor].kind
	if kind == "rot" && a.rotationFilter() != "" {
		return false
	}
	return a.expandFilterSection(a.panel.sectionClosed[kind])
}

func (a *App) filterSectionActive(kind string) bool {
	if kind == "rot" && a.rotationFilter() != "" {
		return true
	}
	var section data.Filters
	copySection(kind, &section, a.filters)
	return section.Active()
}

func (a *App) filterHint() string {
	p := &a.panel
	if p.cursor >= len(p.entries) {
		return "B Back"
	}
	e := p.entries[p.cursor]
	if e.disabled || e.info || (e.kind == "rot" && a.rotationFilter() != "") {
		return "B Back"
	}
	if e.kind == "clear" {
		return "A Clear  B Back"
	}
	if a.onFilterSectionHeader() {
		if p.sectionClosed[e.kind] {
			return "A " + gfx.ArrowRight + " Open  B Back"
		}
		return "A " + gfx.ArrowLeft + " Close  B Back"
	}
	parts := []string{"A Toggle", "B Back"}
	if a.canOnlyFilter() {
		parts = append(parts, "Y Only/all")
	}
	// Left closes the containing group; Right only opens a closed subgroup.
	if group := a.selectedDecade(); group != "" && !p.yearOpen[group] {
		parts = append(parts, gfx.ArrowLeft+" Close", gfx.ArrowRight+" Open")
	} else {
		parts = append(parts, gfx.ArrowLeft+" Close")
	}
	hint := strings.Join(parts, "  ")
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		hint = strings.ReplaceAll(hint, "Y Only/all", "Y Only")
	}
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		hint = strings.ReplaceAll(hint, "  B Back", "")
	}
	return hint
}
