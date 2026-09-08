package gfx

import "image"

// Rotation of the logical canvas into the physical frame, named by how the
// MONITOR is turned. A monitor turned left (counter-clockwise, MiSTer
// osd_rotate=2) has its physical right edge at the top, so the logical top
// row must land on the physical right column; a monitor turned right
// (clockwise, osd_rotate=1) has its physical left edge at the top.
type Rotation int

const (
	RotNone  Rotation = iota // landscape: logical == physical
	RotRight                 // monitor turned right (osd_rotate=1): logical top -> physical left
	RotLeft                  // monitor turned left (osd_rotate=2): logical top -> physical right
)

func (r Rotation) String() string {
	switch r {
	case RotRight:
		return "right"
	case RotLeft:
		return "left"
	}
	return "off"
}

// Rotated reports whether the logical canvas is portrait.
func (r Rotation) Rotated() bool { return r != RotNone }

// RotateRect copies rectangle r of the logical canvas src into the physical
// dst, rotated, and returns the physical rectangle it touched. With a
// physical W x H frame and logical point (lx, ly):
//
//	RotLeft:  px = W-1-ly, py = lx      (logical top -> physical right)
//	RotRight: px = ly,     py = H-1-lx  (logical top -> physical left)
//
// Only the pixels of r are visited, so a small dirty rect stays cheap.
func RotateRect(dst, src *image.RGBA, r image.Rectangle, rot Rotation) image.Rectangle {
	r = r.Intersect(src.Rect)
	if r.Empty() {
		return image.Rectangle{}
	}
	W, H := dst.Rect.Dx(), dst.Rect.Dy()
	var out image.Rectangle
	switch rot {
	case RotLeft:
		out = image.Rect(W-r.Max.Y, r.Min.X, W-r.Min.Y, r.Max.X)
		for ly := r.Min.Y; ly < r.Max.Y; ly++ {
			px := W - 1 - ly
			so := src.PixOffset(r.Min.X, ly)
			for lx := r.Min.X; lx < r.Max.X; lx++ {
				do := dst.PixOffset(px, lx)
				copy(dst.Pix[do:do+4], src.Pix[so:so+4])
				so += 4
			}
		}
	case RotRight:
		out = image.Rect(r.Min.Y, H-r.Max.X, r.Max.Y, H-r.Min.X)
		for ly := r.Min.Y; ly < r.Max.Y; ly++ {
			px := ly
			so := src.PixOffset(r.Min.X, ly)
			for lx := r.Min.X; lx < r.Max.X; lx++ {
				do := dst.PixOffset(px, H-1-lx)
				copy(dst.Pix[do:do+4], src.Pix[so:so+4])
				so += 4
			}
		}
	default:
		out = r
		w := r.Dx() * 4
		for y := r.Min.Y; y < r.Max.Y; y++ {
			so := src.PixOffset(r.Min.X, y)
			do := dst.PixOffset(r.Min.X, y)
			copy(dst.Pix[do:do+w], src.Pix[so:so+w])
		}
	}
	return out.Intersect(dst.Rect)
}

// RotateAll copies the whole logical canvas into dst.
func RotateAll(dst, src *image.RGBA, rot Rotation) image.Rectangle {
	return RotateRect(dst, src, src.Rect, rot)
}
