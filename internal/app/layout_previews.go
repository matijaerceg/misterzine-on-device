package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// paintLayoutPreviews renders the actual list three ways, then shrinks each
// complete logical screen. A copy keeps preview layout/scroll changes out of
// the live app; missing artwork requests still reach the image provider.
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
	box := image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+shotH+a.sm.H+10)
	c.Fill(box, gen.Eva.Surface)
	c.Box(box, gen.Eva.Line)
	frame := gfx.New(w, h)
	// Prefer the current game, then another visible game, then the catalogue.
	// Only the preview copy changes selection; filters and the live cursor stay put.
	sample := *a
	matches := func(index int) bool {
		r := &a.ds.Rows[index]
		return r.Img != "" && len(r.ImgSlots) > 0 && r.ImgW > 0 && r.ImgH > 0 && (r.ImgH > r.ImgW) == a.lay.Portrait
	}
	if len(sample.view) == 0 || !matches(sample.view[sample.cursor]) {
		found := false
		for pos, index := range sample.view {
			if matches(index) {
				sample.cursor, found = pos, true
				break
			}
		}
		if !found {
			for index := range a.ds.Rows {
				if matches(index) {
					sample.view = []int{index}
					sample.cursor, sample.top = 0, 0
					sample.marks = nil
					break
				}
			}
		}
	}
	for i, style := range listLayouts {
		preview := sample
		preview.screen = ScreenList
		preview.notice = ""
		preview.cfg.ListLayout = style
		preview.lay = NewLayout(w, h, a.cfg.SafeInsetX, a.cfg.SafeInsetY, a.body, a.rowFont().W, a.dateCols(), style)
		preview.wants = nil
		preview.ensureVisible()
		frame.Fill(frame.Rect, gen.Eva.Bg)
		preview.paintList(frame)
		for _, req := range preview.wants {
			a.want(req)
		}
		x := box.Min.X + 2 + i*(cellW+gap)
		r := image.Rect(x+(cellW-shotW)/2, box.Min.Y+4, x+(cellW-shotW)/2+shotW, box.Min.Y+4+shotH)
		for dy := 0; dy < shotH; dy++ {
			for dx := 0; dx < shotW; dx++ {
				c.SetRGBA(r.Min.X+dx, r.Min.Y+dy, frame.RGBAAt(dx*w/shotW, dy*h/shotH))
			}
		}
		col := gen.Eva.Fg
		if style == a.ListLayout() {
			col = gen.Eva.Accent
			c.Box(r.Inset(-2), col)
		}
		c.Text(x+(cellW-a.sm.Width(style))/2, r.Max.Y+3, a.sm, style, col)
	}
}
