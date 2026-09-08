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
	a.paintPane(c)
	if a.lay.Portrait {
		a.paintHint(c, "A info  > shots  Y sort  X filters+settings")
	} else {
		a.paintHint(c, "A details  > shots  Y sort  X filters and settings")
	}
}

func (a *App) paintStatus(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Status, gen.Eva.Surface)
	y := l.Status.Min.Y + 2
	left := a.mode.String()
	if a.filters.Active() {
		left += "  " + itoa(len(a.view)) + " of " + itoa(len(a.ds.Rows)) + " releases"
	} else {
		left += "  " + itoa(len(a.ds.Rows)) + " releases"
	}
	c.Text(l.Status.Min.X+2, y, a.sm, left, gen.Eva.Accent)
	right := a.net
	if a.notice != "" {
		right = a.notice
	}
	if right != "" {
		c.TextRight(l.Status.Max.X-2, y, a.sm, gfx.Fit(right, a.sm.Cols(l.Status.Dx()-a.sm.Width(left)-8)), gen.Eva.Fg)
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
		if x+a.sm.Width(btn+" "+rest) > maxX {
			break
		}
		x += c.Text(x, y, a.sm, btn, gen.Eva.Accent)
		x += c.Text(x, y, a.sm, " "+rest+"  ", gen.Eva.Muted)
	}
}

// paintRows draws the visible list lines: rows, and the last-look marker.
func (a *App) paintRows(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.List, gen.Eva.Bg)
	if len(a.view) == 0 {
		c.Text(l.List.Min.X+a.body.W, l.List.Min.Y+l.Line, a.body, gfx.Fit("no rows match, X: filters", l.Cols-1), gen.Eva.Muted)
		return
	}
	// which view position sits on each visible line
	pos := 0
	// advance pos to the first visible line
	for pos < len(a.view) && a.screenLine(pos) < a.top {
		pos++
	}
	for n := 0; n < l.Lines; n++ {
		line := a.top + n
		r := l.lineRect(n)
		switch {
		case a.topMark && line == 0:
			a.paintMarker(c, r, "nothing new since "+a.seen.Label(a.cfg.Now(), a.cfg.ClockTrusted))
		case a.split >= 0 && line == a.screenLine(a.split)+1:
			a.paintMarker(c, r, a.seen.Label(a.cfg.Now(), a.cfg.ClockTrusted))
		default:
			if pos >= len(a.view) {
				return
			}
			a.paintRow(c, r, pos)
			pos++
		}
	}
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
	// a thin rule where the sort date changes, inside the row
	if pos > 0 {
		prev := &a.ds.Rows[a.view[pos-1]]
		cur, was := row.Updated, prev.Updated
		if a.mode == data.SortDebut {
			cur, was = row.Date, prev.Date
		}
		if cur != was {
			c.HLine(r.Min.X, r.Max.X-1, r.Min.Y, gen.Eva.Line)
		}
	}
	x := r.Min.X
	y := r.Min.Y
	fw := a.body.W
	// fav
	if a.cfg.Favorites[row.K] {
		c.Text(x, y, a.body, "*", gen.Eva.Accent)
	}
	x += fw * 2
	// title
	st := a.status(i)
	titleCol := gen.Eva.Fg
	if selected {
		titleCol = gen.Eva.Accent
	} else if st == data.StatusNotFound {
		titleCol = gen.Eva.Muted
	}
	c.Text(x, y, a.body, gfx.Fit(d.Title, l.TitleCol), titleCol)
	x += fw * (l.TitleCol + 1)
	// status glyph
	g, gc := statusGlyph(st)
	c.Text(x, y, a.body, g, gc)
	x += fw * 2
	// date
	date := row.Updated
	if a.mode == data.SortDebut {
		date = row.Date
	}
	// rows changed since the last look show their date in the accent
	dateCol := gen.Eva.Muted
	if a.seen != nil && a.seen.MarkerOn(a.mode) && a.seen.Unseen(row) {
		dateCol = gen.Eva.Accent
	}
	c.Text(x, y, a.body, a.dateCol(date), dateCol)
}

// paintPane draws the highlighted row's thumbnail and specs.
func (a *App) paintPane(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Pane, gen.Eva.Bg)
	if l.Portrait {
		c.HLine(l.Pane.Min.X, l.Pane.Max.X-1, l.Pane.Min.Y-1, gen.Eva.Line)
	} else {
		c.VLine(l.Pane.Min.X-1, l.Pane.Min.Y, l.Pane.Max.Y-1, gen.Eva.Line)
	}
	row, d, i := a.current()
	if row == nil {
		return
	}
	a.paintThumb(c, l.Thumb, row)
	var lines []paneLine
	hue := typeHue(row.Base)
	cols := a.sm.Cols(l.PaneText.Dx())
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
			if strings.HasPrefix(x, "boots") {
				col = gen.Eva.Warn
			}
		}
		lines = append(lines, paneLine{strings.Join(ch, ", "), col})
	}
	a.paintPaneText(c, l.PaneText, lines)
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

// paintThumb draws a row's list thumbnail or a placeholder into box.
func (a *App) paintThumb(c *gfx.Canvas, box image.Rectangle, row *data.Row) {
	c.Fill(box, gen.Eva.Surface)
	c.Box(box, gen.Eva.Line)
	key, slot := thumbSlot(row)
	if key == "" {
		a.placeholder(c, box, "no shot")
		return
	}
	req := ImageReq{Key: key, Slot: slot, W: box.Dx() - 2, H: box.Dy() - 2}
	img, st := a.cfg.Images.Get(req)
	if img == nil {
		a.want(req)
		switch st {
		case ImageLoading:
			a.placeholder(c, box, "loading")
		case ImageOffline:
			a.placeholder(c, box, "offline")
		default:
			a.placeholder(c, box, "no shot")
		}
		return
	}
	p := image.Pt(box.Min.X+(box.Dx()-img.Rect.Dx())/2, box.Min.Y+(box.Dy()-img.Rect.Dy())/2)
	if slot == "system" {
		c.BlitAlpha(p, img)
	} else {
		c.Blit(p, img)
	}
}

func (a *App) placeholder(c *gfx.Canvas, box image.Rectangle, text string) {
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
