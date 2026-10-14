package images

import "image"

// pixelBlocks is how many blocks span the shorter side of a pixelated
// picture: coarse enough that nothing in a mature screenshot reads, at a
// thumbnail and full screen alike, while its colours still hint at the game.
const pixelBlocks = 10

// Pixelate replaces img, in place, with flat blocks of its average colour.
func Pixelate(img *image.RGBA) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if w == 0 || h == 0 {
		return
	}
	b := max(2, min(w, h)/pixelBlocks)
	for by := 0; by < h; by += b {
		for bx := 0; bx < w; bx += b {
			ex, ey := min(bx+b, w), min(by+b, h)
			var r, g, bl, a, n int
			for y := by; y < ey; y++ {
				o := img.PixOffset(img.Rect.Min.X+bx, img.Rect.Min.Y+y)
				for x := bx; x < ex; x++ {
					r += int(img.Pix[o])
					g += int(img.Pix[o+1])
					bl += int(img.Pix[o+2])
					a += int(img.Pix[o+3])
					o += 4
					n++
				}
			}
			px := [4]uint8{uint8(r / n), uint8(g / n), uint8(bl / n), uint8(a / n)}
			for y := by; y < ey; y++ {
				o := img.PixOffset(img.Rect.Min.X+bx, img.Rect.Min.Y+y)
				for x := bx; x < ex; x++ {
					copy(img.Pix[o:o+4], px[:])
					o += 4
				}
			}
		}
	}
}

// scaleFor is Scale plus the request's pixelation.
func scaleFor(src *image.RGBA, k scaledKey) *image.RGBA {
	out := Scale(src, k.W, k.H, k.Native, k.Stretch)
	if k.Pixelate {
		Pixelate(out)
	}
	return out
}
