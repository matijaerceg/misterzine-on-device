package app

import (
	"image"
	"sort"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// panelEntry is one line of the Filters or Options screen.
type panelEntry struct {
	text      string
	header    bool
	info      bool     // plain text, never selectable
	help      string   // shown in the help area while selected
	kind      string   // filter section: "base","src","rot","plr","genre","install","fav"; settings: "rotation","inset","prefetch","rescan","refresh","clear","about","settings","back"
	value     string   // facet value for filter entries
	vals      []string // settings: the choices, Left/Right pick one
	idx       int      // settings: the current choice
	checked   bool
	partial   bool // some, but not all, years in a decade are enabled
	disabled  bool // settings: shown muted, Left/Right ignored
	count     int
	showCount bool
}

type panelState struct {
	entries       []panelEntry
	cursor        int
	top           int
	yearOpen      map[string]bool // expanded decades; browsing state only
	sectionClosed map[string]bool // section browsing state; filters stay unchanged
	// settings snapshot
	prefetch bool
	lines    int // entries that fit, from the last paint (paging)
	// optionsRow is the Options row B left from, so Options reopens on it
	// the way the main list keeps its row; nil before the first visit.
	optionsRow *panelEntry
}

func (a *App) openPanel(s Screen) {
	a.screen = s
	a.panel.entries = nil
	a.panel.cursor = 0
	a.panel.top = 0
	if s == ScreenFilter {
		a.resetFilterExpansion()
	} else if a.panel.optionsRow != nil {
		// reopen on the row left last time; buildPanel finds it by identity
		a.panel.entries = []panelEntry{*a.panel.optionsRow}
	}
	a.buildPanel()
	a.skipDisabled() // Filters opens on its first heading while Clear all filters is greyed out
	a.all = true
}

// skipDisabled moves the cursor off a greyed row onto the next row that
// can be used.
func (a *App) skipDisabled() {
	p := &a.panel
	for p.cursor < len(p.entries)-1 && p.entries[p.cursor].disabled {
		p.cursor++
		for p.cursor < len(p.entries)-1 && (p.entries[p.cursor].info || (p.entries[p.cursor].header && p.entries[p.cursor].text == "")) {
			p.cursor++
		}
	}
}

func (a *App) buildPanel() {
	// remember the entry under the cursor by identity, not by index
	var was *panelEntry
	if a.panel.cursor < len(a.panel.entries) {
		w := a.panel.entries[a.panel.cursor]
		was = &w
	}
	switch a.screen {
	case ScreenOptions:
		a.panel.entries = a.optionsEntries()
	case ScreenViews:
		a.panel.entries = a.viewsEntries()
	case ScreenCredits:
		a.panel.entries = a.creditsEntries()
	default:
		a.panel.entries = a.filterEntries()
	}
	if was != nil {
		for i, e := range a.panel.entries {
			if e.kind == was.kind && e.value == was.value && e.header == was.header && (e.header || e.kind != "" || e.text == was.text) {
				a.panel.cursor = i
				break
			}
		}
	}
	if a.panel.cursor >= len(a.panel.entries) {
		a.panel.cursor = len(a.panel.entries) - 1
	}
	// skip spacer and info lines
	for a.panel.cursor < len(a.panel.entries)-1 && (a.panel.entries[a.panel.cursor].info || (a.panel.entries[a.panel.cursor].header && a.panel.entries[a.panel.cursor].text == "")) {
		a.panel.cursor++
	}
	a.all = true
}

func sortedFacet(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return data.Compare(keys[i], keys[j]) < 0 })
	return keys
}

// facetCounts applies the search and every filter except the section being
// counted. Unchecked choices keep their potential counts and remain selectable.
func (a *App) facetCounts(kind string) map[string]int {
	f := a.effectiveFilters()
	if a.mode == data.SortFavorites {
		f.FavOnly = true
	}
	value := func(r *data.Row, d *data.Derived) string { return "" }
	switch kind {
	case "base":
		f.BaseOff, f.BetaOff = nil, nil
		value = func(r *data.Row, d *data.Derived) string { return r.Base }
	case "beta": // Arcade's Stable/Beta children count within the other types' choices
		f.BetaOff = nil
		base := map[string]bool{}
		for v := range f.BaseOff {
			if v != "Arcade" {
				base[v] = true
			}
		}
		f.BaseOff = base
		value = func(r *data.Row, d *data.Derived) string { return data.BetaKind(r) }
	case "src":
		f.SrcOff = nil
		value = func(r *data.Row, d *data.Derived) string { return r.Src }
	case "rot":
		f.RotOff = nil
		value = func(r *data.Row, d *data.Derived) string { return d.RotGroup }
	case "res":
		f.ResOff = nil
		value = func(r *data.Row, d *data.Derived) string { return r.Res }
	case "genre":
		f.GenreOff = nil
		value = func(r *data.Row, d *data.Derived) string { return r.Genre }
	case "directions":
		f.DirectionsOff = nil
		value = func(r *data.Row, d *data.Derived) string { return d.Directions }
	case "buttons":
		f.ButtonsOff = nil
		value = func(r *data.Row, d *data.Derived) string { return d.Buttons }
	case "plr":
		f.PlrOff = nil
		value = func(r *data.Row, d *data.Derived) string { return r.Plr }
	case "year":
		f.YearOff = nil
		value = func(r *data.Row, d *data.Derived) string { return d.Year }
	}
	counts := map[string]int{}
	query := searchText(a.query)
	for i := range a.ds.Rows {
		r, d := &a.ds.Rows[i], &a.ds.Der[i]
		if kind != "base" && kind != "src" && !r.IsArcade() {
			continue
		}
		if query != "" && !strings.Contains(searchText(d.Title), query) {
			continue
		}
		unseen := a.seen != nil && a.seen.Unseen(r)
		if f.Pass(r, d, a.status(i), a.cfg.Favorites[r.K], unseen) {
			counts[value(r, d)]++
		}
	}
	return counts
}

func (a *App) filterEntries() []panelEntry {
	var out []panelEntry
	hidden := false
	for _, e := range a.rawFilterEntries() {
		if e.header {
			hidden = false
			if !e.info && e.kind != "" {
				hidden = a.panel.sectionClosed[e.kind]
				arrow := gfx.ArrowDown
				if hidden {
					arrow = gfx.ArrowRight
				}
				mark := ""
				if a.filterSectionActive(e.kind) {
					mark = "* "
				}
				e.text = arrow + " " + mark + e.text
			}
			out = append(out, e)
		} else if !hidden {
			out = append(out, e)
		}
	}
	return out
}

func (a *App) rawFilterEntries() []panelEntry {
	f := &a.filters
	var E []panelEntry
	text := "Clear all filters"
	if a.rotationFilter() != "" {
		text = "Clear other filters"
	}
	E = append(E, panelEntry{text: text, kind: "clear", disabled: !f.Active()}, panelEntry{header: true, info: true})
	if a.rotationFilter() != "" {
		E = append(E, panelEntry{text: a.rotationFilterLabel(), info: true},
			panelEntry{text: "Change in Options", info: true})
	}
	E = append(E, panelEntry{text: "On the card", header: true, kind: "install"})
	counts := a.cardCounts()
	installCounts := map[string]int{
		data.InstallAll:     a.total,
		data.InstallFound:   counts[data.StatusCurrent] + counts[data.StatusOutdated] + counts[data.StatusLikelyOutdated] + counts[data.StatusFoundUndated],
		data.InstallCurrent: counts[data.StatusCurrent], data.InstallOlder: counts[data.StatusOutdated] + counts[data.StatusLikelyOutdated],
		data.InstallUndated: counts[data.StatusFoundUndated], data.InstallMissing: counts[data.StatusNotFound],
	}
	for _, v := range []struct{ val, text string }{{data.InstallAll, "everything"}, {data.InstallFound, "found on card"}, {data.InstallCurrent, "up to date"}, {data.InstallOlder, "older installed"}, {data.InstallUndated, "date unknown"}, {data.InstallMissing, "not found on card"}} {
		cur := f.Install
		if cur == "" {
			cur = data.InstallAll
		}
		E = append(E, panelEntry{text: v.text, kind: "install", value: v.val, checked: cur == v.val, count: installCounts[v.val], showCount: counts[data.StatusUnknown] < a.total})
	}
	E = append(E, panelEntry{text: "Since last look", header: true, kind: "since"})
	if a.seen == nil || a.seen.BaseRows == nil {
		E = append(E, panelEntry{text: "available after your first visit", info: true})
	} else {
		E = append(E, panelEntry{text: "only rows changed since my last look", kind: "since", checked: f.Since})
	}
	hiddenSrc := a.effectiveFilters().SrcHidden
	section := func(title, kind string, facet map[string]int, off map[string]bool, label func(string) string) {
		counts := a.facetCounts(kind)
		E = append(E, panelEntry{text: title, header: true, kind: kind})
		if kind == "rot" && a.rotationFilter() != "" {
			E = append(E, panelEntry{text: a.rotationFilterLabel(), info: true})
			return
		}
		for _, v := range sortedFacet(facet) {
			if kind == "base" && v == "Arcade" {
				E = append(E, a.arcadeEntries(counts[v])...)
				continue
			}
			if kind == "src" && hiddenSrc[v] {
				continue // Options -> Sources: installed; not a choice here
			}
			E = append(E, panelEntry{text: label(v), kind: kind, value: v, checked: !off[v], count: counts[v], showCount: true})
		}
	}
	ident := func(s string) string { return s }
	section("Type", "base", a.ds.Facets.Base, f.BaseOff, ident)
	section("Source", "src", a.ds.Facets.Src, f.SrcOff, func(s string) string {
		if s == "" {
			return "Unknown"
		}
		return data.SrcShort(s)
	})
	if f.BaseOff["Arcade"] || a.ds.Facets.Base["Arcade"] == 0 {
		return E
	}
	E = append(E, panelEntry{header: true, info: true},
		panelEntry{text: "Arcade game filters:", header: true, info: true})
	E = append(E, a.yearEntries()...)
	section("Rotation", "rot", a.ds.Facets.Rot, f.RotOff, func(s string) string {
		switch s {
		case "h":
			return "Horizontal"
		case "v":
			return "Vertical"
		}
		return "Unknown"
	})
	section("Resolution", "res", a.ds.Facets.Res, f.ResOff, func(s string) string {
		if s == "" {
			return "Unknown"
		}
		return s
	})
	section("Genre", "genre", a.ds.Facets.Genre, f.GenreOff, func(s string) string {
		if s == "" {
			return "Unknown"
		}
		return s
	})
	section("Controls", "directions", a.ds.Facets.Directions, f.DirectionsOff, func(s string) string {
		if s == "" {
			return "Unknown"
		}
		return s
	})
	section("Buttons", "buttons", a.ds.Facets.Buttons, f.ButtonsOff, func(s string) string {
		if s == "" {
			return "Unknown"
		}
		if s == "1" {
			return "1 button"
		}
		return s + " buttons"
	})
	section("Players", "plr", a.ds.Facets.Plr, f.PlrOff, func(s string) string {
		if s == "" {
			return "Unknown"
		}
		return s
	})
	return E
}

func (a *App) optionsEntries() []panelEntry {
	rotIdx := map[gfx.Rotation]int{gfx.RotRight: 0, gfx.RotNone: 1, gfx.RotLeft: 2}[a.rot]
	scrollIdx := 1
	for i, v := range ScrollValues {
		if v == a.scrollText() {
			scrollIdx = i
		}
	}
	launcherIdx := 0
	if a.cfg.Launcher != nil && a.cfg.Launcher() {
		launcherIdx = 1
	}
	prefetchIdx := 0
	if a.panel.prefetch {
		prefetchIdx = 1
	}
	saverIdx := 1
	for i, v := range saverValues {
		if v == a.Screensaver() {
			saverIdx = i
		}
	}
	dateIdx := 0
	for i, f := range dateFormats {
		if f == a.DateFormat() {
			dateIdx = i
		}
	}
	rotationHelp := "Left/Right turn the image; the choice is saved. Labels describe the monitor's turn."
	if a.FollowRotation() {
		rotationHelp = "Set by the active MiSTer INI. Turn off Follow INI rotation above to rotate manually."
	}
	updateText := "Run Update All"
	updateHelp := "Update with live output and a stage bar. Hold " + a.btn("B") + " for 2 seconds to cancel; system writes finish first. A restart may be required."
	if a.appUpdate != "" {
		updateText = "Update MisterZine + all"
		updateHelp = "MisterZine " + a.appUpdate + " is available. Run Update All, then quit and reopen MisterZine to use it."
	}
	group := func(title string) panelEntry { return panelEntry{text: title, header: true, info: true} }
	spacer := panelEntry{header: true, info: true}
	E := []panelEntry{group("Data:"),
		{text: "Refresh data now", kind: "refresh",
			help: "Check misterzine.fyi for new releases now. This also happens on launch and every 30 minutes."},
		{text: updateText, kind: "update", help: updateHelp},
		{text: "Last update result", kind: "update-result",
			help: "Review the last Update All result and its saved output. This does not start another update."},
		{text: "Rescan card", kind: "rescan",
			help: "Refresh on-card status after an external update. The built-in Update All rescans automatically when it finishes."},
		{text: "Prefetch shots" + a.progressText(), kind: "prefetch", vals: []string{"off", "on"}, idx: prefetchIdx,
			help: "Download every screenshot in the background (about 55 MB) so browsing never waits; the tally counts up as they land. Off: only what you look at."},
		{text: "Clear image cache", kind: "clearimg",
			help: "Delete the downloaded screenshots and system photos; they come back as you browse."},
		spacer, group("List:"),
		{text: "Sources", kind: "sources", vals: []string{"all", "installed only"}, idx: map[bool]int{false: 0, true: 1}[a.cfg.InstalledOnly],
			help: a.sourcesHelp()},
		{text: "Filter by rotation", kind: "filter-rotation", vals: []string{"off", "on"}, idx: map[bool]int{false: 0, true: 1}[a.FilterRotation()],
			help: "Show only games made for the current orientation (INI or manual); unknowns hidden. Off restores manual filters."},
		{text: "Remember last view", kind: "remember-sort", vals: []string{"off", "on"}, idx: map[bool]int{false: 0, true: 1}[a.RememberSort()],
			help: "On (default): reopen in the view you left, Favorites included. Off: every visit starts in Core updated."},
		{text: "Views (" + a.viewsSummary() + ")", kind: "views",
			help: "Which views " + a.btn("Y") + " cycles through: core updated, MiSTer debut, original year, A-Z, manufacturer, Favorites, Recents (launches from here). " + a.btn("A") + " opens the list."},
		{text: "Title font", kind: "title-font", vals: []string{"normal", "narrow", "narrow tall"}, idx: map[string]int{"normal": 0, "narrow": 1, "tall": 2}[a.TitleFont()],
			help: "Narrow fonts fit a third more title; tall (default) matches the body font height. Normal: body font."},
		{text: "List shots", kind: "list-shot", vals: []string{"gameplay", "title"}, idx: map[string]int{"gameplay": 0, "title": 1}[a.ListShot()],
			help: "Which screenshot the list pane shows: gameplay (default) or the title screen. Details and the artwork view still show every shot."},
		{text: "Date format", kind: "date-format", vals: dateFormatLabels, idx: dateIdx,
			help: a.dateFormatHelp()},
		{text: "List layout", kind: "list-layout", vals: listLayouts, idx: map[string]int{"list": 0, "split": 1, "picture": 2}[a.ListLayout()],
			help: "List (default): the full list with a small pane. Split: a wider pane with a bigger picture. Picture: the picture across the screen with a few rows."},
		spacer, group("Display:"),
		{text: "Follow INI rotation", kind: "follow-rotation", vals: []string{"off", "on"}, idx: map[bool]int{false: 0, true: 1}[a.FollowRotation()],
			help: "On (default): match osd_rotate in the active MiSTer INI at every startup. Off: rotate manually below. Never edits the INI."},
		{text: "Rotation", kind: "rotation", vals: []string{"monitor CW", "horizontal", "monitor CCW"}, idx: rotIdx, disabled: a.FollowRotation(),
			help: rotationHelp},
		{text: "Screensaver", kind: "screensaver", vals: []string{"off", "1 min", "2 min", "5 min", "10 min"}, idx: saverIdx,
			help: "Dim and scroll black lettering after idle time. Left/Right sets the delay; " + a.btn("A") + " previews. Other buttons wake without acting; Menu works as usual."},
		{text: "Button labels", kind: "button-labels", vals: buttonLabelValues(), idx: map[string]int{"mister": 0, "xbox": 1, "playstation": 2, "numbers": 3}[a.ButtonLabels()],
			help: "How the legends name the pad buttons, in MiSTer's A B X Y order as set in its define buttons screen. Xbox and PlayStation names go by position."},
		{text: "Edit safe zone", kind: "inset",
			help: "Margin kept clear of the screen edge (overscan): now " + itoa(a.cfg.SafeInsetX) + " px at the sides, " + itoa(a.cfg.SafeInsetY) + " px top and bottom. A opens the frame; fit it just inside the picture."},
		{text: "HDMI picture", kind: "canvas", vals: []string{"fit display", "320x240"}, idx: map[bool]int{false: 0, true: 1}[a.Canvas() == "320x240"],
			help: "Fit display (default): fills the HDMI screen height, 360x270 on 1080p, for more rows. 320x240: the classic size with bars. Applies at the next start."},
		spacer, group("Operation:"),
		{text: "Scroll speed", kind: "scroll", vals: []string{"20 Hz", "30 Hz", "60 Hz"}, idx: scrollIdx,
			help: "How many rows (or pages, with Left/Right) a held direction moves per second. 60 Hz is one row every frame."},
		{text: "Hold delay", kind: "hold-delay", vals: []string{"short", "normal", "long"}, idx: map[int]int{200: 0, 300: 1, 500: 2}[a.HoldDelay()],
			help: "Wait before held navigation repeats: short 200 ms, normal 300 ms, long 500 ms. Scroll speed sets the pace after this delay."},
		{text: "Main menu launcher", kind: "launcher", vals: []string{"off", "on"}, idx: launcherIdx,
			help: "Show MisterZine in the MiSTer main menu. Off removes the entry when you leave; on puts it back. With it off, Scripts -> MisterZine-Run opens MisterZine."},
		{text: "Open at boot", kind: "open-at-boot", vals: []string{"off", "on"}, idx: map[bool]int{false: 0, true: 1}[a.cfg.OpenAtBoot], disabled: launcherIdx == 0,
			help: "On: once the MiSTer menu is up after power-on or reboot, MisterZine opens as if picked from it. A bootcore in the INI wins. Needs the launcher."},
		{text: "Return after game", kind: "return-after-game", vals: []string{"off", "on"}, idx: map[bool]int{false: 0, true: 1}[a.cfg.ReturnAfterGame], disabled: launcherIdx == 0,
			help: "On: when a game started here exits to the MiSTer menu, MisterZine reopens on that game. Not after quitting with the Menu button. Needs the launcher."},
		{text: "Menu button", kind: "menu-button", vals: []string{"Options", "quit MisterZine"}, idx: map[bool]int{false: 0, true: 1}[a.MenuButton() == "leave"],
			help: "What the pad button defined as MiSTer's menu (OSD) button does here: Options (default; held 2 s it quits) or quit at once. Keyboard F12 still quits."},
		{text: "Troubleshooting", kind: "troubleshooting",
			help: "Test your Start button or game launching. Results stay on screen for a photo; no keyboard or log files needed."},
		{text: "Credits", kind: "credits"},
		{text: "Quit MisterZine", kind: "quit"},
	}
	return E
}

// dateFormatHelp shows today's date in the chosen list format.
func (a *App) dateFormatHelp() string {
	today := a.cfg.Now().Format("2006-01-02")
	example := formatListDate(a.DateFormat(), today, a.cfg.Now().Year())
	if a.DateFormat() == "yymmdd" {
		return "How list dates read. Today is " + example + ": two-digit year, month, day, for every row."
	}
	return "How list dates read. Today is " + example + ". Rows from earlier years show the year instead."
}

func (a *App) scrollText() string {
	if a.cfg.Scroll == "" {
		return "30"
	}
	return a.cfg.Scroll
}

func (a *App) progressText() string {
	if a.cfg.Progress == nil {
		return ""
	}
	have, total := a.cfg.Progress()
	if total == 0 {
		return ""
	}
	return "  " + itoa(have) + "/" + itoa(total)
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func short(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

func (a *App) paintPanel(c *gfx.Canvas) {
	l := &a.lay
	font := a.sm
	a.paintStatus(c)
	if a.notice == "" {
		c.Fill(l.Status, gen.Eva.Surface)
		title := "Filters"
		if a.screen == ScreenOptions {
			title = "Options"
		} else if a.screen == ScreenViews {
			title = "Views"
		} else if a.screen == ScreenCredits {
			title = "Credits"
		}
		c.Text(l.Status.Min.X+2, l.Status.Min.Y+2, a.sm, title, gen.Eva.Accent)
		a.paintHoldBar(c)
	}
	c.Fill(l.Body, gen.Eva.Bg)
	// Filters frames its entries. Options frames the help text instead and
	// lists the build and data details, greyed, just above the hint bar.
	var inner, helpBox image.Rectangle
	helpLines := 0
	if a.screen == ScreenOptions {
		// Horizontal uses three help lines; the narrower tate view keeps four.
		helpLines = 4
		if !l.Portrait {
			helpLines = 3
		}
		versionH := 2*font.H + 2
		helpH := helpLines*(font.H+1) + 5
		helpBox = image.Rect(l.Body.Min.X, l.Body.Max.Y-versionH-helpH, l.Body.Max.X, l.Body.Max.Y-versionH)
		inner = image.Rect(l.Body.Min.X+2, l.Body.Min.Y+2, l.Body.Max.X-2, helpBox.Min.Y-2)
		cols := font.Cols(l.Body.Dx() - 4)
		vy := helpBox.Max.Y + 2
		c.Text(l.Body.Min.X+2, vy, font, gfx.Fit("misterzine "+a.cfg.Version, cols), gen.Eva.Muted)
		c.Text(l.Body.Min.X+2, vy+font.H, font, gfx.Fit("data "+a.ds.Updated.Format("2006-01-02 15:04")+"  "+short(a.ds.Hash), cols), gen.Eva.Muted)
	} else {
		c.Box(l.Body, gen.Eva.Line)
		inner = l.Body.Inset(2)
	}
	lh := font.H
	lines := inner.Dy() / lh
	p := &a.panel
	p.lines = lines
	// spacer rows are half a row; everything else is a full row
	rowH := func(i int) int {
		if e := p.entries[i]; e.header && e.info && e.text == "" {
			return lh / 2
		}
		return lh
	}
	// fit is how many entries from top fill the area
	fit := func(top int) int {
		y, n := 0, 0
		for i := top; i < len(p.entries) && y+rowH(i) <= inner.Dy(); i++ {
			y += rowH(i)
			n++
		}
		return n
	}
	// the selected row stays near the center, as in the main list, and the
	// list stays filled at the end
	p.top = centeredTop(p.cursor, len(p.entries), lines)
	maxTop := len(p.entries)
	for y := 0; maxTop > 0 && y+rowH(maxTop-1) <= inner.Dy(); maxTop-- {
		y += rowH(maxTop - 1)
	}
	p.top = max(0, min(p.top, maxTop))
	if p.cursor < p.top {
		p.top = p.cursor
	}
	for p.cursor >= p.top+fit(p.top) {
		p.top++
	}
	shown := fit(p.top)
	// the last cell of every row is a gutter for the scroll cues
	edge := inner.Max.X - 2 - font.W
	cols := font.Cols(edge - inner.Min.X - 2)
	vx := a.valueColumn(inner, edge)
	y := inner.Min.Y
	lastY := y
	for n := p.top; n < len(p.entries) && n < p.top+shown; n++ {
		e := p.entries[n]
		lastY = y
		r := image.Rect(inner.Min.X, y, inner.Max.X, y+rowH(n))
		if n == p.cursor {
			c.Fill(r, gen.Eva.Surface)
		}
		switch {
		case e.header && e.info:
			c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, cols), gen.Eva.Muted)
		case e.header:
			c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, cols), gen.Eva.Accent)
		case e.kind == "year" || e.kind == "decade" || e.kind == "res" || e.kind == "base" || e.kind == "beta" || e.kind == "src" || e.kind == "rot" || e.kind == "plr" || e.kind == "genre" || e.kind == "directions" || e.kind == "buttons" || e.kind == "install" || e.kind == "fav" || e.kind == "since" || e.kind == "view":
			mark := "[ ] "
			if e.checked {
				mark = "[x] "
			}
			if e.partial {
				mark = "[-] "
			}
			suffix := ""
			if e.showCount {
				suffix = " (" + itoa(e.count) + ")"
			}
			text := mark + gfx.Fit(e.text, cols-len(mark)-len(suffix)) + suffix
			col := gen.Eva.Fg
			if e.disabled && a.screen == ScreenViews {
				col = gen.Eva.Muted // the last view on stays on
			} else if n == p.cursor && a.screen == ScreenOptions {
				col = gen.Eva.Accent
			}
			c.Text(inner.Min.X+2, y, font, gfx.Fit(text, cols), col)
		case len(e.vals) > 0:
			col := gen.Eva.Fg
			if n == p.cursor {
				col = gen.Eva.Accent
			}
			// the value with an arrow on each side that can still move
			la, ra := " ", " "
			if e.disabled {
				col = gen.Eva.Muted // shown, not changeable here
			} else {
				if e.idx > 0 {
					la = gfx.ArrowLeft
				}
				if e.idx < len(e.vals)-1 {
					ra = gfx.ArrowRight
				}
			}
			val := la + " " + e.vals[e.idx] + " " + ra
			vw := font.Width(val)
			if vx > 0 {
				c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, font.Cols(vx-inner.Min.X-2-font.W)), col)
				c.Text(vx, y, font, val, col)
			} else {
				c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, font.Cols(edge-inner.Min.X-2-font.W-vw)), col)
				c.Text(edge-vw, y, font, val, col)
			}
		default:
			col := gen.Eva.Fg
			if e.disabled {
				col = gen.Eva.Muted // nothing to do yet
			} else if n == p.cursor && a.screen != ScreenFilter {
				col = gen.Eva.Accent
			}
			c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, cols), col)
		}
		y += rowH(n)
	}
	// more above or below: a cue in the gutter of the first or last drawn
	// row, only while entries are actually out of view
	if p.top > 0 {
		c.Text(edge, inner.Min.Y, font, gfx.ArrowUp, gen.Eva.Muted)
	}
	if p.top+shown < len(p.entries) {
		c.Text(edge, lastY, font, gfx.ArrowDown, gen.Eva.Muted)
	}
	if a.screen == ScreenOptions {
		c.Box(helpBox, gen.Eva.Line)
		if p.cursor < len(p.entries) && p.entries[p.cursor].help != "" {
			hy := helpBox.Min.Y + 3
			for _, ln := range gfx.Wrap(p.entries[p.cursor].help, font.Cols(helpBox.Dx()-6), helpLines) {
				c.Text(helpBox.Min.X+3, hy, font, ln, gen.Eva.Fg)
				hy += font.H + 1
			}
		}
		a.paintHint(c, gfx.ArrowLeft+" "+gfx.ArrowRight+" change  A open  B back")
	} else if a.screen == ScreenViews {
		a.paintHint(c, "A on/off  B back")
	} else if a.screen == ScreenCredits {
		a.paintHint(c, "B back")
	} else {
		a.paintHint(c, a.filterHint())
	}
}

// valueColumn is where Options values start: a fixed column at the
// half mark, or nearer the right edge (before the gutter at edge) when
// the widest value needs the room. 0 means right-align each value
// instead, because a fixed column would cut into the labels, as in the
// narrow tate layout.
func (a *App) valueColumn(inner image.Rectangle, edge int) int {
	if a.screen != ScreenOptions {
		return 0
	}
	font := a.sm
	widest, maxVal := 0, 0
	for _, e := range a.panel.entries {
		widest = max(widest, font.Width(e.text))
		for _, v := range e.vals {
			maxVal = max(maxVal, font.Width(gfx.ArrowLeft+" "+v+" "+gfx.ArrowRight))
		}
	}
	x := inner.Min.X + inner.Dx()/2
	if x+maxVal > edge {
		x = edge - maxVal
	}
	if x < inner.Min.X+2+widest+font.W {
		return 0
	}
	return x
}

func (a *App) actPanel(k platform.Key) bool {
	p := &a.panel
	n := len(p.entries)
	selectable := func(i int) bool {
		return i >= 0 && i < n && !p.entries[i].info && !(p.entries[i].header && p.entries[i].text == "") && !(a.screen == ScreenFilter && p.entries[i].disabled)
	}
	switch k {
	case platform.KeyBack:
		if a.screen == ScreenOptions {
			if p.cursor < n {
				row := p.entries[p.cursor]
				p.optionsRow = &row
			}
			a.closeOptions() // back to the screen it was opened over
			return true
		}
		if a.screen == ScreenViews {
			a.closeViews()
			return true
		}
		if a.screen == ScreenCredits {
			a.closeCredits()
			return true
		}
		a.screen = ScreenList
		a.all = true
		return true
	case platform.KeyUp, platform.KeyDown:
		d := 1
		if k == platform.KeyUp {
			d = -1
		}
		for step := 1; step <= n; step++ {
			i := p.cursor + step*d
			// Only a fresh press at the end of Options can wrap.
			if a.screen == ScreenOptions && a.rep.count == 0 {
				i = (i%n + n) % n
			} else if i < 0 || i >= n {
				break
			}
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyLeft:
		if a.screen == ScreenOptions {
			return a.stepValue(-1)
		}
		if a.screen != ScreenFilter {
			return false
		}
		return a.expandFilterSection(false)
	case platform.KeyRight:
		if a.screen == ScreenOptions {
			return a.stepValue(1)
		}
		if a.screen != ScreenFilter {
			return false
		}
		return a.expandFilterSection(true)
	case platform.KeyPageUp, platform.KeyHome:
		if a.screen == ScreenFilter && k == platform.KeyPageUp {
			return a.jumpFilterSection(-1)
		}
		for i := 0; i < n; i++ {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyPageDown, platform.KeyEnd:
		if a.screen == ScreenFilter && k == platform.KeyPageDown {
			return a.jumpFilterSection(1)
		}
		for i := n - 1; i >= 0; i-- {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyTab:
		if a.screen != ScreenFilter {
			return false
		}
		return a.actPanel(platform.KeyBack) // X opened Filters; X closes it again
	case platform.KeySpace:
		if a.screen == ScreenFilter {
			return a.onlyFilter()
		}
		return a.togglePanel()
	case platform.KeyEnter:
		return a.togglePanel()
	default:
		return false
	}
	a.all = true
	return true
}

// nearest is the selectable entry closest to target inside the list: for a
// target past either end, the first or last selectable entry.
func (p *panelState) nearest(target int, selectable func(int) bool) int {
	n := len(p.entries)
	if target < 0 {
		target = 0
	}
	if target > n-1 {
		target = n - 1
	}
	for d := 0; d < n; d++ {
		if selectable(target + d) {
			return target + d
		}
		if selectable(target - d) {
			return target - d
		}
	}
	return p.cursor
}

// stepValue moves a settings choice one step left or right.
func (a *App) stepValue(d int) bool {
	p := &a.panel
	if p.cursor >= len(p.entries) {
		return false
	}
	e := p.entries[p.cursor]
	if len(e.vals) == 0 || e.disabled {
		return false
	}
	i := e.idx + d
	if i < 0 || i >= len(e.vals) {
		return false
	}
	switch e.kind {
	case "rotation":
		a.SetRotation([]gfx.Rotation{gfx.RotRight, gfx.RotNone, gfx.RotLeft}[i])
		if a.cfg.Action != nil {
			a.cfg.Action("rotation", []string{"right", "off", "left"}[i])
		}
	case "scroll":
		a.cfg.Scroll = ScrollValues[i]
	case "hold-delay":
		a.cfg.HoldDelay = []int{200, 300, 500}[i]
	case "remember-sort":
		a.cfg.RememberSort = i == 1
	case "open-at-boot":
		a.cfg.OpenAtBoot = i == 1
	case "return-after-game":
		a.cfg.ReturnAfterGame = i == 1
	case "follow-rotation":
		a.cfg.FollowRotation = i == 1
		if i == 0 && a.cfg.Action != nil {
			a.cfg.Action("rotation", map[gfx.Rotation]string{gfx.RotNone: "off", gfx.RotLeft: "left", gfx.RotRight: "right"}[a.rot])
		}
	case "filter-rotation":
		a.cfg.FilterRotation = i == 1
		a.Refilter()
	case "sources":
		a.cfg.InstalledOnly = i == 1
		a.Refilter()
	case "screensaver":
		a.cfg.Screensaver = saverValues[i]
	case "title-font":
		a.cfg.TitleFont = titleFonts[i]
	case "list-shot":
		a.cfg.ListShot = []string{"gameplay", "title"}[i]
	case "date-format":
		a.cfg.DateFormat = dateFormats[i]
		a.setRotation(a.rot) // the date column width changes the row layout
	case "list-layout":
		a.cfg.ListLayout = listLayouts[i]
		a.setRotation(a.rot)
	case "button-labels":
		a.cfg.ButtonLabels = buttonLabelSets[i]
	case "canvas":
		a.cfg.Canvas = []string{"fit", "320x240"}[i]
	case "menu-button":
		a.cfg.MenuButton = []string{"options", "leave"}[i]
	case "prefetch":
		a.panel.prefetch = i == 1
		if a.cfg.Action != nil {
			a.cfg.Action("prefetch", onOff(a.panel.prefetch))
		}
	case "launcher":
		if a.cfg.Action != nil && a.cfg.Launcher != nil {
			a.cfg.Action("launcher", []string{"off", "on"}[i])
		}
	}
	if a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
	a.buildPanel()
	a.all = true
	return true
}

// togglePanel applies the entry under the cursor.
func (a *App) togglePanel() bool {
	p := &a.panel
	if p.cursor >= len(p.entries) {
		return false
	}
	e := p.entries[p.cursor]
	if a.screen == ScreenFilter && e.header && !e.info && e.kind != "" {
		return a.toggleFilterSection()
	}
	if e.kind == "rot" && a.rotationFilter() != "" {
		return false
	}
	f := a.filters
	off := func(m map[string]bool) map[string]bool {
		out := map[string]bool{}
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	if e.kind == "beta" || (e.kind == "base" && e.value == "Arcade") {
		return a.toggleArcade(false)
	}
	switch e.kind {
	case "year", "decade":
		return a.toggleYears(false)
	case "troubleshooting":
		a.OpenTroubleshooting()
		return true
	case "views":
		a.openViews()
		return true
	case "credits":
		a.openCredits()
		return true
	case "view":
		if e.disabled {
			a.Notice("Keep at least one view on", 3*time.Second)
			return true
		}
		m, _ := data.ParseSort(e.value)
		if !a.setViewOn(m, !e.checked) {
			return false
		}
		a.buildPanel()
		return true
	case "screensaver":
		a.startSaver(a.cfg.TimerNow())
		return true
	case "update-result":
		if a.update.ID == "" {
			a.Notice("No saved update result", 4*time.Second)
		} else {
			a.SetUpdate(a.update, true)
		}
		return true
	case "update":
		a.OpenUpdate()
		return true
	case "clear":
		if e.disabled {
			return false
		}
		a.SetFilters(data.Filters{})
		p.cursor = 0
		a.buildPanel()
		a.skipDisabled() // the row just greyed out; move on to the first heading
		return true
	case "base", "src", "rot", "plr", "genre", "directions", "buttons", "res":
		var m map[string]bool
		var facet map[string]int
		switch e.kind {
		case "base":
			m, facet = off(f.BaseOff), a.ds.Facets.Base
		case "src":
			m, facet = off(f.SrcOff), a.ds.Facets.Src
		case "rot":
			m, facet = off(f.RotOff), a.ds.Facets.Rot
		case "plr":
			m, facet = off(f.PlrOff), a.ds.Facets.Plr
		case "res":
			m, facet = off(f.ResOff), a.ds.Facets.Res
		case "genre":
			m, facet = off(f.GenreOff), a.ds.Facets.Genre
		case "directions":
			m, facet = off(f.DirectionsOff), a.ds.Facets.Directions
		case "buttons":
			m, facet = off(f.ButtonsOff), a.ds.Facets.Buttons
		}
		_ = facet
		if m[e.value] {
			delete(m, e.value)
		} else {
			m[e.value] = true
		}
		switch e.kind {
		case "base":
			f.BaseOff = m
		case "src":
			f.SrcOff = m
		case "rot":
			f.RotOff = m
		case "plr":
			f.PlrOff = m
		case "res":
			f.ResOff = m
		case "genre":
			f.GenreOff = m
		case "directions":
			f.DirectionsOff = m
		case "buttons":
			f.ButtonsOff = m
		}
	case "install":
		if !e.header {
			f.Install = e.value
		}
	case "fav":
		if !e.header {
			f.FavOnly = !f.FavOnly
		}
	case "since":
		if a.seen == nil || a.seen.BaseRows == nil {
			return false
		}
		if !e.header {
			f.Since = !f.Since
		}
	case "rotation", "follow-rotation", "filter-rotation", "sources", "launcher", "scroll", "hold-delay", "remember-sort", "prefetch", "title-font", "list-shot", "date-format", "list-layout", "button-labels":
		return true // Left/Right pick these
	case "inset":
		a.screen = ScreenCalibrate
		a.all = true
		return true
	case "quit":
		if a.cfg.Quit != nil {
			a.cfg.Quit()
		}
		return false
	case "rescan":
		a.OpenScan()
		return true
	case "refresh", "clearimg":
		if a.cfg.Action != nil {
			a.cfg.Action(e.kind, "")
		}
		a.Notice(strings.TrimSuffix(e.text, " now")+gfx.Ellipsis, 3e9)
		return true
	default:
		return false
	}
	a.SetFilters(f)
	a.buildPanel()
	return true
}

// SetPrefetch reflects the host's prefetch setting in the panel.
func (a *App) SetPrefetch(on bool) { a.panel.prefetch = on }

// paintCalibrate draws the safe-zone calibration pattern.
func (a *App) paintCalibrate(c *gfx.Canvas) {
	l := &a.lay
	full := c.Rect
	c.Fill(full, gen.Eva.Bg)
	c.Box(l.Root, gen.Eva.Accent)
	c.Box(l.Root.Inset(4), gen.Eva.Line)
	// crosshair
	cx, cy := full.Dx()/2, full.Dy()/2
	c.HLine(cx-20, cx+20, cy, gen.Eva.Muted)
	c.VLine(cx, cy-20, cy+20, gen.Eva.Muted)
	// the d-pad nudges the top-right corner: a diagonal arrow points at it
	// from inside the frame
	ax, ay := l.Root.Max.X-1, l.Root.Min.Y
	for i := 3; i < 20; i++ { // the shaft, from inside out
		c.Fill(image.Rect(ax-i-1, ay+i-1, ax-i+1, ay+i+1), gen.Eva.Fg)
	}
	for i := 0; i < 8; i++ { // the head, hugging the corner
		c.HLine(ax-8+i, ax-1, ay+i, gen.Eva.Fg)
	}
	lines := []string{
		"Sides: " + itoa(a.cfg.SafeInsetX) + " px",
		"Top/bottom: " + itoa(a.cfg.SafeInsetY) + " px",
		gfx.ArrowLeft + " " + gfx.ArrowRight + " " + gfx.ArrowUp + " " + gfx.ArrowDown + " nudge the corner",
		a.btn("B") + " save and go back",
		"the green frame should sit just",
		"inside the edge of your screen",
	}
	y := min(cy+24, l.Root.Max.Y-4-(len(lines)-1)*(a.sm.H+2)-a.sm.H)
	for _, s := range lines {
		s = gfx.Fit(s, a.sm.Cols(l.Root.Dx()-4))
		c.Text(cx-a.sm.Width(s)/2, y, a.sm, s, gen.Eva.Fg)
		y += a.sm.H + 2
	}
}

func (a *App) actCalibrate(k platform.Key) bool {
	switch k {
	// the corner moves with the d-pad: right and up grow the frame
	case platform.KeyLeft:
		a.SetInset(a.cfg.SafeInsetX+1, a.cfg.SafeInsetY)
	case platform.KeyRight:
		a.SetInset(a.cfg.SafeInsetX-1, a.cfg.SafeInsetY)
	case platform.KeyUp:
		a.SetInset(a.cfg.SafeInsetX, a.cfg.SafeInsetY-1)
	case platform.KeyDown:
		a.SetInset(a.cfg.SafeInsetX, a.cfg.SafeInsetY+1)
	case platform.KeyBack:
		if a.cfg.SettingsChanged != nil {
			a.cfg.SettingsChanged()
		}
		a.openOptions()
	default:
		return false
	}
	a.all = true
	return true
}
