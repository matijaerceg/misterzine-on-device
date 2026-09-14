package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// paintLayoutPreviews draws diagrams of the three layouts at the current orientation.
func (a *App) paintLayoutPreviews(c *gfx.Canvas, area image.Rectangle) {
	const gap = 4
	cellW := (area.Dx() - 2*gap - 4) / 3
	w, h := c.W(), c.H()
	shotW := cellW - 4
	shotH := shotW * h / w
	maxH := area.Dy() - a.sm.H - 10
	if shotH > maxH {
		shotH = maxH
		shotW = shotH * w / h
	}
	if shotW < 1 || shotH < 1 {
		return
	}
	box := area
	c.Fill(box, gen.Eva.Surface)
	c.Box(box, gen.Eva.Line)
	for i, style := range listLayouts {
		x := box.Min.X + 2 + i*(cellW+gap)
		r := image.Rect(x+(cellW-shotW)/2, box.Min.Y+4, x+(cellW-shotW)/2+shotW, box.Min.Y+4+shotH)
		a.paintLayoutDiagram(c, r, style)
		col := gen.Eva.Fg
		if style == a.ListLayout() {
			col = gen.Eva.Accent
			c.Box(r.Inset(-2), col)
		}
		c.Text(x+(cellW-a.sm.Width(style))/2, r.Max.Y+3, a.sm, style, col)
	}
}

// Diagram geometry follows the real layout, with a representative 4:3 or
// 3:4 artwork rectangle. Lines are drawn at thumbnail resolution to stay clear.
func (a *App) paintLayoutDiagram(c *gfx.Canvas, r image.Rectangle, style string) {
	l := NewLayout(c.W(), c.H(), a.cfg.SafeInsetX, a.cfg.SafeInsetY, a.body, a.rowFont().W, a.dateCols(), style)
	art := l.Thumb
	if l.Portrait {
		art.Max.X = art.Min.X + art.Dy()*3/4
		if !l.TextBeside {
			art = art.Add(image.Pt((l.Thumb.Dx()-art.Dx())/2, 0))
		}
	}
	meta := l.PaneText
	if l.TextBeside {
		if l.Pane.Max.X-art.Max.X-4 >= 60 {
			meta = image.Rect(art.Max.X+4, l.Pane.Min.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		} else {
			meta = image.Rect(l.Thumb.Min.X, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		}
	}
	c.Fill(r, gen.Eva.Bg)
	target := r.Inset(2)
	scaled := func(b image.Rectangle) image.Rectangle {
		return image.Rect(target.Min.X+(b.Min.X-l.Body.Min.X)*target.Dx()/l.Body.Dx(), target.Min.Y+(b.Min.Y-l.Body.Min.Y)*target.Dy()/l.Body.Dy(), target.Min.X+(b.Max.X-l.Body.Min.X)*target.Dx()/l.Body.Dx(), target.Min.Y+(b.Max.Y-l.Body.Min.Y)*target.Dy()/l.Body.Dy()).Intersect(target)
	}
	list := scaled(l.List)
	picture := scaled(art)
	metadata := scaled(meta)
	// Keep a visible gutter even when the real layout's spacing rounds down
	// to less than a pixel in these small diagrams.
	const gutter = 3
	if l.Portrait {
		list.Max.Y = min(list.Max.Y, min(picture.Min.Y, metadata.Min.Y)-gutter)
	} else if l.PaneTop {
		list.Min.Y = max(list.Min.Y, picture.Max.Y+gutter)
		metadata.Max.Y = min(metadata.Max.Y, list.Min.Y-gutter)
	} else {
		list.Max.X = min(list.Max.X, min(picture.Min.X, metadata.Min.X)-gutter)
	}
	if !l.Portrait || style != "list" {
		if meta.Min.X >= art.Max.X {
			metadata.Min.X = max(metadata.Min.X, picture.Max.X+gutter)
		} else {
			metadata.Min.Y = max(metadata.Min.Y, picture.Max.Y+gutter)
		}
	}
	for y := list.Min.Y; y < list.Max.Y; y += 4 {
		c.HLine(list.Min.X, list.Max.X-1, y, gen.Eva.Fg)
	}
	if !picture.Empty() {
		c.Box(picture, gen.Eva.Fg)
	}
	if !l.Portrait && style == "split" && !list.Empty() {
		lastListY := list.Min.Y + (list.Dy()-1)/4*4
		metadata.Max.Y = min(metadata.Max.Y, lastListY+1)
	}
	for y, n := metadata.Min.Y, 0; y < metadata.Max.Y && n < 6; y, n = y+3, n+1 {
		width := metadata.Dx() * 3 / 4
		if n%2 == 1 {
			width = metadata.Dx() / 2
		}
		if width > 0 {
			c.HLine(metadata.Min.X, metadata.Min.X+width-1, y, gen.Eva.Warn)
		}
	}
}
