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

// panelEntry is one line of the filter or settings panel.
type panelEntry struct {
	text    string
	header  bool
	info    bool   // plain text, never selectable
	help    string // shown in the help area while selected
	kind    string // filter section: "base","src","rot","plr","genre","install","fav"; settings: "rotation","inset","prefetch","rescan","refresh","clear","about","settings","back"
	value   string // facet value for filter entries
	checked bool
	count   int
}

type panelState struct {
	entries []panelEntry
	cursor  int
	top     int
	// settings snapshot
	prefetch bool
	insetWas int // inset before calibration, for cancel
}

func (a *App) openPanel(s Screen) {
	a.screen = s
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
	if a.screen == ScreenSettings {
		a.panel.entries = a.settingsEntries()
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
	E = append(E, panelEntry{text: "Settings...  (safe zone, rotation)", kind: "settings"})
	if f.Active() {
		E = append(E, panelEntry{text: "Clear all filters", kind: "clear"})
	}
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
	section("Players", "plr", a.ds.Facets.Plr, f.PlrOff, func(s string) string {
		if s == "" {
			return "unknown"
		}
		return s
	})
	section("Genre", "genre", a.ds.Facets.Genre, f.GenreOff, func(s string) string {
		if s == "" {
			return "No genre"
		}
		return s
	})
	E = append(E, panelEntry{text: "Since last look", header: true, kind: "since"})
	E = append(E, panelEntry{text: "only rows changed since my last look", kind: "since", checked: f.Since})
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
	return E
}

func (a *App) settingsEntries() []panelEntry {
	rot := map[gfx.Rotation]string{gfx.RotNone: "off", gfx.RotRight: "turned right", gfx.RotLeft: "turned left"}[a.rot]
	launcher := "n/a"
	if a.cfg.Launcher != nil {
		launcher = onOff(a.cfg.Launcher())
	}
	return []panelEntry{
		{text: "Settings", header: true, info: true},
		{text: "Rotation: " + rot, kind: "rotation",
			help: "How your monitor is turned. Auto follows osd_rotate in MiSTer.ini. A cycles off / right / left."},
		{text: "Safe zone: " + itoa(a.cfg.SafeInset) + " px", kind: "inset",
			help: "Margin kept clear of the screen edge (overscan). A opens the calibration frame; fit it just inside the picture."},
		{text: "Scroll speed: " + a.scrollText(), kind: "scroll",
			help: "How fast a held Up/Down (rows) or Left/Right (pages) moves through the list. Normal is 20 rows a second, fast 30, turbo 60: one row every frame."},
		{text: "Prefetch all shots: " + onOff(a.panel.prefetch) + a.progressText(), kind: "prefetch",
			help: "Download every screenshot in the background (about 55 MB) so browsing never waits. Off: only what you look at."},
		{text: "Main menu launcher: " + launcher, kind: "launcher",
			help: "Puts a MisterZine entry in the MiSTer main menu, next to Arcade and Console. Adds one line to linux/user-startup.sh and ships MisterZine.mgl. Off removes both; the Scripts menu entry keeps working."},
		{text: "Rescan card", kind: "rescan",
			help: "Re-read which cores and MRAs are on the card. Do this after running update_all."},
		{text: "Refresh data now", kind: "refresh",
			help: "Ask misterzine.fyi for new releases right now. The app also checks on launch and every 30 minutes while open; new rows show a notice and their dates in green."},
		{text: "Clear image cache", kind: "clearimg",
			help: "Delete the downloaded screenshots and system photos; they come back as you browse."},
		{text: "Input test", kind: "inputtest",
			help: "Shows every press and release the app receives, with timing, to debug a pad or keyboard encoder."},
		{text: "Quit misterzine", kind: "quit",
			help: "Back to the MiSTer menu. The pad's menu button does the same."},
		{text: "", header: true, info: true},
		{text: "misterzine " + a.cfg.Version, header: true, info: true},
		{text: "data " + a.ds.Updated.Format("2006-01-02 15:04") + "  " + short(a.ds.Hash), header: true, info: true},
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
		return "fast"
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
	return "  " + itoa(have) + "/" + itoa(total) + " on card"
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
	box := l.Body
	helpH := 0
	if a.screen == ScreenSettings {
		helpH = 4*(a.sm.H+1) + 3
	}
	box.Max.Y -= helpH
	c.Fill(l.Body, gen.Eva.Bg)
	c.Box(box, gen.Eva.Line)
	inner := box.Inset(2)
	lh := a.body.H
	lines := inner.Dy() / lh
	p := &a.panel
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+lines {
		p.top = p.cursor - lines + 1
	}
	cols := a.body.Cols(inner.Dx() - 4)
	y := inner.Min.Y
	for n := p.top; n < len(p.entries) && n < p.top+lines; n++ {
		e := p.entries[n]
		r := image.Rect(inner.Min.X, y, inner.Max.X, y+lh)
		if n == p.cursor {
			c.Fill(r, gen.Eva.Surface)
		}
		switch {
		case e.header && e.info:
			c.Text(inner.Min.X+2, y, a.body, gfx.Fit(e.text, cols), gen.Eva.Muted)
		case e.header:
			c.Text(inner.Min.X+2, y, a.body, gfx.Fit(e.text, cols), gen.Eva.Accent)
		case e.kind == "base" || e.kind == "src" || e.kind == "rot" || e.kind == "plr" || e.kind == "genre" || e.kind == "install" || e.kind == "fav" || e.kind == "since":
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
			c.Text(inner.Min.X+2, y, a.body, gfx.Fit(text, cols), col)
		default:
			col := gen.Eva.Fg
			if n == p.cursor {
				col = gen.Eva.Accent
			}
			c.Text(inner.Min.X+2, y, a.body, gfx.Fit(e.text, cols), col)
		}
		y += lh
	}
	if a.screen == ScreenSettings {
		// help for the selected item, wrapped to four small lines
		if p.cursor < len(p.entries) && p.entries[p.cursor].help != "" {
			hy := box.Max.Y + 2
			for _, ln := range gfx.Wrap(p.entries[p.cursor].help, a.sm.Cols(l.Body.Dx()-4), 4) {
				c.Text(l.Body.Min.X+2, hy, a.sm, ln, gen.Eva.Fg)
				hy += a.sm.H + 1
			}
		}
		a.paintHint(c, "A change  B back to filters")
	} else {
		a.paintHint(c, "A toggle  "+gfx.ArrowLeft+" "+gfx.ArrowRight+" section  X close  Settings at top")
	}
}

func (a *App) actPanel(k platform.Key) bool {
	p := &a.panel
	n := len(p.entries)
	selectable := func(i int) bool {
		return i >= 0 && i < n && !p.entries[i].info && !(p.entries[i].header && p.entries[i].text == "")
	}
	switch k {
	case platform.KeyBack, platform.KeyTab:
		if a.screen == ScreenSettings {
			a.openPanel(ScreenFilter)
			return true
		}
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
	case platform.KeyLeft, platform.KeyPageUp:
		// previous section header (headers are selectable: A on one = all)
		for i := p.cursor - 1; i >= 0; i-- {
			if p.entries[i].header && p.entries[i].text != "" {
				p.cursor = i
				break
			}
		}
	case platform.KeyRight, platform.KeyPageDown:
		for i := p.cursor + 1; i < n; i++ {
			if p.entries[i].header && p.entries[i].text != "" {
				p.cursor = i
				break
			}
		}
	case platform.KeyHome:
		for i := 0; i < n; i++ {
			if selectable(i) {
				p.cursor = i
				break
			}
		}
	case platform.KeyEnd:
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
	case "settings":
		a.panel.cursor = 0
		a.openPanel(ScreenSettings)
		return true
	case "clear":
		a.SetFilters(data.Filters{})
		p.cursor = 0
		a.buildPanel()
		return true
	case "base", "src", "rot", "plr", "genre":
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
	case "rotation":
		a.SetRotation((a.rot + 1) % 3)
		if a.cfg.SettingsChanged != nil {
			a.cfg.SettingsChanged()
		}
		a.buildPanel()
		return true
	case "inset":
		a.panel.insetWas = a.cfg.SafeInset
		a.screen = ScreenCalibrate
		a.all = true
		return true
	case "launcher":
		if a.cfg.Action != nil && a.cfg.Launcher != nil {
			if a.cfg.Launcher() {
				a.cfg.Action("launcher", "off")
			} else {
				a.cfg.Action("launcher", "on")
			}
		}
		a.buildPanel()
		return true
	case "scroll":
		order := []string{"normal", "fast", "turbo"}
		cur := a.scrollText()
		for i, v := range order {
			if v == cur {
				a.cfg.Scroll = order[(i+1)%len(order)]
				break
			}
		}
		if a.cfg.SettingsChanged != nil {
			a.cfg.SettingsChanged()
		}
		a.buildPanel()
		return true
	case "inputtest":
		a.screen = ScreenInput
		a.all = true
		return true
	case "quit":
		if a.cfg.Quit != nil {
			a.cfg.Quit()
		}
		return false
	case "prefetch":
		a.panel.prefetch = !a.panel.prefetch
		if a.cfg.Action != nil {
			a.cfg.Action("prefetch", onOff(a.panel.prefetch))
		}
		a.buildPanel()
		return true
	case "rescan", "refresh", "clearimg":
		if a.cfg.Action != nil {
			a.cfg.Action(e.kind, "")
		}
		a.Notice(strings.TrimSuffix(e.text, " now")+"..", 3e9)
		return true
	case "back":
		a.openPanel(ScreenFilter)
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
	// a big UP so rotation is verifiable
	c.Text(cx-a.body.Width("UP")/2, l.Root.Min.Y+8, a.body, "UP", gen.Eva.Fg)
	for y := 0; y < 10; y++ {
		c.HLine(cx-y, cx+y, l.Root.Min.Y+24+y, gen.Eva.Fg)
	}
	lines := []string{
		"Safe zone: " + itoa(a.cfg.SafeInset) + " px (0-32)",
		gfx.ArrowLeft + " " + gfx.ArrowRight + " adjust, A save, B cancel",
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
	case platform.KeyLeft:
		a.SetInset(a.cfg.SafeInset - 1)
	case platform.KeyRight:
		a.SetInset(a.cfg.SafeInset + 1)
	case platform.KeyEnter:
		if a.cfg.SettingsChanged != nil {
			a.cfg.SettingsChanged()
		}
		a.openPanel(ScreenSettings)
	case platform.KeyBack:
		a.SetInset(a.panel.insetWas)
		a.openPanel(ScreenSettings)
	default:
		return false
	}
	a.all = true
	return true
}
