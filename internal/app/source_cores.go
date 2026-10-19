package app

import (
	"sort"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Filters > Source opens into the cores each source ships, the way Type >
// Arcade opens into Stable and Beta. SrcOff hides a whole source; CoreOff
// hides one core of it (data.CoreKey). Every core of a source off collapses
// into the source off, so the two never say the same thing twice.

// srcGroup is the yearOpen key of a source's fold.
func srcGroup(src string) string { return "src:" + src }

// buildSourceCores lists each source's cores within the enabled catalogue,
// by label, for the panel being built: a source with a single core has
// nothing to open.
func (a *App) buildSourceCores() map[string][]string {
	label := map[string]string{}
	bySrc := map[string][]string{}
	for i := range a.ds.Rows {
		r := &a.ds.Rows[i]
		if !a.catalogueIncludes(r) {
			continue
		}
		key := data.CoreKey(r)
		if _, ok := label[key]; ok {
			continue
		}
		label[key] = a.ds.Der[i].CoreLabel
		bySrc[r.Src] = append(bySrc[r.Src], key)
	}
	for src, keys := range bySrc {
		sort.Slice(keys, func(i, j int) bool { return data.Compare(label[keys[i]], label[keys[j]]) < 0 })
		bySrc[src] = keys
	}
	a.panel.coreLabel = label
	return bySrc
}

// sourceEntries is one source's row of the Source section with its cores
// when open; count is the source's own tally.
func (a *App) sourceEntries(src, text string, count int, counts map[string]int) []panelEntry {
	f := &a.filters
	on := !f.SrcOff[src]
	cores := a.panel.srcCores[src]
	off := 0
	for _, k := range cores {
		if f.CoreOff[k] {
			off++
		}
	}
	if len(cores) < 2 {
		return []panelEntry{{text: text, kind: "src", value: src, checked: on && off == 0, partial: on && off > 0, count: count, showCount: true}}
	}
	open := a.panel.yearOpen[srcGroup(src)]
	arrow := gfx.ArrowRight
	if open {
		arrow = gfx.ArrowDown
	}
	E := []panelEntry{{text: arrow + " " + text, kind: "src", value: src, checked: on && off == 0, partial: on && off > 0, count: count, showCount: true}}
	if open {
		for _, k := range cores {
			E = append(E, panelEntry{text: "  " + a.panel.coreLabel[k], kind: "core", value: k, checked: on && !f.CoreOff[k], count: counts[k], showCount: true})
		}
	}
	return E
}

// toggleSource applies A (only=false) or Y (only=true) to a source row or
// one of its cores.
func (a *App) toggleSource(only bool) bool {
	e := a.panel.entries[a.panel.cursor]
	f := a.filters
	src, core := e.value, ""
	if e.kind == "core" {
		src, core = data.CoreKeySrc(e.value), e.value
	}
	srcOff, coreOff := copySet(f.SrcOff), copySet(f.CoreOff)
	listed := a.panel.srcCores[src]
	// clear forgets every core choice of the source, listed or not
	clear := func() {
		for k := range coreOff {
			if data.CoreKeySrc(k) == src {
				delete(coreOff, k)
			}
		}
	}
	// others is every listed core of the source but core
	others := func() map[string]bool {
		out := map[string]bool{}
		for _, k := range listed {
			if k != core {
				out[k] = true
			}
		}
		return out
	}
	switch {
	case only:
		// only this source or core: everything else in the section off,
		// unless it already is, in which case the whole section comes back
		wantSrc, wantCore := map[string]bool{}, map[string]bool{}
		for v := range a.ds.Facets.Src {
			if v != src {
				wantSrc[v] = true
			}
		}
		if core != "" {
			wantCore = others()
		}
		if sameSet(srcOff, wantSrc) && sameSet(coreOff, wantCore) {
			srcOff, coreOff = map[string]bool{}, map[string]bool{}
		} else {
			srcOff, coreOff = wantSrc, wantCore
		}
	case core == "":
		// the source: all on unless everything already is
		srcOn := !srcOff[src]
		for _, k := range listed {
			srcOn = srcOn && !coreOff[k]
		}
		clear()
		if srcOn {
			srcOff[src] = true
		} else {
			delete(srcOff, src)
		}
	case srcOff[src]:
		// a core coming back while its source is off: that core alone
		delete(srcOff, src)
		clear()
		for k := range others() {
			coreOff[k] = true
		}
	case coreOff[core]:
		delete(coreOff, core)
	default:
		coreOff[core] = true
		all := true
		for _, k := range listed {
			all = all && coreOff[k]
		}
		if all {
			srcOff[src] = true
			clear()
		}
	}
	f.SrcOff, f.CoreOff = nilIfEmpty(srcOff), nilIfEmpty(coreOff)
	a.SetFilters(f)
	a.buildPanel()
	return true
}
