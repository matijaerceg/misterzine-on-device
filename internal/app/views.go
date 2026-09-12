package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Options -> Views: which of the list orders Y cycles through. The choice
// is kept as the names of the views left out (settings.json "views_off"),
// so a view added later is on for everyone; a fresh install leaves only
// Recents out.

// viewOn reports whether a view is in the Y cycle.
func (a *App) viewOn(m data.SortMode) bool { return m.Valid() && !a.viewsOff[m] }

// ViewsOff lists the views left out of the cycle, by settings name, for
// the host to persist.
func (a *App) ViewsOff() []string {
	var out []string
	for _, m := range data.ViewOrder {
		if a.viewsOff[m] {
			out = append(out, m.Name())
		}
	}
	return out
}

// parseViewsOff turns the saved names into the set; unknown names are
// dropped, and a set that would leave nothing on is ignored.
func parseViewsOff(names []string) map[data.SortMode]bool {
	off := map[data.SortMode]bool{}
	for _, n := range names {
		if m, ok := data.ParseSort(n); ok {
			off[m] = true
		}
	}
	if len(off) >= len(data.ViewOrder) {
		return map[data.SortMode]bool{}
	}
	return off
}

// viewsOnCount is how many views the cycle has.
func (a *App) viewsOnCount() int {
	n := 0
	for _, m := range data.ViewOrder {
		if a.viewOn(m) {
			n++
		}
	}
	return n
}

// firstView is the first view in the cycle that is on: where a visit
// starts when the remembered order is off or turned off.
func (a *App) firstView() data.SortMode {
	for _, m := range data.ViewOrder {
		if a.viewOn(m) {
			return m
		}
	}
	return data.SortUpdated
}

// nextSort is the view Y moves to: the next one on in ViewOrder, wrapping.
func (a *App) nextSort() data.SortMode {
	order := data.ViewOrder
	at := -1
	for i, m := range order {
		if m == a.mode {
			at = i
		}
	}
	for step := 1; step <= len(order); step++ {
		if m := order[(at+step+len(order))%len(order)]; a.viewOn(m) {
			return m
		}
	}
	return a.mode
}

// setViewOn puts a view into the cycle or takes it out. The last view on
// stays on; taking out the current view moves the list to the next one.
func (a *App) setViewOn(m data.SortMode, on bool) bool {
	if !m.Valid() || a.viewOn(m) == on {
		return false
	}
	if !on && a.viewsOnCount() <= 1 {
		return false
	}
	if a.viewsOff == nil {
		a.viewsOff = map[data.SortMode]bool{}
	}
	if on {
		delete(a.viewsOff, m)
	} else {
		a.viewsOff[m] = true
		if a.mode == m {
			a.SetSort(a.nextSort())
		}
	}
	if a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
	return true
}

// viewLabels name the views on the Views page and in the Views help.
var viewLabels = map[data.SortMode]string{
	data.SortUpdated: "Core updated", data.SortDebut: "MiSTer debut", data.SortYear: "Original year",
	data.SortAlphabetical: "A-Z", data.SortMaker: "Maker", data.SortFavorites: "Favorites", data.SortRecents: "Recents",
}

// viewsEntries builds the Views page: one checkbox per view in cycle order.
func (a *App) viewsEntries() []panelEntry {
	E := []panelEntry{{text: a.btn("Y") + " cycles the views that are on:", header: true, info: true}}
	only := a.viewsOnCount() <= 1
	for _, m := range data.ViewOrder {
		on := a.viewOn(m)
		E = append(E, panelEntry{text: viewLabels[m], kind: "view", value: m.Name(), checked: on, disabled: on && only})
	}
	return E
}

// viewsSummary is the Options row's tally, "6 of 7".
func (a *App) viewsSummary() string {
	return itoa(a.viewsOnCount()) + " of " + itoa(len(data.ViewOrder))
}

// openViews opens the Views page from Options.
func (a *App) openViews() {
	a.screen = ScreenViews
	a.panel.entries = nil
	a.panel.cursor = 0
	a.panel.top = 0
	a.buildPanel()
	a.all = true
}

// closeViews returns to Options on the Views row.
func (a *App) closeViews() {
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "views" {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}
