// Package gfx is the software renderer: an RGBA canvas with dirty-rect
// tracking, fills, blits, bitmap-font text and the 90-degree rotation that
// turns the tate canvas into the physical landscape frame. Everything writes
// straight into pixel memory; nothing here allocates per frame.
package gfx

import (
	"image"
	"image/color"
)

// MaxDirty caps the dirty list; beyond it the list collapses to one box.
const MaxDirty = 8

// Canvas is an RGBA frame plus the rectangles touched since the last take.
type Canvas struct {
	*image.RGBA
	dirty []image.Rectangle
}

// New allocates a canvas.
func New(w, h int) *Canvas {
	return &Canvas{RGBA: image.NewRGBA(image.Rect(0, 0, w, h))}
}

// W and H are the canvas size.
func (c *Canvas) W() int { return c.Rect.Dx() }
func (c *Canvas) H() int { return c.Rect.Dy() }

// Dirty records a touched rectangle.
func (c *Canvas) Dirty(r image.Rectangle) {
	r = r.Intersect(c.Rect)
	if r.Empty() {
		return
	}
	for i, d := range c.dirty {
		if d.Overlaps(r) || d.Eq(r) {
			c.dirty[i] = d.Union(r)
			return
		}
	}
	c.dirty = append(c.dirty, r)
	if len(c.dirty) > MaxDirty {
		u := c.dirty[0]
		for _, d := range c.dirty[1:] {
			u = u.Union(d)
		}
		c.dirty = c.dirty[:1]
		c.dirty[0] = u
	}
}

// DirtyAll marks the whole canvas.
func (c *Canvas) DirtyAll() {
	c.dirty = c.dirty[:0]
	c.dirty = append(c.dirty, c.Rect)
}

// TakeDirty returns the touched rectangles and clears the list.
func (c *Canvas) TakeDirty() []image.Rectangle {
	out := make([]image.Rectangle, len(c.dirty))
	copy(out, c.dirty)
	c.dirty = c.dirty[:0]
	return out
}

// Fill paints a rectangle.
func (c *Canvas) Fill(r image.Rectangle, col color.RGBA) {
	r = r.Intersect(c.Rect)
	if r.Empty() {
		return
	}
	w := r.Dx()
	first := c.PixOffset(r.Min.X, r.Min.Y)
	row := c.Pix[first : first+w*4]
	for x := 0; x < w*4; x += 4 {
		row[x], row[x+1], row[x+2], row[x+3] = col.R, col.G, col.B, 255
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		off := c.PixOffset(r.Min.X, y)
		copy(c.Pix[off:off+w*4], row)
	}
	c.Dirty(r)
}

// HLine paints a horizontal 1 px line from x0 to x1 inclusive.
func (c *Canvas) HLine(x0, x1, y int, col color.RGBA) {
	c.Fill(image.Rect(x0, y, x1+1, y+1), col)
}

// VLine paints a vertical 1 px line from y0 to y1 inclusive.
func (c *Canvas) VLine(x, y0, y1 int, col color.RGBA) {
	c.Fill(image.Rect(x, y0, x+1, y1+1), col)
}

// Box paints a 1 px border just inside r.
func (c *Canvas) Box(r image.Rectangle, col color.RGBA) {
	c.HLine(r.Min.X, r.Max.X-1, r.Min.Y, col)
	c.HLine(r.Min.X, r.Max.X-1, r.Max.Y-1, col)
	c.VLine(r.Min.X, r.Min.Y, r.Max.Y-1, col)
	c.VLine(r.Max.X-1, r.Min.Y, r.Max.Y-1, col)
}

// Blit copies src opaquely with its top-left at p, clipped.
func (c *Canvas) Blit(p image.Point, src *image.RGBA) {
	dst := src.Rect.Sub(src.Rect.Min).Add(p).Intersect(c.Rect)
	if dst.Empty() {
		return
	}
	w := dst.Dx() * 4
	for y := dst.Min.Y; y < dst.Max.Y; y++ {
		sy := y - p.Y + src.Rect.Min.Y
		sx := dst.Min.X - p.X + src.Rect.Min.X
		so := src.PixOffset(sx, sy)
		do := c.PixOffset(dst.Min.X, y)
		copy(c.Pix[do:do+w], src.Pix[so:so+w])
	}
	c.Dirty(dst)
}

// BlitAlpha composites src over the canvas (source-over, premultiplied
// alpha as image.RGBA stores it). Used for system photos only.
func (c *Canvas) BlitAlpha(p image.Point, src *image.RGBA) {
	dst := src.Rect.Sub(src.Rect.Min).Add(p).Intersect(c.Rect)
	if dst.Empty() {
		return
	}
	for y := dst.Min.Y; y < dst.Max.Y; y++ {
		sy := y - p.Y + src.Rect.Min.Y
		for x := dst.Min.X; x < dst.Max.X; x++ {
			sx := x - p.X + src.Rect.Min.X
			so := src.PixOffset(sx, sy)
			a := uint32(src.Pix[so+3])
			if a == 0 {
				continue
			}
			do := c.PixOffset(x, y)
			if a == 255 {
				copy(c.Pix[do:do+4], src.Pix[so:so+4])
				continue
			}
			inv := 255 - a
			for i := 0; i < 3; i++ {
				c.Pix[do+i] = uint8((uint32(src.Pix[so+i])*255 + uint32(c.Pix[do+i])*inv) / 255)
			}
			c.Pix[do+3] = 255
		}
	}
	c.Dirty(dst)
}

// Sub returns a view of a rectangle of the canvas sharing pixels.
func (c *Canvas) Sub(r image.Rectangle) *image.RGBA {
	return c.SubImage(r).(*image.RGBA)
}
