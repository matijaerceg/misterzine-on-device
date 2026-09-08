package gfx

import "image"

// Rotation of the logical canvas into the physical frame.
type Rotation int

const (
	RotNone Rotation = iota // landscape: logical == physical
	RotCW                   // MiSTer osd_rotate=1, monitor turned right (+90)
	RotCCW                  // MiSTer osd_rotate=2, monitor turned left (-90)
)

// Rotated reports whether the logical canvas is portrait.
func (r Rotation) Rotated() bool { return r != RotNone }

// RotateRect copies rectangle r of the logical canvas src into the physical
// dst, rotated, and returns the physical rectangle it touched. With a
// physical W x H frame and logical point (lx, ly):
//
//	CW:  px = W-1-ly, py = lx
//	CCW: px = ly,     py = H-1-lx
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
	case RotCW:
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
	case RotCCW:
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
