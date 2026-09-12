package app

import (
	"image"
	"image/color"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

type rgb = color.RGBA

// paintList draws the status bar, the list, the pane and the hint bar.
func (a *App) paintList(c *gfx.Canvas) {
	a.paintStatus(c)
	a.paintRows(c)
	a.paintScrollbar(c)
	a.paintPane(c)
	hint := "A details  B Options  X Filters  Y view"
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		hint = "A open  B opts  X filt  Y mode"
	}
	if a.query != "" {
		hint = "A details  B clear find  X Filters"
	}
	if a.quickHeld() {
		hint = "Y layout  X shots"
	}
	a.paintHint(c, hint)
}

func (a *App) paintStatus(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Status, gen.Eva.Surface)
	c.HLine(l.Status.Min.X, l.Status.Max.X-1, l.Status.Max.Y-1, gen.Eva.Muted)
	y := l.Status.Min.Y + 2
	if a.notice != "" {
		// Notices (including the month/year after a jump) remain visible while
		// searching; the query/count comes back when the notice expires.
		c.Text(l.Status.Min.X+2, y, a.sm, gfx.Fit(a.notice, a.sm.Cols(l.Status.Dx()-4)), gen.Eva.Fg)
		return
	}
	if a.query != "" {
		count := itoa(len(a.view)) + " matches"
		c.TextRight(l.Status.Max.X-2, y, a.sm, count, gen.Eva.Muted)
		cols := a.sm.Cols(l.Status.Dx()-8-a.sm.Width(count)) - len("Find: ") - 1
		query := a.query
		if len(query) > cols {
			query = query[len(query)-max(0, cols):]
		}
		c.Text(l.Status.Min.X+2, y, a.sm, "Find: "+query+"_", gen.Eva.Accent)
		return
	}
	left := "by: core updated"
	if a.mode == data.SortDebut {
		left = "by: MiSTer debut"
	} else if a.mode == data.SortYear {
		left = "by: original year"
	} else if a.mode == data.SortAlphabetical {
		left = "by: A-Z"
	} else if a.mode == data.SortMaker {
		left = "by: maker"
	} else if a.mode == data.SortFavorites {
		left = "Favorites A-Z"
	} else if a.mode == data.SortRecents {
		left = "Recents"
	}
	if a.appUpdate != "" {
		// Reserve space for a persistent app notice even on narrow tate screens.
		left = map[data.SortMode]string{data.SortUpdated: "Core updated", data.SortDebut: "MiSTer debut", data.SortYear: "Original year", data.SortAlphabetical: "A-Z", data.SortMaker: "Maker A-Z", data.SortFavorites: "Favorites A-Z", data.SortRecents: "Recents"}[a.mode]
		c.Text(l.Status.Min.X+2, y, a.sm, left, gen.Eva.Accent)
		c.TextRight(l.Status.Max.X-2, y, a.sm, "App update", gen.Eva.Accent)
		return
	}
	count := itoa(a.total) + " releases"
	if a.filtersActive() || a.mode == data.SortFavorites || a.mode == data.SortRecents {
		count = itoa(len(a.view)) + " of " + itoa(a.total) + " releases"
	}
	c.Text(l.Status.Min.X+2, y, a.sm, left, gen.Eva.Accent)
	x := l.Status.Min.X + 2 + a.sm.Width(left) + a.sm.W*2
	if a.sm.Width(count) > l.Status.Max.X-2-x {
		count = itoa(a.total)
		if a.filtersActive() || a.mode == data.SortFavorites || a.mode == data.SortRecents {
			count = itoa(len(a.view)) + "/" + count
		}
	}
	count = gfx.Fit(count, a.sm.Cols(l.Status.Max.X-2-x))
	c.Text(x, y, a.sm, count, gen.Eva.Muted)
	left += "  " + count
	if a.net != "" {
		c.TextRight(l.Status.Max.X-2, y, a.sm, gfx.Fit(a.net, a.sm.Cols(l.Status.Dx()-a.sm.Width(left)-8)), gen.Eva.Fg)
	}
}

// paintHint draws the bottom hint bar. Chunks are separated by two spaces;
// the first word of each chunk is the button and paints in the accent.
func (a *App) paintHint(c *gfx.Canvas, s string) {
	l := &a.lay
	c.Fill(l.Hint, gen.Eva.Surface)
	a.hintLine(c, l.Hint.Min.X+2, l.Hint.Min.Y+2, l.Hint.Dx()-4, s)
}

func (a *App) hintLine(c *gfx.Canvas, x, y, w int, s string) {
	maxX := x + w
	for _, chunk := range strings.Split(s, "  ") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		btn, rest, _ := strings.Cut(chunk, " ")
		if btn == "Hold" {
			button, tail, _ := strings.Cut(rest, " ")
			btn, rest = btn+" "+button, tail
		}
		// arrows right after the button belong to it: a pair of arrows is
		// one button, and "A < > open/close" names two controls for one action
		for {
			b2, r2, ok := strings.Cut(rest, " ")
			if !ok || !isArrow(b2) || !(isArrow(btn[len(btn)-1:]) || btn == "A") {
				break
			}
			btn, rest = btn+" "+b2, r2
		}
		if x+a.sm.Width(btn+" "+rest) > maxX {
			break
		}
		x += c.Text(x, y, a.sm, a.btn(btn), gen.Eva.Accent)
		x += c.Text(x, y, a.sm, " "+rest+"  ", gen.Eva.Muted)
	}
}

func isArrow(s string) bool {
	return s == gfx.ArrowUp || s == gfx.ArrowDown || s == gfx.ArrowLeft || s == gfx.ArrowRight
}

func (a *App) emptyListMessage() string {
	if a.mode == data.SortFavorites && len(a.cfg.Favorites) == 0 {
		return "No favorites yet"
	}
	if a.mode == data.SortRecents && len(a.cfg.RecentLaunches) == 0 {
		return "No launches yet"
	}
	if a.query != "" {
		return "no matches, " + a.btn("B") + ": clear find"
	}
	if a.filters.Since {
		if a.seen == nil || a.seen.BaseRows == nil {
			return "no previous visit yet"
		}
		other := a.filters
		other.Since = false
		if !other.Active() {
			return "No changed matches"
		}
	}
	return "no rows match, " + a.btn("X") + ": filters"
}

// paintRows draws the visible list lines: rows and the marker lines
// between them (the last-look divider, the maker headers).
func (a *App) paintRows(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.List, gen.Eva.Bg)
	if len(a.view) == 0 {
		c.Text(l.List.Min.X+a.body.W, l.List.Min.Y+l.Line, a.body, gfx.Fit(a.emptyListMessage(), l.Cols-1), gen.Eva.Muted)
		return
	}
	// the first row and the first marker at or below the top line
	pos, mk := 0, 0
	for pos < len(a.view) && a.screenLine(pos) < a.top {
		pos++
	}
	for mk < len(a.marks) && (a.marks[mk] < pos || a.markLine(mk) < a.top) {
		mk++
	}
	for n := 0; n < l.Lines; n++ {
		r := l.lineRect(n)
		if mk < len(a.marks) && a.marks[mk] == pos {
			a.paintMarker(c, r, a.markText(mk))
			mk++
			continue
		}
		if pos >= len(a.view) {
			break
		}
		a.paintRow(c, r, pos)
		pos++
	}
	// a maker whose header scrolled off keeps its name on the top line
	if h := a.pinnedHeader(); h != "" {
		r := l.lineRect(0)
		c.Fill(r, gen.Eva.Bg)
		a.paintMarker(c, r, h)
	}
}

// paintScrollbar draws the track beside the list with a thumb sized to the
// visible share of the lines and placed at the scroll position.
func (a *App) paintScrollbar(c *gfx.Canvas) {
	l := &a.lay
	t := l.Scroll
	if t.Empty() {
		return
	}
	total := a.totalLines()
	if total <= l.Lines {
		return
	}
	h := t.Dy() * l.Lines / total
	if h < 4 {
		h = 4
	}
	// A group jump can put the last group at the top with blank rows below.
	// Its thumb still stops at the end of the track.
	y := t.Min.Y + (t.Dy()-h)*min(a.top, total-l.Lines)/(total-l.Lines)
	c.Fill(image.Rect(t.Min.X, y, t.Max.X, y+h), gen.Eva.Accent)
}

func (a *App) paintMarker(c *gfx.Canvas, r image.Rectangle, text string) {
	mid := r.Min.Y + r.Dy()/2
	c.HLine(r.Min.X, r.Max.X-1, mid, gen.Eva.Line)
	s := " " + gfx.Fit(text, a.sm.Cols(r.Dx())-2) + " "
	x := r.Min.X + a.body.W
	c.Fill(image.Rect(x, r.Min.Y, x+a.sm.Width(s), r.Max.Y), gen.Eva.Bg)
	c.Text(x, r.Min.Y+2, a.sm, s, gen.Eva.Muted)
}

func (a *App) paintRow(c *gfx.Canvas, r image.Rectangle, pos int) {
	l := &a.lay
	i := a.view[pos]
	row := &a.ds.Rows[i]
	d := &a.ds.Der[i]
	selected := pos == a.cursor
	if selected {
		c.Fill(r, gen.Eva.Surface)
	}
	x := r.Min.X
	y := r.Min.Y
	fw := a.body.W
	// fav
	if a.cfg.Favorites[row.K] {
		c.Text(x, y, a.body, "*", gen.Eva.Accent)
	}
	x += fw
	// title
	st := a.status(i)
	titleCol := gen.Eva.Fg
	if selected {
		titleCol = gen.Eva.Accent
	} else if st == data.StatusNotFound {
		titleCol = gen.Eva.Muted
	}
	a.paintTitle(c, x, y, l.TitleW, d.Title, row.Beta, titleCol)
	// the status glyph and the date use the narrow font at the titles'
	// height, whose baseline sits one pixel below the body font's
	rf := a.rowFont()
	x += l.TitleW + rf.W
	g, gc := statusGlyph(st)
	c.Text(x, y-1, rf, g, gc)
	// date, right-aligned in its column
	date := row.Updated
	if a.mode == data.SortDebut {
		date = row.Date
	} else if a.mode == data.SortRecents {
		date = a.launchedAt(row.K) // when it was launched last
	}
	// rows changed since the last look show their date in the accent
	dateCol := gen.Eva.Muted
	if a.seen != nil && a.seen.MarkerOn(a.mode) && a.seen.Unseen(row) {
		dateCol = gen.Eva.Accent
	}
	text := a.dateCol(date)
	if a.mode == data.SortYear || a.mode == data.SortMaker {
		// the original release year as the catalogue has it, "198?" included
		text = gfx.Fit(strings.TrimSpace(row.Year), a.dateCols())
		for len(text) < a.dateCols() {
			text = " " + text
		}
	}
	c.Text(r.Max.X-a.dateCols()*rf.W, y-1, rf, text, dateCol)
}

// paintTitle draws a list title into w pixels, in the narrow font when
// chosen, with a beta sign after a Patreon beta core's name.
func (a *App) paintTitle(c *gfx.Canvas, x, y, w int, title string, beta bool, col rgb) {
	if f := a.titleFont(); f != nil {
		y-- // scientifica's baseline is one pixel lower than the body font's
		if beta {
			w -= f.Advance(' ') + f.Advance(gfx.Beta[0])
		}
		tw := c.TextProp(x, y, f, gfx.FitProp(f, title, w), col)
		if beta {
			c.TextProp(x+tw+f.Advance(' '), y, f, gfx.Beta, gen.Eva.Warn)
		}
		return
	}
	cols := a.body.Cols(w)
	if beta {
		cols -= 2
	}
	s := gfx.Fit(title, cols)
	c.Text(x, y, a.body, s, col)
	if beta {
		c.Text(x+a.body.Width(s)+a.body.W, y, a.body, gfx.Beta, gen.Eva.Warn)
	}
}

// paintPane draws the highlighted row's thumbnail and specs.
func (a *App) paintPane(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Pane, gen.Eva.Bg)
	switch {
	case l.PaneTop:
		c.HLine(l.Pane.Min.X, l.Pane.Max.X-1, l.Pane.Max.Y, gen.Eva.Line)
	case l.Portrait:
		c.HLine(l.Pane.Min.X, l.Pane.Max.X-1, l.Pane.Min.Y-1, gen.Eva.Line)
	default:
		c.VLine(l.Pane.Min.X-1, l.Pane.Min.Y, l.Pane.Max.Y-1, gen.Eva.Line)
	}
	row, d, i := a.current()
	if row == nil {
		return
	}
	drawn := a.paintThumb(c, l.Thumb, row)
	text := l.PaneText
	if l.TextBeside {
		// beside the picture when at least 60 px remain there, else below
		if beside := l.Pane.Max.X - (drawn.Max.X + 4); beside >= 60 {
			text = image.Rect(drawn.Max.X+4, l.Pane.Min.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		} else {
			text = image.Rect(l.Thumb.Min.X, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		}
	}
	var lines []paneLine
	hue := typeHue(row.Base)
	cols := a.sm.Cols(text.Dx())
	for _, t := range gfx.Wrap(d.Title, cols, 2) {
		lines = append(lines, paneLine{t, gen.Eva.Fg})
	}
	// the card answer comes right after the title: it is the point of the app
	if a.cfg.Status != nil {
		st := a.status(i)
		_, sc := statusGlyph(st)
		lines = append(lines, paneLine{statusText(st, ""), sc})
	}
	kind := d.TypeLabel
	if row.IsArcade() {
		kind = "Arcade / " + d.SrcShort
	}
	lines = append(lines, paneLine{kind, hue})
	if row.Core != "" && d.CoreLabel != row.Title {
		lines = append(lines, paneLine{d.CoreLabel, gen.Eva.Muted})
	}
	if row.Year != "" || row.Manufacturer != "" {
		lines = append(lines, paneLine{strings.TrimSpace(row.Year + " " + data.ASCII(row.Manufacturer)), gen.Eva.Fg})
	}
	if row.IsArcade() {
		spec := strings.TrimSpace(strings.Join(nonEmpty(rotShort(row.Rot), plrShort(row.Plr)), " / "))
		if spec != "" {
			lines = append(lines, paneLine{spec, gen.Eva.Fg})
		}
		if d.Ctl != "" {
			lines = append(lines, paneLine{d.Ctl, gen.Eva.Fg})
		}
	}
	if ch := chips(row, d); len(ch) > 0 {
		col := gen.Eva.Muted
		for _, x := range ch {
			if x == "beta" || strings.HasPrefix(x, "boots") {
				col = gen.Eva.Warn
			}
		}
		lines = append(lines, paneLine{strings.Join(ch, ", "), col})
	}
	a.paintPaneText(c, text, lines)
}

type paneLine struct {
	text string
	col  rgb
}

func (a *App) paintPaneText(c *gfx.Canvas, r image.Rectangle, lines []paneLine) {
	lh := a.sm.H + 1
	cols := a.sm.Cols(r.Dx())
	y := r.Min.Y
	for _, ln := range lines {
		if y+a.sm.H > r.Max.Y {
			return
		}
		c.Text(r.Min.X, y, a.sm, gfx.Fit(ln.text, cols), ln.col)
		y += lh
	}
}

// paintThumb draws a row's list thumbnail or a placeholder into box and
// returns the rectangle it covered.
func (a *App) paintThumb(c *gfx.Canvas, box image.Rectangle, row *data.Row) image.Rectangle {
	key, slot := thumbSlot(row, a.ListShot())
	if key == "" {
		a.placeholder(c, box, "no shot")
		return box
	}
	// a horizontal game fills the 4:3 box like on the site; a vertical one
	// keeps its shape and sits on the left edge with the text
	req := ImageReq{Key: key, Slot: slot, W: box.Dx(), H: box.Dy(), Stretch: slot != "system" && row.ImgW > row.ImgH}
	img, st := a.cfg.Images.Get(req)
	if img == nil {
		a.want(req)
		switch st {
		case ImageLoading:
			a.placeholder(c, box, "loading")
		case ImageOffline:
			a.placeholder(c, box, "no connection")
		default:
			a.placeholder(c, box, "no shot")
		}
		return box
	}
	// horizontal: on the left edge, in line with the text below; the
	// classic tate pane centres it in the box beside the text; the wider
	// layouts keep it left so the text can sit beside it
	p := image.Pt(box.Min.X, box.Min.Y+(box.Dy()-img.Rect.Dy())/2)
	if a.lay.Portrait && !a.lay.TextBeside {
		p.X = box.Min.X + (box.Dx()-img.Rect.Dx())/2
	}
	if slot == "system" {
		c.BlitAlpha(p, img)
	} else {
		c.Blit(p, img)
	}
	return image.Rectangle{Min: p, Max: p.Add(img.Rect.Size())}
}

// placeholder stands in for a picture: a solid black shape with a word on
// it (pictures themselves draw bare, no frame).
func (a *App) placeholder(c *gfx.Canvas, box image.Rectangle, text string) {
	c.Fill(box, rgb{R: 0, G: 0, B: 0, A: 255})
	w := a.sm.Width(text)
	c.Text(box.Min.X+(box.Dx()-w)/2, box.Min.Y+(box.Dy()-a.sm.H)/2, a.sm, text, gen.Eva.Muted)
}

func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
