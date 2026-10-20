package app

import (
	"image"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Umbrella lines are the list's form of the site's same-game groups
// (data.Fold): one cursor stop for a game the catalogue ships on two cores
// or more, A opening it onto its members and closing it again. The view
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
// umbrella's data.GroupKey.
func (a *App) keyAt(pos int) string {
	if u := a.umbrellaAt(pos); u != nil {
		return data.GroupKey(u.G)
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

// setExpanded opens or closes the umbrella of group g.
func (a *App) setExpanded(g string, open bool) {
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

// paintCountedTitle draws an umbrella's name into w pixels with its member
// count right after it, "(2)", in the title's font but muted, when Options
// -> Version counts is on; a long name then gives way to the count.
func (a *App) paintCountedTitle(c *gfx.Canvas, x, y, w int, u *data.Group, col rgb) {
	count := "(" + itoa(u.Versions) + ")"
	title := u.Title(a.ds)
	if f := a.titleFont(); f != nil {
		y-- // scientifica's baseline is one pixel lower than the body font's
		if !a.VersionCounts() {
			c.TextProp(x, y, f, gfx.FitProp(f, title, w), col)
			return
		}
		gap := f.Advance(' ')
		cw := f.PropWidth(count)
		tw := c.TextProp(x, y, f, gfx.FitProp(f, title, w-gap-cw), col)
		c.TextProp(x+tw+gap, y, f, count, pal.Muted)
		return
	}
	if !a.VersionCounts() {
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
