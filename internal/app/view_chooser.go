package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func (a *App) openViewChooser() {
	p := panelState{}
	genres := false
	for _, m := range a.enabledViews() {
		if m.Genre() != "" && !genres {
			p.entries = append(p.entries, panelEntry{text: "Genres", info: true})
			genres = true
		}
		if m == a.mode {
			p.cursor = len(p.entries)
		}
		p.entries = append(p.entries, panelEntry{text: viewLabels[m], value: m.Name(), checked: m == a.mode})
	}
	a.viewChooser = p
	a.screen = ScreenViewChooser
	a.all = true
}

func (a *App) actViewChooser(k platform.Key) bool {
	p := &a.viewChooser
	n := len(p.entries) // the fixed Edit row occupies index n
	switch k {
	case platform.KeyBack:
		a.screen = ScreenList
	case platform.KeyEnter:
		if p.cursor == n {
			a.openViews("chooser")
		} else {
			m, ok := data.ParseSort(p.entries[p.cursor].value)
			if !ok {
				return false
			}
			a.screen = ScreenList
			if m != a.mode {
				a.SetSort(m)
			}
		}
	case platform.KeyUp, platform.KeyDown:
		d := 1
		if k == platform.KeyUp {
			d = -1
		}
		for i := p.cursor + d; ; i += d {
			if i < 0 || i > n {
				if a.rep.count > 0 {
					break
				}
				i = (i + n + 1) % (n + 1)
			}
			if i == n || !p.entries[i].info {
				p.cursor = i
				break
			}
		}
	case platform.KeyHome, platform.KeyPageUp:
		p.cursor = 0
		for p.cursor < n && p.entries[p.cursor].info {
			p.cursor++
		}
	case platform.KeyEnd, platform.KeyPageDown:
		p.cursor = n
	default:
		return false
	}
	a.all = true
	return true
}

func (a *App) paintViewChooser(c *gfx.Canvas) {
	p, f := &a.viewChooser, a.sm
	body := a.lay.Body
	// Keep a visible margin around the popup, even inside a narrow safe zone.
	w := min(body.Dx()-8, 38*f.W+12)
	lines := max(1, min(12, min(len(p.entries), (body.Dy()-16)/f.H-3)))
	h := (lines+3)*f.H + 12
	x, y := body.Min.X+(body.Dx()-w)/2, body.Min.Y+(body.Dy()-h)/2
	box := image.Rect(x, y, x+w, y+h)
	c.Fill(box, pal.Bg)
	c.Box(box, pal.Line)
	c.Text(x+6, y+4, f, "Choose view", pal.Accent)
	topY := y + f.H + 7
	editY := topY + lines*f.H + 3
	c.HLine(x+4, x+w-5, editY-2, pal.Line)
	cols := f.Cols(w-12) - 3
	if p.cursor < len(p.entries) {
		p.top = centeredTop(p.cursor, len(p.entries), lines)
	}
	p.top = max(0, min(p.top, max(0, len(p.entries)-lines)))
	p.lines = lines
	for i := p.top; i < len(p.entries) && i < p.top+lines; i++ {
		e := p.entries[i]
		ry := topY + (i-p.top)*f.H
		col := pal.Fg
		if i == p.cursor {
			c.Fill(image.Rect(x+3, ry, x+w-3, ry+f.H), pal.Surface)
			col = pal.Accent
		}
		mark := "  "
		if e.checked {
			mark = gfx.ArrowRight + " "
		}
		if e.info {
			col = pal.Muted
			mark = ""
		}
		c.Text(x+6, ry, f, mark+gfx.Fit(e.text, cols), col)
	}
	if p.top > 0 {
		c.Text(x+w-6-f.W, topY, f, gfx.ArrowUp, pal.Muted)
	}
	if p.top+lines < len(p.entries) {
		c.Text(x+w-6-f.W, topY+(lines-1)*f.H, f, gfx.ArrowDown, pal.Muted)
	}
	col := pal.Fg
	if p.cursor == len(p.entries) {
		c.Fill(image.Rect(x+3, editY, x+w-3, editY+f.H), pal.Surface)
		col = pal.Accent
	}
	c.Text(x+6, editY, f, gfx.Fit("Edit available views...", f.Cols(w-12)), col)
	hint := "A Select  B Cancel"
	if p.cursor == len(p.entries) {
		hint = "A Edit  B Cancel"
	}
	a.paintHint(c, hint)
}
