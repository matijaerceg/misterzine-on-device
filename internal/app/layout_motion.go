package app

import (
	"image"
	"math"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

const layoutMotionDuration = 200 * time.Millisecond

// Only geometry moves. Artwork uses a frozen bitmap; text retains its pixel size.
// The real layout is always the destination, so input never waits for animation.
type layoutMotion struct {
	active                           bool
	dirty                            bool
	at                               time.Time
	progress                         float64
	from, current                    Layout
	fromY, currentY                  int
	fromThumb, toThumb, currentThumb image.Rectangle
	fromText, toText, currentText    image.Rectangle
	art, frame, caption, toCaption   *image.RGBA
	request                          ImageReq
	key, shot                        string
	view                             pageIdentity
	artX                             []int
	fromDivider, currentDivider      paneDivider
}

func (a *App) LayoutTransitionRunning() bool { return a.layoutMotion.active }

func copyLayoutPixels(src *image.RGBA, r image.Rectangle) *image.RGBA {
	r = r.Intersect(src.Rect)
	if r.Empty() {
		return nil
	}
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		i := src.PixOffset(r.Min.X, r.Min.Y+y)
		copy(out.Pix[y*out.Stride:y*out.Stride+r.Dx()*4], src.Pix[i:i+r.Dx()*4])
	}
	return out
}

func (a *App) captureLayoutMotion() layoutMotion {
	if !a.transition.enabled || !a.PageTransitions() || !a.transition.painted || a.screen != ScreenList || a.saver.active || a.PageTransitionRunning() {
		return layoutMotion{}
	}
	m := layoutMotion{active: true, from: a.lay, fromY: a.lay.List.Min.Y + (a.screenLine(a.cursor)-a.top)*a.lay.Line + a.listMotion.offset,
		fromDivider: dividerForLayout(a.lay), fromThumb: a.paintedThumb, fromText: a.paintedPaneText, key: a.CursorKey(), shot: a.ListShot(), view: a.pageIdentity()}
	if a.LayoutTransitionRunning() {
		old := &a.layoutMotion
		m.from, m.fromY, m.fromThumb, m.fromText = old.current, old.currentY, old.currentThumb, old.currentText
		m.art = old.art
		m.fromDivider = old.currentDivider
	} else if a.paintedThumbImage != nil {
		m.art = copyLayoutPixels(a.logical.RGBA, m.fromThumb)
	}
	m.caption = copyLayoutPixels(a.logical.RGBA, m.fromText)
	m.frame = copyLayoutPixels(a.logical.RGBA, a.logical.Rect)
	return m
}

func (a *App) startLayoutMotion(m layoutMotion) {
	if !m.active {
		return
	}
	// Resolve only the destination image size. Never send intermediate sizes to
	// the image worker/cache while the pane is travelling.
	c := gfx.New(a.lay.W, a.lay.H)
	c.Fill(c.Rect, gen.Eva.Bg)
	images := a.cfg.Images
	if m.art != nil {
		size := image.Point{}
		if row, _, _ := a.current(); row != nil {
			size = image.Pt(row.ImgW, row.ImgH)
		}
		a.cfg.Images = layoutFallbackImages{Images: images, art: m.art, size: size}
	}
	a.paintPane(c)
	a.cfg.Images = images
	m.toThumb, m.toText = a.paintedThumb, a.paintedPaneText
	m.toCaption = copyLayoutPixels(c.RGBA, m.toText)
	if row, _, _ := a.current(); row != nil && !a.lay.Thumb.Empty() {
		key, slot := thumbSlot(row, a.ListShot())
		m.request = ImageReq{Key: key, Slot: slot, W: a.lay.Thumb.Dx(), H: a.lay.Thumb.Dy(), Stretch: slot != "system" && row.ImgW > row.ImgH}
	}
	if m.art == nil && a.paintedThumbImage != nil {
		m.art = copyLayoutPixels(c.RGBA, m.toThumb)
	}
	// To or from the text layout the pane's contents keep their size and
	// slide off the body's far edge, the side that layout's empty pane marks.
	if m.from.Pane.Empty() && !a.lay.Pane.Empty() {
		off := offBody(m.from, a.lay.Body)
		m.fromThumb, m.fromText = m.toThumb.Add(off), m.toText.Add(off)
		m.caption = m.toCaption
	} else if !m.from.Pane.Empty() && a.lay.Pane.Empty() {
		off := offBody(a.lay, a.lay.Body)
		m.toThumb, m.toText = m.fromThumb.Add(off), m.fromText.Add(off)
		m.toCaption = m.caption
	}
	m.current, m.currentY, m.currentThumb, m.currentText = m.from, m.fromY, m.fromThumb, m.fromText
	m.currentDivider = m.fromDivider
	m.artX = make([]int, a.lay.W)
	m.at = a.cfg.TimerNow()
	a.layoutMotion = m
}

func (a *App) validateLayoutMotion() {
	m := &a.layoutMotion
	if m.active && (!a.PageTransitions() || a.screen != ScreenList || a.saver.active || a.pageIdentity() != m.view || a.CursorKey() != m.key || a.ListShot() != m.shot) {
		a.layoutMotion = layoutMotion{}
		a.all = true
	}
}

func (a *App) tickLayoutMotion(now time.Time) bool {
	a.validateLayoutMotion()
	m := &a.layoutMotion
	if !m.active {
		return false
	}
	m.progress = min(1, max(0, float64(now.Sub(m.at))/float64(layoutMotionDuration)))
	m.dirty = true
	if m.progress >= 1 {
		a.layoutMotion = layoutMotion{}
		a.all = true
	}
	return true
}

func layoutMix(a, b int, p float64) int { return a + int(math.Round(float64(b-a)*p)) }
func layoutRect(a, b image.Rectangle, p float64) image.Rectangle {
	return image.Rect(layoutMix(a.Min.X, b.Min.X, p), layoutMix(a.Min.Y, b.Min.Y, p), layoutMix(a.Max.X, b.Max.X, p), layoutMix(a.Max.Y, b.Max.Y, p))
}

func (a *App) paintLayoutMotion(c *gfx.Canvas) {
	m := &a.layoutMotion
	m.dirty = false
	if m.request.Key != "" {
		a.want(m.request)
	}
	if m.progress == 0 {
		body := &gfx.Canvas{RGBA: c.Sub(a.lay.Body)}
		body.Blit(image.Point{}, m.frame)
		c.Dirty(a.lay.Body)
		return
	}
	// Cubic ease-out starts promptly and settles without overshoot.
	p := 1 - math.Pow(1-m.progress, 3)
	final := a.lay
	l := final
	l.List = layoutRect(m.from.List, final.List, p)
	l.Scroll = layoutRect(m.from.Scroll, final.Scroll, p)
	l.Pane = layoutRect(m.from.Pane, final.Pane, p)
	l.Thumb = layoutRect(m.fromThumb, m.toThumb, p)
	l.PaneText = layoutRect(m.fromText, m.toText, p)
	l.TitleW = l.List.Dx() - a.body.W - a.rowFont().W*(3+a.dateCols())
	if l.PictureColumns {
		l.TitleW = l.List.Dx() - a.body.W
	} else {
		l.TitleW = max(8*a.body.W, l.TitleW)
	}
	l.Cols = max(1, l.List.Dx()/a.body.W)
	l.Lines = max(1, l.List.Dy()/l.Line)
	y := layoutMix(m.fromY, final.List.Min.Y+(a.screenLine(a.cursor)-a.top)*final.Line, p)
	y = max(l.List.Min.Y, min(y, l.List.Max.Y-l.Line))
	m.current, m.currentY, m.currentThumb, m.currentText = l, y, l.Thumb, l.PaneText
	a.lay = l
	defer func() { a.lay = final }()
	// Draw the same selected row at its interpolated position, revealing rows at
	// the edges. Clipping prevents partial rows painting over the fixed bars.
	rows := &gfx.Canvas{RGBA: c.Sub(l.List)}
	offset := y - (l.List.Min.Y + (a.screenLine(a.cursor)-a.top)*l.Line)
	first := max(0, a.top-offset/l.Line-1)
	pos, mk := 0, 0
	for pos < len(a.view) && a.screenLine(pos) < first {
		pos++
	}
	for mk < len(a.marks) && (a.marks[mk] < pos || a.markLine(mk) < first) {
		mk++
	}
	for line := first; ; line++ {
		r := l.lineRect(line - a.top).Add(image.Pt(0, offset))
		if r.Min.Y >= l.List.Max.Y {
			break
		}
		if mk < len(a.marks) && a.marks[mk] == pos {
			a.paintMarker(rows, r, a.markText(mk))
			mk++
			continue
		}
		if pos >= len(a.view) {
			break
		}
		a.paintRow(rows, r, pos)
		pos++
	}
	a.paintScrollbar(c)
	// Keep the pane opaque while its contents move with it; nothing of it
	// spills past the body when it slides out for the text layout.
	body := &gfx.Canvas{RGBA: c.Sub(final.Body)}
	body.Fill(l.Pane, gen.Eva.Bg)
	if m.art != nil {
		scaleLayoutArt(body, m.art, l.Thumb, m.artX)
	}
	// Dissolve between the two fixed caption rasters at their moving origin.
	// Neither lettering nor line spacing is scaled or rewrapped per frame.
	clip := l.PaneText.Intersect(l.Pane).Intersect(body.Rect)
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			src := m.caption
			if int(saverBayer[y&7][x&7]) < int(m.progress*256) {
				src = m.toCaption
			}
			if src == nil {
				continue
			}
			sx, sy := x-l.PaneText.Min.X, y-l.PaneText.Min.Y
			if !image.Pt(sx, sy).In(src.Rect) {
				continue
			}
			si, di := src.PixOffset(sx, sy), c.PixOffset(x, y)
			copy(c.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	m.currentDivider = m.fromDivider.interpolate(dividerForLayout(final), p)
	m.currentDivider.paint(c, final.Body)
	c.Dirty(final.Body)
}

// offBody is the translation that moves a rectangle in the body just past
// the edge where l's empty pane sits (text layout): right, or down in tate.
func offBody(l Layout, body image.Rectangle) image.Point {
	if l.Portrait {
		return image.Pt(0, body.Dy())
	}
	return image.Pt(body.Dx(), 0)
}

// Nearest-neighbour is deliberately inexpensive and preserves screenshot pixels.
func scaleLayoutArt(c *gfx.Canvas, src *image.RGBA, box image.Rectangle, xOffsets []int) {
	if box.Empty() {
		return
	}
	r := box.Intersect(c.Rect)
	for x := r.Min.X; x < r.Max.X; x++ {
		xOffsets[x] = (x - box.Min.X) * src.Rect.Dx() / box.Dx() * 4
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		sy := src.Rect.Min.Y + (y-box.Min.Y)*src.Rect.Dy()/box.Dy()
		row := src.PixOffset(src.Rect.Min.X, sy)
		di := c.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			si := row + xOffsets[x]
			copy(c.Pix[di:di+4], src.Pix[si:si+4])
			di += 4
		}
	}
}

// Resolve the destination's geometry even when its full-quality bitmap is not
// ready yet. Captions must travel to the actual fitted image, not a placeholder.
type layoutFallbackImages struct {
	Images
	art  *image.RGBA
	size image.Point
}

func (f layoutFallbackImages) Get(req ImageReq) (*image.RGBA, ImageState) {
	if img, state := f.Images.Get(req); img != nil {
		return img, state
	}
	w, h := req.W, req.H
	if !req.Stretch {
		size := f.size
		if size.X <= 0 || size.Y <= 0 {
			size = f.art.Rect.Size()
		}
		w, h = req.W, size.Y*req.W/size.X
		if h > req.H {
			h = req.H
			w = size.X * h / size.Y
		}
		if f.size.X > 0 && f.size.Y > 0 && (w > 2*size.X || h > 2*size.Y) {
			w, h = 2*size.X, 2*size.Y
		}
	}
	c := gfx.New(max(1, w), max(1, h))
	scaleLayoutArt(c, f.art, c.Rect, make([]int, c.W()))
	return c.RGBA, ImageReady
}

// The divider is a connected vertical/horizontal path. When a side pane
// becomes a top pane, the vertical leg shortens as the horizontal leg grows
// around its corner; it never disappears or cuts diagonally through the art.
type paneDivider [3]image.Point

func dividerForLayout(l Layout) paneDivider {
	if l.Pane.Empty() {
		// the text layout: a divider on the body's far edge, which paint
		// clips away; a transition slides the real one out to it
		if l.Portrait {
			left, right := image.Pt(l.Pane.Min.X, l.Body.Max.Y), image.Pt(l.Pane.Max.X-1, l.Body.Max.Y)
			return paneDivider{left, left, right}
		}
		top, bottom := image.Pt(l.Body.Max.X, l.Pane.Min.Y), image.Pt(l.Body.Max.X, l.Pane.Max.Y-1)
		return paneDivider{top, bottom, bottom}
	}
	if l.PaneTop || l.Portrait {
		y := l.Pane.Min.Y - 1
		if l.PaneTop {
			y = l.Pane.Max.Y
		}
		left, right := image.Pt(l.Pane.Min.X, y), image.Pt(l.Pane.Max.X-1, y)
		return paneDivider{left, left, right}
	}
	top, bottom := image.Pt(l.Pane.Min.X-1, l.Pane.Min.Y), image.Pt(l.Pane.Min.X-1, l.Pane.Max.Y-1)
	return paneDivider{top, bottom, bottom}
}
func (d paneDivider) interpolate(to paneDivider, p float64) paneDivider {
	var out paneDivider
	for i := range out {
		out[i] = image.Pt(layoutMix(d[i].X, to[i].X, p), layoutMix(d[i].Y, to[i].Y, p))
	}
	return out
}
func (d paneDivider) paint(c *gfx.Canvas, body image.Rectangle) {
	b := &gfx.Canvas{RGBA: c.Sub(body)}
	b.VLine(d[0].X, d[0].Y, d[1].Y, gen.Eva.Line)
	b.HLine(d[1].X, d[2].X, d[2].Y, gen.Eva.Line)
}
