package app

import (
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
	arrows := gfx.ArrowLeft + " " + gfx.ArrowRight
	if a.onFilterSectionHeader() {
		// A and the arrows both open or close the heading under the cursor.
		return "A " + arrows + " open/close  B back"
	}
	hint := "A toggle  " + arrows + " open/close  B back"
	if a.canOnlyFilter() {
		hint = "A toggle  Y only/all  " + arrows + " open/close  B back"
	}
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		hint = "A toggle  " + arrows + " open/close  B"
		if a.canOnlyFilter() {
			hint = "A toggle  Y only  " + arrows + " open/close  B back"
		}
	}
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		// At the largest tate inset, retain action descriptions and omit
		// the conventional Back reminder rather than clipping a control.
		hint = "A toggle  " + gfx.ArrowLeft + gfx.ArrowRight + " open/close"
		if a.canOnlyFilter() {
			hint = "A toggle  Y only  " + gfx.ArrowLeft + gfx.ArrowRight + " open/close"
		}
	}
	return hint
}
