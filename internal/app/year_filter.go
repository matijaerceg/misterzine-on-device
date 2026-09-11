package app

import "github.com/matijaerceg/misterzine-on-device/internal/data"

func (a *App) yearEntries() []panelEntry {
	entries := []panelEntry{{text: "Original release year", header: true, kind: "year"},
		{text: "Right: years  Left: close", info: true}}
	groups := map[string]int{}
	for y, n := range a.ds.Facets.Year {
		groups[data.Decade(y)] += n
	}
	counts := a.facetCounts("year")
	for _, decade := range sortedFacet(groups) {
		if decade == "" {
			entries = append(entries, panelEntry{text: "Unknown", kind: "year", value: "", checked: !a.filters.YearOff[""], count: counts[""], showCount: true})
			continue
		}
		on, total, count := 0, 0, 0
		var years []panelEntry
		for _, y := range sortedFacet(a.ds.Facets.Year) {
			if data.Decade(y) != decade {
				continue
			}
			total++
			if !a.filters.YearOff[y] {
				on++
			}
			count += counts[y]
			years = append(years, panelEntry{text: "  " + y, kind: "year", value: y, checked: !a.filters.YearOff[y], count: counts[y], showCount: true})
		}
		arrow := " > years"
		if a.panel.yearOpen[decade] {
			arrow = " < close"
		}
		entries = append(entries, panelEntry{text: decade + arrow, kind: "decade", value: decade, checked: on == total, partial: on > 0 && on < total, count: count, showCount: true})
		if a.panel.yearOpen[decade] {
			entries = append(entries, years...)
		}
	}
	return entries
}

// Left/Right expand or collapse only when on a decade or one of its years.
func (a *App) expandYears(open bool) bool {
	if a.panel.cursor >= len(a.panel.entries) {
		return false
	}
	e := a.panel.entries[a.panel.cursor]
	decade := e.value
	if e.kind == "year" && !e.header && e.value != "" {
		decade = data.Decade(e.value)
	} else if e.kind != "decade" {
		return false
	}
	if a.panel.yearOpen == nil {
		a.panel.yearOpen = map[string]bool{}
	}
	a.panel.yearOpen[decade] = open
	// Move to the parent before rebuilding so a collapsed year cannot leave
	// selection pointing at an unrelated filter further down the panel.
	if !open {
		for i, parent := range a.panel.entries {
			if parent.kind == "decade" && parent.value == decade {
				a.panel.cursor = i
				break
			}
		}
	}
	a.buildPanel()
	if open && a.panel.lines > 3 {
		// Show the first few child years immediately when the decade was
		// selected near the bottom of the screen.
		a.panel.top = max(a.panel.top, a.panel.cursor-a.panel.lines+4)
	}
	return true
}

func (a *App) toggleYears(only bool) bool {
	e := a.panel.entries[a.panel.cursor]
	selected := func(y string) bool {
		return e.header || (e.kind == "decade" && data.Decade(y) == e.value) || (e.kind == "year" && y == e.value)
	}
	off := map[string]bool{}
	for y, v := range a.filters.YearOff {
		if v {
			off[y] = true
		}
	}
	allOn, alreadyOnly := true, true
	for y := range a.ds.Facets.Year {
		if selected(y) && off[y] {
			allOn = false
		}
		if off[y] == selected(y) {
			alreadyOnly = false
		}
	}
	if only {
		off = map[string]bool{}
		if !alreadyOnly {
			for y := range a.ds.Facets.Year {
				if !selected(y) {
					off[y] = true
				}
			}
		}
	} else {
		for y := range a.ds.Facets.Year {
			if selected(y) {
				if allOn {
					off[y] = true
				} else {
					delete(off, y)
				}
			}
		}
		if e.header && !allOn {
			off = map[string]bool{}
		}
	}
	f := a.filters
	f.YearOff = off
	a.SetFilters(f)
	a.buildPanel()
	return true
}
