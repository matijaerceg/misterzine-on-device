package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The shared Available views editor keeps existing settings names: standard
// views are enabled unless in views_off; genres are off unless in genre_views_on.
// Both families share viewsOff inside the app.

// viewOn reports whether a view is in the chooser.
func (a *App) viewOn(m data.SortMode) bool { return m.Valid() && !a.viewsOff[m] }

// ViewsOff lists the views left out of the chooser, by settings name, for
// the host to persist; the genre views are GenreViewsOn's.
func (a *App) ViewsOff() []string {
	out := []string{} // never nil, so every view on saves as [] rather than null
	for _, m := range a.viewsPageOrder() {
		if a.viewsOff[m] {
			out = append(out, m.Name())
		}
	}
	return out
}

// GenreViewsOn lists the genre views in the chooser, by settings name, for
// the host to persist.
func (a *App) GenreViewsOn() []string {
	out := []string{}
	for _, m := range data.GenreViews {
		if a.viewOn(m) {
			out = append(out, m.Name())
		}
	}
	return out
}

// parseViewsOff turns the saved names into the set: the views off by name
// and the genre views not on by name. Unknown names are dropped, and a
// choice that would leave nothing on is ignored.
func parseViewsOff(off, genreOn []string) map[data.SortMode]bool {
	set := map[data.SortMode]bool{}
	for _, m := range data.GenreViews {
		set[m] = true
	}
	if data.ViewsOn(off, genreOn) == 0 {
		return set
	}
	for _, n := range off {
		if m, ok := data.ParseSort(n); ok && m.Genre() == "" {
			set[m] = true
		}
	}
	for _, n := range genreOn {
		if m, ok := data.ParseSort(n); ok && m.Genre() != "" {
			delete(set, m)
		}
	}
	return set
}

// viewsPageOrder is the standard-view order, also used for saving views_off.
func (a *App) viewsPageOrder() []data.SortMode {
	out := make([]data.SortMode, 0, len(data.ViewOrder))
	for _, m := range data.ViewOrder {
		if m.Genre() == "" {
			out = append(out, m)
		}
	}
	return out
}

// viewsOnCount is how many views are enabled.
func (a *App) viewsOnCount() int {
	n := 0
	for _, m := range data.ViewOrder {
		if a.viewOn(m) {
			n++
		}
	}
	return n
}

// firstView is the first view in the chooser that is on: where a visit
// starts when the remembered order is off or turned off.
func (a *App) firstView() data.SortMode {
	for _, m := range data.ViewOrder {
		if a.viewOn(m) {
			return m
		}
	}
	return data.SortUpdated
}

// nextSort finds the next enabled view when the current one is disabled.
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

// setViewOn puts a view into the chooser or takes it out. The last view on
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

// viewLabels name the views in the chooser, shared editor and
// the Default view row.
var viewLabels = map[data.SortMode]string{
	data.SortUpdated: "Build date", data.SortDebut: "MiSTer debut", data.SortYear: "Original year",
	data.SortAlphabetical: "A-Z", data.SortMaker: "Manufacturer", data.SortCore: "Core", data.SortFavorites: "Favorites", data.SortRecents: "Recents",
}

func init() {
	for _, m := range data.GenreViews {
		viewLabels[m] = m.String()
	}
}

// viewsEntries lists both families under non-selectable headings.
func (a *App) viewsEntries() []panelEntry {
	views := data.ViewOrder
	E := []panelEntry{{text: "Standard views", header: true, info: true}}
	genre := false
	only := a.viewsOnCount() <= 1
	for _, m := range views {
		if m.Genre() != "" && !genre {
			E = append(E, panelEntry{text: "Genres", header: true, info: true})
			genre = true
		}
		on := a.viewOn(m)
		E = append(E, panelEntry{text: viewLabels[m], kind: "view", value: m.Name(), checked: on, disabled: on && only})
	}
	return E
}

// viewsSummary counts enabled views across both families.
func (a *App) viewsSummary() string {
	return itoa(a.viewsOnCount()) + " of " + itoa(len(data.ViewOrder))
}

// openViews opens the shared editor, remembering its entry point.
func (a *App) openViews(row string) {
	a.viewsPage = row
	a.screen = ScreenViews
	a.panel.entries = nil
	a.panel.cursor = 0
	a.panel.top = 0
	a.buildPanel()
	a.all = true
}

// viewsTitle names the open checkbox page in its top bar.
func (a *App) viewsTitle() string { return "Available views" }

// closeViews returns to the chooser or the originating Options row.
func (a *App) closeViews() {
	if a.viewsPage == "chooser" {
		a.openViewChooser()
		return
	}
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == a.viewsPage {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}

// DefaultView is the enabled starting view; unavailable choices fall back safely.
func (a *App) DefaultView() data.SortMode {
	if a.viewOn(a.cfg.DefaultSort) {
		return a.cfg.DefaultSort
	}
	return a.firstView()
}
func (a *App) enabledViews() []data.SortMode {
	var modes []data.SortMode
	for _, m := range data.ViewOrder {
		if a.viewOn(m) {
			modes = append(modes, m)
		}
	}
	return modes
}
func (a *App) defaultViewEntry() panelEntry {
	e := panelEntry{text: "Default view", kind: "default-view", child: true, depth: 1,
		help: "Start in " + viewLabels[a.DefaultView()] + ". Used when MisterZine opens with remembering off. Only enabled views are offered."}
	for i, m := range a.enabledViews() {
		e.vals = append(e.vals, viewLabels[m])
		if m == a.DefaultView() {
			e.idx = i
		}
	}
	return e
}
