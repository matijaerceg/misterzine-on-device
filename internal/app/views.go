package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Options -> Views: which of the list orders Y cycles through. The choice
// is kept as the names of the views left out (settings.json "views_off"),
// so a view added later is on for everyone; a fresh install leaves only
// Recents out. The genre views (Options -> Genre views) go the other way:
// off unless named in "genre_views_on", since nobody asked for four more
// stops in the cycle. In the app both pages share one set, viewsOff.

// viewOn reports whether a view is in the Y cycle.
func (a *App) viewOn(m data.SortMode) bool { return m.Valid() && !a.viewsOff[m] }

// ViewsOff lists the views left out of the cycle, by settings name, for
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

// GenreViewsOn lists the genre views in the cycle, by settings name, for
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

// viewsPageOrder is the Views page: every view but the genre views, which
// have their own page.
func (a *App) viewsPageOrder() []data.SortMode {
	out := make([]data.SortMode, 0, len(data.ViewOrder))
	for _, m := range data.ViewOrder {
		if m.Genre() == "" {
			out = append(out, m)
		}
	}
	return out
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

// viewLabels name the views on the Views and Genre views pages and in
// the Default view row.
var viewLabels = map[data.SortMode]string{
	data.SortUpdated: "Build date", data.SortDebut: "MiSTer debut", data.SortYear: "Original year",
	data.SortAlphabetical: "A-Z", data.SortMaker: "Manufacturer", data.SortCore: "Core", data.SortFavorites: "Favorites", data.SortRecents: "Recents",
	data.SortShmups: "Shoot 'em ups", data.SortFighting: "Fighting", data.SortBeatEmUps: "Beat 'em ups", data.SortRunNGuns: "Run 'n guns",
}

// viewsPageViews are the views the open checkbox page lists: the Views
// page's, or the genre views on the Genre views page.
func (a *App) viewsPageViews() []data.SortMode {
	if a.viewsPage == "genre-views" {
		return data.GenreViews
	}
	return a.viewsPageOrder()
}

// viewsEntries builds the Views or Genre views page: one checkbox per
// view in cycle order, nothing else; the Options row already says what
// the page is for.
func (a *App) viewsEntries() []panelEntry {
	views := a.viewsPageViews()
	E := make([]panelEntry, 0, len(views))
	only := a.viewsOnCount() <= 1
	for _, m := range views {
		on := a.viewOn(m)
		E = append(E, panelEntry{text: viewLabels[m], kind: "view", value: m.Name(), checked: on, disabled: on && only})
	}
	return E
}

// viewsSummary is the Views row's tally, "7 of 8": the genre views count
// on their own row.
func (a *App) viewsSummary() string {
	n := 0
	for _, m := range a.viewsPageOrder() {
		if a.viewOn(m) {
			n++
		}
	}
	return itoa(n) + " of " + itoa(len(a.viewsPageOrder()))
}

// genreViewsSummary is the Genre views row's tally, "0 of 4".
func (a *App) genreViewsSummary() string {
	return itoa(len(a.GenreViewsOn())) + " of " + itoa(len(data.GenreViews))
}

// openViews opens the Views page ("views") or the Genre views page
// ("genre-views") from its Options row.
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
func (a *App) viewsTitle() string {
	if a.viewsPage == "genre-views" {
		return "Genre views"
	}
	return "Views"
}

// closeViews returns to Options on the row that opened the page.
func (a *App) closeViews() {
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
