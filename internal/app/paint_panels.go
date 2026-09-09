package app

import (
	"image"
	"sort"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// panelEntry is one line of the Filters or Options screen.
type panelEntry struct {
	text    string
	header  bool
	info    bool     // plain text, never selectable
	help    string   // shown in the help area while selected
	kind    string   // filter section: "base","src","rot","plr","genre","install","fav"; settings: "rotation","inset","prefetch","rescan","refresh","clear","about","settings","back"
	value   string   // facet value for filter entries
	vals    []string // settings: the choices, Left/Right pick one
	idx     int      // settings: the current choice
	checked bool
	count   int
}

type panelState struct {
	entries []panelEntry
	cursor  int
	top     int
	// settings snapshot
	prefetch bool
	lines    int // entries that fit, from the last paint (paging)
}

func (a *App) openPanel(s Screen) {
	a.screen = s
	a.panel.entries = nil
	a.panel.cursor = 0
	a.panel.top = 0
	a.buildPanel()
	a.all = true
}

func (a *App) buildPanel() {
	// remember the entry under the cursor by identity, not by index
	var was *panelEntry
	if a.panel.cursor < len(a.panel.entries) {
		w := a.panel.entries[a.panel.cursor]
		was = &w
	}
	if a.screen == ScreenOptions {
		a.panel.entries = a.optionsEntries()
	} else {
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

func (a *App) filterEntries() []panelEntry {
	f := &a.filters
	var E []panelEntry
	if f.Active() {
		E = append(E, panelEntry{text: "Clear all filters", kind: "clear"})
	}
	E = append(E, panelEntry{text: "On the card", header: true, kind: "install"})
	for _, v := range []struct{ val, text string }{{data.InstallAll, "everything"}, {data.InstallFound, "found on card (any build)"}, {data.InstallCurrent, "current build"}, {data.InstallOlder, "older build than shipped"}, {data.InstallUndated, "build date unknown"}, {data.InstallMissing, "not found on card"}} {
		cur := f.Install
		if cur == "" {
			cur = data.InstallAll
		}
		E = append(E, panelEntry{text: v.text, kind: "install", value: v.val, checked: cur == v.val})
	}
	E = append(E, panelEntry{text: "Favorites", header: true, kind: "fav"})
	E = append(E, panelEntry{text: "favorites only", kind: "fav", checked: f.FavOnly})
	E = append(E, panelEntry{text: "Since last look", header: true, kind: "since"})
	E = append(E, panelEntry{text: "only rows changed since my last look", kind: "since", checked: f.Since})
	section := func(title, kind string, facet map[string]int, off map[string]bool, label func(string) string) {
		E = append(E, panelEntry{text: title, header: true, kind: kind})
		for _, v := range sortedFacet(facet) {
			E = append(E, panelEntry{text: label(v), kind: kind, value: v, checked: !off[v], count: facet[v]})
		}
	}
	ident := func(s string) string { return s }
	section("Type", "base", a.ds.Facets.Base, f.BaseOff, ident)
	section("Source", "src", a.ds.Facets.Src, f.SrcOff, func(s string) string {
		if s == "" {
			return "unknown"
		}
		return data.SrcShort(s)
	})
	section("Rotation", "rot", a.ds.Facets.Rot, f.RotOff, func(s string) string {
		switch s {
		case "h":
			return "Horizontal"
		case "v":
			return "Vertical"
		}
		return "unknown"
	})
	section("Genre", "genre", a.ds.Facets.Genre, f.GenreOff, func(s string) string {
		if s == "" {
			return "No genre"
		}
		return s
	})
	section("Input directions", "directions", a.ds.Facets.Directions, f.DirectionsOff, func(s string) string {
		if s == "" {
			return "not specified"
		}
		return s
	})
	section("Buttons", "buttons", a.ds.Facets.Buttons, f.ButtonsOff, func(s string) string {
		if s == "" {
			return "0 / not specified"
		}
		if s == "1" {
			return "1 button"
		}
		return s + " buttons"
	})
	section("Players", "plr", a.ds.Facets.Plr, f.PlrOff, func(s string) string {
		if s == "" {
			return "unknown"
		}
		return s
	})
	return E
}

func (a *App) optionsEntries() []panelEntry {
	rot := map[gfx.Rotation]string{gfx.RotNone: "off", gfx.RotRight: "turned right", gfx.RotLeft: "turned left"}[a.rot]
	_ = rot
	rotIdx := map[gfx.Rotation]int{gfx.RotNone: 0, gfx.RotRight: 1, gfx.RotLeft: 2}[a.rot]
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
	return []panelEntry{
		{text: "Rotation", kind: "rotation", vals: []string{"off", "turned right", "turned left"}, idx: rotIdx,
			help: "How your monitor is turned. The default follows osd_rotate in MiSTer.ini."},
		{text: "Edit safe zone", kind: "inset",
			help: "Margin kept clear of the screen edge (overscan): now " + itoa(a.cfg.SafeInsetX) + " px at the sides, " + itoa(a.cfg.SafeInsetY) + " px top and bottom. A opens the frame; fit it just inside the picture."},
		{text: "Scroll speed", kind: "scroll", vals: []string{"20 Hz", "30 Hz", "60 Hz"}, idx: scrollIdx,
			help: "How many rows (or pages, with Left/Right) a held direction moves per second. 60 Hz is one row every frame."},
		{text: "Prefetch shots" + a.progressText(), kind: "prefetch", vals: []string{"off", "on"}, idx: prefetchIdx,
			help: "Download every screenshot in the background (about 55 MB) so browsing never waits; the tally counts up as they land. Off: only what you look at."},
		{text: "Main menu launcher", kind: "launcher", vals: []string{"off", "on"}, idx: launcherIdx,
			help: "Show MisterZine in the MiSTer main menu. Off removes the entry when you return to Menu. You can still open MisterZine from Scripts and turn this back on."},
		{text: "Run Update All", kind: "update",
			help: "Update with live output and a stage bar. Browsing waits until it finishes. Hold B for 2 seconds to cancel; system writes finish first. A restart may be required."},
		{text: "Rescan card", kind: "rescan",
			help: "Refresh on-card status after an external update. The built-in Update All rescans automatically when it finishes."},
		{text: "Refresh data now", kind: "refresh",
			help: "Ask misterzine.fyi for new releases right now. The app also checks on launch and every 30 minutes while open; new rows show a notice and their dates in green."},
		{text: "Clear image cache", kind: "clearimg",
			help: "Delete the downloaded screenshots and system photos; they come back as you browse."},
		{text: "Quit MisterZine", kind: "quit",
			help: "Back to the MiSTer menu. The pad's menu button does the same."},
	}
}

func (a *App) launcherText() string {
	if a.cfg.Launcher == nil {
		return "n/a"
	}
	if a.cfg.Launcher() {
		return "on  (A turns off)"
	}
	return "off  (A puts misterzine in the main menu)"
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
	font := a.body
	if a.screen == ScreenFilter {
		font = a.sm
	}
	a.paintStatus(c)
	if a.notice == "" {
		c.Fill(l.Status, gen.Eva.Surface)
		title := "Filters"
		if a.screen == ScreenOptions {
			title = "Options"
		}
		c.Text(l.Status.Min.X+2, l.Status.Min.Y+2, a.sm, title, gen.Eva.Accent)
	}
	box := l.Body
	helpH := 0
	helpLines := 4
	if a.screen == ScreenOptions {
		if !l.Portrait {
			helpLines = 3
		}
		helpH = helpLines*(a.sm.H+1) + 3
	}
	box.Max.Y -= helpH
	c.Fill(l.Body, gen.Eva.Bg)
	c.Box(box, gen.Eva.Line)
	inner := box.Inset(2)
	if a.screen == ScreenOptions {
		// Keep build and data details visible even when the action list scrolls.
		footerY := inner.Max.Y - 2*a.body.H
		cols := a.body.Cols(inner.Dx() - 4)
		c.Text(inner.Min.X+2, footerY, a.body, gfx.Fit("misterzine "+a.cfg.Version, cols), gen.Eva.Muted)
		c.Text(inner.Min.X+2, footerY+a.body.H, a.body, gfx.Fit("data "+a.ds.Updated.Format("2006-01-02 15:04")+"  "+short(a.ds.Hash), cols), gen.Eva.Muted)
		inner.Max.Y = footerY - 3
	}
	lh := font.H
	lines := inner.Dy() / lh
	p := &a.panel
	p.lines = lines
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+lines {
		p.top = p.cursor - lines + 1
	}
	cols := font.Cols(inner.Dx() - 4)
	y := inner.Min.Y
	for n := p.top; n < len(p.entries) && n < p.top+lines; n++ {
		e := p.entries[n]
		r := image.Rect(inner.Min.X, y, inner.Max.X, y+lh)
		if n == p.cursor {
			c.Fill(r, gen.Eva.Surface)
		}
		switch {
		case e.header && e.info:
			c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, cols), gen.Eva.Muted)
		case e.header:
			c.Text(inner.Min.X+2, y, font, gfx.Fit(e.text, cols), gen.Eva.Accent)
		case e.kind == "base" || e.kind == "src" || e.kind == "rot" || e.kind == "plr" || e.kind == "genre" || e.kind == "directions" || e.kind == "buttons" || e.kind == "install" || e.kind == "fav" || e.kind == "since":
			mark := "[ ] "
			if e.checked {
				mark = "[x] "
			}
			text := mark + e.text
			if e.count > 0 {
				text += " (" + itoa(e.count) + ")"
			}
			col := gen.Eva.Fg
			if n == p.cursor {
				col = gen.Eva.Accent
			}
			c.Text(inner.Min.X+2, y, font, gfx.Fit(text, cols), col)
		case len(e.vals) > 0:
			col := gen.Eva.Fg
			if n == p.cursor {
				col = gen.Eva.Accent
			}
			// the value with an arrow on each side that can still move
			left, right := " ", " "
			if e.idx > 0 {
				left = gfx.ArrowLeft
			}
			if e.idx < len(e.vals)-1 {
				right = gfx.ArrowRight
			}
			val := left + " " + e.vals[e.idx] + " " + right
			vw := a.body.Width(val)
			c.Text(inner.Min.X+2, y, a.body, gfx.Fit(e.text, a.body.Cols(inner.Dx()-6-vw)), col)
			c.Text(inner.Max.X-2-vw, y, a.body, val, col)
		default:
			col := gen.Eva.Fg
			if n == p.cursor {
				col = gen.Eva.Accent
			}
			c.Text(inner.Min.X+2, y, a.body, gfx.Fit(e.text, cols), col)
		}
		y += lh
	}
	if a.screen == ScreenOptions {
		// Horizontal uses three help lines; the narrower tate view keeps four.
		if p.cursor < len(p.entries) && p.entries[p.cursor].help != "" {
			hy := box.Max.Y + 2
			for _, ln := range gfx.Wrap(p.entries[p.cursor].help, a.sm.Cols(l.Body.Dx()-4), helpLines) {
				c.Text(l.Body.Min.X+2, hy, a.sm, ln, gen.Eva.Fg)
				hy += a.sm.H + 1
			}
		}
		a.paintHint(c, gfx.ArrowLeft+" "+gfx.ArrowRight+" change  A open  B back")
	} else {
		a.paintHint(c, "A toggle  "+gfx.ArrowLeft+" "+gfx.ArrowRight+" page  B back")
	}
}

func (a *App) actPanel(k platform.Key) bool {
	p := &a.panel
	n := len(p.entries)
	selectable := func(i int) bool {
		return i >= 0 && i < n && !p.entries[i].info && !(p.entries[i].header && p.entries[i].text == "")
	}
	switch k {
	case platform.KeyBack:
		a.screen = ScreenList
		a.all = true
		return true
	case platform.KeyUp:
		for i := p.cursor - 1; i >= 0; i-- {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyDown:
		for i := p.cursor + 1; i < n; i++ {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyLeft:
		if a.screen == ScreenOptions {
			return a.stepValue(-1)
		}
		// the last entry of the previous page, shown at the bottom
		p.cursor = p.nearest(p.top-1, selectable)
		p.top = p.cursor - max(p.lines, 1) + 1
		if p.top < 0 {
			p.top = 0
		}
	case platform.KeyRight:
		if a.screen == ScreenOptions {
			return a.stepValue(1)
		}
		// the first entry of the next page, shown at the top
		p.cursor = p.nearest(p.top+max(p.lines, 1), selectable)
		p.top = p.cursor
		if p.top > n-max(p.lines, 1) {
			p.top = max(n-max(p.lines, 1), 0)
		}
	case platform.KeyPageUp, platform.KeyHome:
		for i := 0; i < n; i++ {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyPageDown, platform.KeyEnd:
		for i := n - 1; i >= 0; i-- {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyEnter, platform.KeySpace:
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
	if len(e.vals) == 0 {
		return false
	}
	i := e.idx + d
	if i < 0 || i >= len(e.vals) {
		return false
	}
	switch e.kind {
	case "rotation":
		a.SetRotation([]gfx.Rotation{gfx.RotNone, gfx.RotRight, gfx.RotLeft}[i])
		if a.cfg.Action != nil {
			a.cfg.Action("rotation", []string{"off", "right", "left"}[i])
		}
	case "scroll":
		a.cfg.Scroll = ScrollValues[i]
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
	f := a.filters
	off := func(m map[string]bool) map[string]bool {
		out := map[string]bool{}
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	switch e.kind {
	case "update":
		a.OpenUpdate()
		return true
	case "clear":
		a.SetFilters(data.Filters{})
		p.cursor = 0
		a.buildPanel()
		return true
	case "base", "src", "rot", "plr", "genre", "directions", "buttons":
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
		case "genre":
			m, facet = off(f.GenreOff), a.ds.Facets.Genre
		case "directions":
			m, facet = off(f.DirectionsOff), a.ds.Facets.Directions
		case "buttons":
			m, facet = off(f.ButtonsOff), a.ds.Facets.Buttons
		}
		if e.header {
			// a header toggles its whole section: all on, or all off when already all on
			allOn := len(m) == 0
			m = map[string]bool{}
			if allOn {
				for v := range facet {
					m[v] = true
				}
			}
		} else if m[e.value] {
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
		if !e.header {
			f.Since = !f.Since
		}
	case "rotation", "launcher", "scroll", "prefetch":
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
	case "rescan", "refresh", "clearimg":
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
		"Safe zone: sides " + itoa(a.cfg.SafeInsetX) + " px, top/bottom " + itoa(a.cfg.SafeInsetY) + " px",
		gfx.ArrowLeft + " " + gfx.ArrowRight + " " + gfx.ArrowUp + " " + gfx.ArrowDown + " nudge the corner",
		"B save and go back",
		"the green frame should sit just",
		"inside the edge of your screen",
	}
	y := cy + 30
	for _, s := range lines {
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
		a.openPanel(ScreenOptions)
	default:
		return false
	}
	a.all = true
	return true
}
