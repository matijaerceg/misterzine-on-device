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
		var section data.Filters
		copySection(e.kind, &section, a.filters)
		a.panel.sectionClosed[e.kind] = !section.Active()
	}
	if a.iniFilter() != "" {
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

func (a *App) filterHint() string {
	arrows := gfx.ArrowLeft + " " + gfx.ArrowRight
	hint := "A toggle  " + arrows + " open/close  B back"
	if a.canOnlyFilter() {
		hint = "A toggle  Y only/all  " + arrows + " open/close  B back"
	}
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		hint = "A toggle  " + arrows + " fold  B back"
		if a.canOnlyFilter() {
			hint = "A  Y only  " + arrows + " fold  B back"
		}
	}
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		hint = "A  Y only  " + arrows + " fold  B"
	}
	return hint
}
