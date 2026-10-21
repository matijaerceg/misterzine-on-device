package app

import (
	"image"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Umbrella lines are the list's form of the site's same-game groups
// (data.Fold): one cursor stop for a game the catalogue ships on two cores
// or more, A opening it onto its members and closing it again. Core batches
// (data/batches.go) use the same lines for the games one core build or
// debut brought together, under the date orders. The view
// stays the one list of lines, an umbrella sitting in it as an index past
// the rows (len(ds.Rows)+n for groups[n]); everything that walks the view
// asks the helpers below for a line's key, the row that places it in the
// order and the row whose picture it shows. An umbrella has no Details, no
// star and nothing to launch: those stay on the members.

// umbrellaAt is the umbrella on view line pos, nil for a row.
func (a *App) umbrellaAt(pos int) *data.Group {
	if pos < 0 || pos >= len(a.view) {
		return nil
	}
	if n := a.view[pos] - len(a.ds.Rows); n >= 0 && n < len(a.groups) {
		return &a.groups[n]
	}
	return nil
}

// memberAt reports whether view line pos is a member shown under its open
// umbrella: the members follow the umbrella line in order, so pos is one
// while the nearest umbrella above it is open and pos is within its span.
func (a *App) memberAt(pos int) bool {
	for p := pos - 1; p >= 0; p-- {
		if u := a.umbrellaAt(p); u != nil {
			return u.Open && pos-p <= len(u.Members)
		}
	}
	return false
}

// keyAt is the key line pos is remembered by: the row's K, or the
// umbrella's Key.
func (a *App) keyAt(pos int) string {
	if u := a.umbrellaAt(pos); u != nil {
		return u.Key()
	}
	return a.ds.Rows[a.view[pos]].K
}

// placeRow is the row whose values place line pos in the order: an
// umbrella's anchor member, so it heads and jumps with the group it sorts in.
func (a *App) placeRow(pos int) int {
	if u := a.umbrellaAt(pos); u != nil {
		return u.Anchor
	}
	return a.view[pos]
}

// previewRow is the row whose picture line pos shows.
func (a *App) previewRow(pos int) int {
	if u := a.umbrellaAt(pos); u != nil {
		return u.Preview(a.ds)
	}
	return a.view[pos]
}

// paneRow is current for the picture: an umbrella's preview member.
func (a *App) paneRow() *data.Row {
	if a.cursor >= len(a.view) {
		return nil
	}
	return &a.ds.Rows[a.previewRow(a.cursor)]
}

// searching reports whether a title search narrows the list.
func (a *App) searching() bool { return searchText(a.query) != "" }

// umbrellaOpen reports whether the umbrella of group g is expanded. A
// search opens every core batch, whose name is its core's and would
// otherwise hide the games that matched, until one is closed by hand; the
// open batches the player chose wait in expanded for the search to end.
func (a *App) umbrellaOpen(g string) bool {
	if data.IsBatch(g) && a.searching() {
		return !a.searchShut[g]
	}
	return a.expanded[g]
}

// setExpanded opens or closes the umbrella of group g.
func (a *App) setExpanded(g string, open bool) {
	if data.IsBatch(g) && a.searching() {
		if open {
			delete(a.searchShut, g)
			return
		}
		if a.searchShut == nil {
			a.searchShut = map[string]bool{}
		}
		a.searchShut[g] = true
		return
	}
	if !open {
		delete(a.expanded, g)
		return
	}
	if a.expanded == nil {
		a.expanded = map[string]bool{}
	}
	a.expanded[g] = true
}

// foldedUnder is the group of a closed umbrella in view whose members
// include the row keyed k, "" when none does.
func (a *App) foldedUnder(k string) string {
	for n := range a.groups {
		u := &a.groups[n]
		if u.Open {
			continue
		}
		for _, i := range u.Members {
			if a.ds.Rows[i].K == k {
				return u.G
			}
		}
	}
	return ""
}

// umbrellaUnseen reports whether a closed umbrella hides a member added or
// rebuilt since the last visit: its date then takes the accent, as each
// member's would. Open, the members show their own.
func (a *App) umbrellaUnseen(u *data.Group) bool {
	if u.Open || a.seen == nil {
		return false
	}
	for _, i := range u.Members {
		if a.seen.Unseen(&a.ds.Rows[i]) {
			return true
		}
	}
	return false
}

// versionsChip is the umbrella's member count as the pane shows it.
func versionsChip(u *data.Group) string {
	return itoa(u.Versions) + " versions"
}

// umbrellaTail is what follows an umbrella's name, muted: a core batch's
// "update (22)" always, "(3/22)" when a search shows only some of its games,
// and a same-game group's "(2)" while Options -> Version counts is on.
func (a *App) umbrellaTail(u *data.Group) string {
	if u.Batch() {
		n := itoa(u.Versions)
		if u.Total > u.Versions {
			n += "/" + itoa(u.Total)
		}
		return a.batchWord(u) + " (" + n + ")"
	}
	if a.VersionCounts() {
		return "(" + itoa(u.Versions) + ")"
	}
	return ""
}

// batchWord says what a core batch is: its games' debut under the debut
// order, and under the build order too when every game's build is its debut
// (a new core's first wave, as the site reads it); else an update.
func (a *App) batchWord(u *data.Group) string {
	if a.mode == data.SortDebut {
		return "debut"
	}
	for _, i := range u.Members {
		if r := &a.ds.Rows[a.ds.SortRow(i)]; r.Updated != r.Date {
			return "update"
		}
	}
	return "debut"
}

// paintCountedTitle draws an umbrella's name into w pixels with its tail
// (umbrellaTail) right after it in the title's font but muted; a long name
// gives way to the tail.
func (a *App) paintCountedTitle(c *gfx.Canvas, x, y, w int, u *data.Group, col rgb) {
	count := a.umbrellaTail(u)
	title := u.Title(a.ds)
	if f := a.titleFont(); f != nil {
		y-- // scientifica's baseline is one pixel lower than the body font's
		if count == "" {
			c.TextProp(x, y, f, gfx.FitProp(f, title, w), col)
			return
		}
		gap := f.Advance(' ')
		cw := f.PropWidth(count)
		tw := c.TextProp(x, y, f, gfx.FitProp(f, title, w-gap-cw), col)
		c.TextProp(x+tw+gap, y, f, count, pal.Muted)
		return
	}
	if count == "" {
		c.Text(x, y, a.body, gfx.Fit(title, a.body.Cols(w)), col)
		return
	}
	cols := a.body.Cols(w) - len(count) - 1
	s := gfx.Fit(title, cols)
	c.Text(x, y, a.body, s, col)
	c.Text(x+a.body.Width(s)+a.body.W, y, a.body, count, pal.Muted)
}

// umbrellaMark is the sign in an umbrella's star column: a plus while it is
// closed, a minus once open. Not an arrow, which reads as a pointer at the
// selected row.
func umbrellaMark(open bool) string {
	if open {
		return gfx.Collapse
	}
	return gfx.Expand
}

// paintUmbrella draws an umbrella line: the plus or minus in the star's column,
// the game's name with its core count, and the group's date (the newest
// build, or the first debut) or the original year, as rows show theirs.
func (a *App) paintUmbrella(c *gfx.Canvas, r image.Rectangle, pos int, u *data.Group) {
	l := &a.lay
	selected := pos == a.cursor
	if selected && a.listMotion.offset == 0 {
		c.Fill(r, pal.Surface)
	}
	x, y := r.Min.X, r.Min.Y
	c.Text(x, y, a.body, umbrellaMark(u.Open), pal.Muted)
	x += a.body.W
	titleCol := pal.Fg
	if selected {
		titleCol = pal.Accent
	}
	rf := a.rowFont()
	tw := l.TitleW
	if a.mode == data.SortYear && !l.PictureColumns {
		tw += a.dateCols() * rf.W
	}
	// the count follows the name, "(2)", and the name takes the rest
	if fit := tw + a.restLayout().TitleW - l.TitleW; fit != tw {
		col := &gfx.Canvas{RGBA: c.Sub(image.Rect(x, c.Rect.Min.Y, x+tw, c.Rect.Max.Y))}
		a.paintCountedTitle(col, x, y, fit, u, titleCol)
	} else {
		a.paintCountedTitle(c, x, y, tw, u, titleCol)
	}
	if l.PictureColumns || a.mode == data.SortYear {
		return
	}
	text := ""
	switch a.mode {
	case data.SortDebut:
		text = a.dateCol(u.Date(a.ds))
	case data.SortMaker:
		text = gfx.Fit(a.ds.Rows[u.Anchor].Year, a.dateCols())
		for len(text) < a.dateCols() {
			text = " " + text
		}
	default:
		text = a.dateCol(u.Updated(a.ds))
	}
	dateCol := pal.Muted
	if a.umbrellaUnseen(u) {
		dateCol = pal.Accent
	}
	c.Text(r.Max.X-a.dateCols()*rf.W, y-1, rf, text, dateCol)
}

// umbrellaPaneLines is what the pane says about an umbrella: its name, the
// version count, each member's core, and the year and maker when the members
// agree on them.
func (a *App) umbrellaPaneLines(u *data.Group, cols int) []paneLine {
	if u.Batch() {
		return a.batchPaneLines(u, cols)
	}
	var lines []paneLine
	for _, t := range gfx.Wrap(u.Title(a.ds), cols, 2) {
		lines = append(lines, paneLine{t, pal.Fg})
	}
	lines = append(lines, paneLine{"Arcade / " + versionsChip(u), pal.TypeArcade})
	// each member's core, by its source where the core is named after the
	// game, as the site's Core cell has it
	for _, i := range u.Members {
		label := a.ds.Der[i].CoreLabel
		if label == "" || label == a.ds.Rows[i].Title {
			label = a.ds.Der[i].SrcShort
		}
		lines = append(lines, paneLine{label, pal.Muted})
	}
	first := &a.ds.Rows[u.Members[0]]
	year, maker := first.Year, first.Manufacturer
	for _, i := range u.Members[1:] {
		if a.ds.Rows[i].Year != year {
			year = ""
		}
		if a.ds.Rows[i].Manufacturer != maker {
			maker = ""
		}
	}
	if s := year + " " + data.ASCII(maker); s != " " {
		lines = append(lines, paneLine{strings.TrimSpace(s), pal.Fg})
	}
	return lines
}

// batchPaneLines is what the pane says about a core batch: its core, its
// type and source, how many games shipped together and on which date, then
// the games themselves as the list shows them.
func (a *App) batchPaneLines(u *data.Group, cols int) []paneLine {
	var lines []paneLine
	for _, t := range gfx.Wrap(u.Title(a.ds), cols, 2) {
		lines = append(lines, paneLine{t, pal.Fg})
	}
	first := a.ds.SortRow(u.Anchor)
	kind := a.ds.Der[first].TypeLabel
	if r := &a.ds.Rows[first]; r.IsArcade() {
		kind = "Arcade"
		src := r.Src
		for _, i := range u.Members {
			if a.ds.Rows[a.ds.SortRow(i)].Src != src {
				src = ""
			}
		}
		if src != "" {
			kind += " / " + a.ds.Der[first].SrcShort
		}
	}
	lines = append(lines, paneLine{kind, typeHue(a.ds.Rows[first].Base)})
	games := itoa(u.Versions) + " games"
	if u.Total > u.Versions {
		games = itoa(u.Versions) + " of " + itoa(u.Total) + " games"
	}
	lines = append(lines, paneLine{games, pal.Fg})
	day := "Build " + u.Day
	if a.mode == data.SortDebut {
		day = "Debut " + u.Day
	}
	lines = append(lines, paneLine{day, pal.Fg})
	for _, i := range u.Members {
		if a.ds.SortRow(i) == i {
			lines = append(lines, paneLine{a.ds.Der[i].Title, pal.Muted})
		}
	}
	return lines
}
