package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Type > Arcade opens into two children, Stable and Beta (the Patreon beta
// cores), like a decade opens into years. BaseOff["Arcade"] hides every
// arcade row; BetaOff hides one child. Both children off collapses into
// Arcade off, so the two never say the same thing twice.

// arcadeEntries is the Arcade row of the Type section with its children
// when open; count is the type's own tally.
func (a *App) arcadeEntries(count int) []panelEntry {
	f := &a.filters
	on := !f.BaseOff["Arcade"]
	open := a.panel.yearOpen["Arcade"]
	arrow := gfx.ArrowRight
	if open {
		arrow = gfx.ArrowDown
	}
	E := []panelEntry{{text: arrow + " Arcade", kind: "base", value: "Arcade", checked: on && len(f.BetaOff) == 0, partial: on && len(f.BetaOff) > 0, count: count, showCount: true}}
	if open {
		counts := a.facetCounts("beta")
		for _, v := range []struct{ value, text string }{{"stable", "Stable"}, {"beta", "Beta"}} {
			E = append(E, panelEntry{text: "  " + v.text, kind: "beta", value: v.value, checked: on && !f.BetaOff[v.value], count: counts[v.value], showCount: true})
		}
	}
	return E
}

// toggleArcade applies A (only=false) or Y (only=true) to the Arcade row or
// one of its children.
func (a *App) toggleArcade(only bool) bool {
	e := a.panel.entries[a.panel.cursor]
	f := a.filters
	base, beta := copySet(f.BaseOff), copySet(f.BetaOff)
	child := ""
	if e.kind == "beta" {
		child = e.value
	}
	other := map[string]string{"stable": "beta", "beta": "stable"}[child]
	switch {
	case only:
		// only this child (or only Arcade): everything else in the section
		// off, unless it already is, in which case the whole section comes back
		wantBase, wantBeta := map[string]bool{}, map[string]bool{}
		for v := range a.ds.Facets.Base {
			if v != "Arcade" {
				wantBase[v] = true
			}
		}
		if child != "" {
			wantBeta[other] = true
		}
		if sameSet(base, wantBase) && sameSet(beta, wantBeta) {
			base, beta = map[string]bool{}, map[string]bool{}
		} else {
			base, beta = wantBase, wantBeta
		}
	case child == "":
		// the parent: all on unless everything already is
		if base["Arcade"] || len(beta) > 0 {
			delete(base, "Arcade")
			beta = map[string]bool{}
		} else {
			base["Arcade"] = true
			beta = map[string]bool{}
		}
	case base["Arcade"]:
		// a child coming back while the type is off: that child alone
		delete(base, "Arcade")
		beta = map[string]bool{other: true}
	case beta[child]:
		delete(beta, child)
	default:
		beta[child] = true
		if beta["stable"] && beta["beta"] {
			base["Arcade"] = true
			beta = map[string]bool{}
		}
	}
	f.BaseOff, f.BetaOff = nilIfEmpty(base), nilIfEmpty(beta)
	a.SetFilters(f)
	a.buildPanel()
	return true
}

func copySet(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range m {
		if v {
			out[k] = true
		}
	}
	return out
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func nilIfEmpty(m map[string]bool) map[string]bool {
	if len(m) == 0 {
		return nil
	}
	return m
}
