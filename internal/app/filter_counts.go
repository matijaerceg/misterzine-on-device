package app

import (
	"slices"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Counts decorate existing rows. Rebuilding the panel discards outstanding
// work, so a filter edit, card scan or catalogue replacement gets fresh totals.
type filterCountWork struct {
	kinds []string
	next  time.Time
}

func (e panelEntry) filterCountKind() string {
	if e.header || e.info {
		return ""
	}
	switch e.kind {
	case "decade":
		return "year"
	case "singles":
		return "core"
	case "install", "rom", "base", "beta", "src", "core", "year", "rot", "res", "genre", "patch", "directions", "buttons", "plr":
		return e.kind
	}
	return ""
}

func (a *App) filterEntries() []panelEntry {
	entries := a.filterRows()
	var work filterCountWork
	for _, e := range entries {
		if kind := e.filterCountKind(); kind != "" && !slices.Contains(work.kinds, kind) {
			work.kinds = append(work.kinds, kind)
		}
	}
	// Static renderers have no animation clock and request the settled page.
	if !a.transition.enabled {
		for _, kind := range work.kinds {
			a.fillFilterCounts(entries, kind)
		}
		work.kinds = nil
	}
	a.panel.filterCounts = work
	return entries
}

// Arm the first tally only after the destination frame has been painted.
// Between tallies the host can present the new numbers and service input.
func (a *App) scheduleFilterCounts() {
	w := &a.panel.filterCounts
	if len(w.kinds) != 0 && w.next.IsZero() && !a.PageTransitionRunning() && !a.saver.active {
		w.next = a.cfg.TimerNow().Add(frameDur)
	}
}

func (a *App) tickFilterCounts(now time.Time) bool {
	w := &a.panel.filterCounts
	if a.screen != ScreenFilter || w.next.IsZero() || now.Before(w.next) || a.PageTransitionRunning() || a.saver.active {
		return false
	}
	a.fillFilterCounts(a.panel.entries, w.kinds[0])
	w.kinds = w.kinds[1:]
	w.next = time.Time{}
	a.all = true
	return true
}

func (a *App) fillFilterCounts(entries []panelEntry, kind string) {
	var counts map[string]int
	switch kind {
	case "install":
		counts = a.installFilterCounts()
	case "rom":
		counts = a.romFilterCounts()
	default:
		counts = a.facetCounts(kind)
	}
	for i := range entries {
		e := &entries[i]
		if e.filterCountKind() != kind || counts == nil {
			continue
		}
		e.showCount = true
		switch e.kind {
		case "decade":
			for year, n := range counts {
				if data.Decade(year) == e.value {
					e.count += n
				}
			}
		case "singles":
			for _, core := range a.panel.srcSingles[e.value] {
				e.count += counts[core]
			}
		default:
			e.count = counts[e.value]
		}
	}
}

// Each section counts within the other filters and the search. Once the card
// scan or ROM check has answered, an empty match shows zero.
func (a *App) installFilterCounts() map[string]int {
	if !a.anyRow(func(i int) bool { return a.status(i) != data.StatusUnknown }) {
		return nil
	}
	f := a.effectiveFilters()
	f.Install = ""
	counts := map[data.Status]int{}
	total := 0
	a.facetRows(f, false, func(i int) { counts[a.status(i)]++; total++ })
	return map[string]int{
		data.InstallAll:     total,
		data.InstallFound:   counts[data.StatusCurrent] + counts[data.StatusOutdated] + counts[data.StatusLikelyOutdated] + counts[data.StatusFoundUndated],
		data.InstallCurrent: counts[data.StatusCurrent],
		data.InstallOlder:   counts[data.StatusOutdated] + counts[data.StatusLikelyOutdated],
		data.InstallUndated: counts[data.StatusFoundUndated],
		data.InstallMissing: counts[data.StatusNotFound],
	}
}

func (a *App) romFilterCounts() map[string]int {
	if !a.anyRow(func(i int) bool { return a.romState(i) != data.ROMUnknown }) {
		return nil
	}
	f := a.effectiveFilters()
	f.ROM = ""
	counts := map[data.ROMState]int{}
	total := 0
	a.facetRows(f, false, func(i int) { counts[a.romState(i)]++; total++ })
	return map[string]int{
		data.ROMAll:    total,
		data.ROMLaunch: counts[data.ROMLaunchIssue],
		data.ROMAny:    counts[data.ROMLaunchIssue] + counts[data.ROMOtherIssue],
		data.ROMNone:   counts[data.ROMClean],
	}
}
