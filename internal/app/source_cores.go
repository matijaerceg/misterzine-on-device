package app

import (
	"sort"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Filters > Source/core opens into the arcade cores each source ships, the
// way Type > Arcade opens into Stable and Beta: one row per core that runs
// two games or more, then one row for all its single-game cores (the Core
// view's gathering, data.CoreSingleGame). SrcOff hides a whole source,
// CoreOff one core of it (data.CoreKey), SinglesOff its single-game cores.
// Every row of a source off collapses into the source off, so the two never
// say the same thing twice. A source with fewer than two rows to show has
// nothing to open.

// singlesText names a source's single-game cores row.
const singlesText = "Single-game cores"

// srcGroup is the yearOpen key of a source's fold.
func srcGroup(src string) string { return "src:" + src }

// buildSourceCores lists each source's arcade cores within the enabled
// catalogue for the panel being built: the multi-game cores by label into
// srcCores, the single-game ones into srcSingles.
func (a *App) buildSourceCores() {
	label := map[string]string{}
	multi, singles := map[string][]string{}, map[string][]string{}
	for i := range a.ds.Rows {
		r, d := &a.ds.Rows[i], &a.ds.Der[i]
		if !r.IsArcade() || r.Core == "" || !a.catalogueIncludes(r) {
			continue
		}
		key := data.CoreKey(r)
		if _, ok := label[key]; ok {
			continue
		}
		label[key] = d.CoreLabel
		if d.CoreGroup == data.CoreSingleGame {
			singles[r.Src] = append(singles[r.Src], key)
		} else {
			multi[r.Src] = append(multi[r.Src], key)
		}
	}
	for src, keys := range multi {
		sort.Slice(keys, func(i, j int) bool { return data.Compare(label[keys[i]], label[keys[j]]) < 0 })
		multi[src] = keys
	}
	a.panel.srcCores, a.panel.srcSingles, a.panel.coreLabel = multi, singles, label
}

// sourceOpens reports whether a source has two rows or more to open into.
func (a *App) sourceOpens(src string) bool {
	n := len(a.panel.srcCores[src])
	if len(a.panel.srcSingles[src]) > 0 {
		n++
	}
	return n > 1
}

// singlesOn reports how many of a source's single-game cores are on, of
// how many: SinglesOff turns them all off, CoreOff single ones.
func (a *App) singlesOn(f *data.Filters, src string) (on, total int) {
	for _, k := range a.panel.srcSingles[src] {
		if !f.SinglesOff[src] && !f.CoreOff[k] {
			on++
		}
	}
	return on, len(a.panel.srcSingles[src])
}

// sourceEntries is one source's row of the Source/core section with its
// cores when open; count is the source's own tally.
func (a *App) sourceEntries(src, text string, count int, counts map[string]int) []panelEntry {
	f := &a.filters
	on := !f.SrcOff[src]
	off := 0
	for _, k := range a.panel.srcCores[src] {
		if f.CoreOff[k] {
			off++
		}
	}
	singlesOn, singles := a.singlesOn(f, src)
	if singlesOn < singles || f.SinglesOff[src] {
		off++
	}
	if !a.sourceOpens(src) {
		return []panelEntry{{text: text, kind: "src", value: src, checked: on && off == 0, partial: on && off > 0, count: count, showCount: true}}
	}
	open := a.panel.yearOpen[srcGroup(src)]
	arrow := gfx.ArrowRight
	if open {
		arrow = gfx.ArrowDown
	}
	E := []panelEntry{{text: arrow + " " + text, kind: "src", value: src, checked: on && off == 0, partial: on && off > 0, count: count, showCount: true}}
	if !open {
		return E
	}
	if singles > 0 {
		n := 0
		for _, k := range a.panel.srcSingles[src] {
			n += counts[k]
		}
		E = append(E, panelEntry{text: "  " + singlesText, kind: "singles", value: src, checked: on && singlesOn == singles,
			partial: on && singlesOn > 0 && singlesOn < singles, count: n, showCount: true})
	}
	for _, k := range a.panel.srcCores[src] {
		E = append(E, panelEntry{text: "  " + a.panel.coreLabel[k], kind: "core", value: k, checked: on && !f.CoreOff[k], count: counts[k], showCount: true})
	}
	return E
}

// toggleSource applies A (only=false) or Y (only=true) to a source row, one
// of its cores or its single-game cores.
func (a *App) toggleSource(only bool) bool {
	e := a.panel.entries[a.panel.cursor]
	f := a.filters
	src, core := e.value, ""
	switch e.kind {
	case "core":
		src, core = data.CoreKeySrc(e.value), e.value
	case "singles":
		core = singlesText
	}
	srcOff, coreOff, singlesOff := copySet(f.SrcOff), copySet(f.CoreOff), copySet(f.SinglesOff)
	multi, singles := a.panel.srcCores[src], a.panel.srcSingles[src]
	// clear forgets every core choice of the source, listed or not
	clear := func() {
		for k := range coreOff {
			if data.CoreKeySrc(k) == src {
				delete(coreOff, k)
			}
		}
		delete(singlesOff, src)
	}
	setSingles := func(coreOff, singlesOff map[string]bool, off bool) {
		for _, k := range singles {
			if off {
				coreOff[k] = true
			} else {
				delete(coreOff, k)
			}
		}
		if off {
			singlesOff[src] = true
		} else {
			delete(singlesOff, src)
		}
	}
	singlesOn := func() bool {
		if singlesOff[src] {
			return false
		}
		for _, k := range singles {
			if coreOff[k] {
				return false
			}
		}
		return true
	}
	// othersOff turns off every row of the source but the one chosen
	othersOff := func(coreOff, singlesOff map[string]bool) {
		for _, k := range multi {
			if k != core {
				coreOff[k] = true
			}
		}
		if len(singles) > 0 && core != singlesText {
			setSingles(coreOff, singlesOff, true)
		}
	}
	allOff := func() bool {
		for _, k := range multi {
			if !coreOff[k] {
				return false
			}
		}
		return len(singles) == 0 || singlesOff[src]
	}
	switch {
	case only:
		// only this source or row: everything else in the section off,
		// unless it already is, in which case the whole section comes back
		wantSrc, wantCore, wantSingles := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for v := range a.ds.Facets.Src {
			if v != src {
				wantSrc[v] = true
			}
		}
		if core != "" {
			othersOff(wantCore, wantSingles)
		}
		if sameSet(srcOff, wantSrc) && sameSet(coreOff, wantCore) && sameSet(singlesOff, wantSingles) {
			srcOff, coreOff, singlesOff = map[string]bool{}, map[string]bool{}, map[string]bool{}
		} else {
			srcOff, coreOff, singlesOff = wantSrc, wantCore, wantSingles
		}
	case e.kind == "src":
		// the source: all on unless everything already is
		on := !srcOff[src] && singlesOn()
		for _, k := range multi {
			on = on && !coreOff[k]
		}
		clear()
		if on {
			srcOff[src] = true
		} else {
			delete(srcOff, src)
		}
	case srcOff[src]:
		// a row coming back while its source is off: that row alone
		delete(srcOff, src)
		clear()
		othersOff(coreOff, singlesOff)
	default:
		if e.kind == "singles" {
			setSingles(coreOff, singlesOff, singlesOn())
		} else if coreOff[core] {
			delete(coreOff, core)
		} else {
			coreOff[core] = true
		}
		if allOff() {
			srcOff[src] = true
			clear()
		}
	}
	f.SrcOff, f.CoreOff, f.SinglesOff = nilIfEmpty(srcOff), nilIfEmpty(coreOff), nilIfEmpty(singlesOff)
	a.SetFilters(f)
	a.buildPanel()
	return true
}
