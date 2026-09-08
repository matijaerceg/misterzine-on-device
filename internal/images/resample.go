// Package images is the picture pipeline: screenshots and system photos are
// fetched into an on-card store, decoded and scaled on a worker, and served
// to the app from a bounded cache. Get never blocks; the app draws a
// placeholder and gets a wake-up when the picture lands.
package images

import (
	"image"
	"image/color"
	"image/draw"
)

// Resample scales src to exactly w x h. Downscaling averages every source
// pixel that falls inside each destination pixel (area average, exact for
// fractional ratios), which is what keeps 1-pixel scanline detail from
// dropping out on a CRT. Upscaling uses nearest neighbour so pixel art
// stays crisp; callers cap it at 2x.
func Resample(src *image.RGBA, w, h int) *image.RGBA {
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	if w <= 0 || h <= 0 || sw == 0 || sh == 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	if w == sw && h == sh {
		out := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(out, out.Rect, src, src.Rect.Min, draw.Src)
		return out
	}
	// horizontal pass into a w x sh intermediate, then vertical into w x h
	mid := axis(src, sw, sh, w, true)
	return axis(mid, w, sh, h, false)
}

// axis scales one axis of an RGBA image with area averaging (or nearest
// when growing). horizontal=true scales width sw->n keeping height sh;
// otherwise it scales height sh->n keeping width sw.
func axis(src *image.RGBA, sw, sh, n int, horizontal bool) *image.RGBA {
	var out *image.RGBA
	if horizontal {
		out = image.NewRGBA(image.Rect(0, 0, n, sh))
	} else {
		out = image.NewRGBA(image.Rect(0, 0, sw, n))
	}
	srcLen := sw
	if !horizontal {
		srcLen = sh
	}
	if n >= srcLen {
		// nearest
		for i := 0; i < n; i++ {
			s := i * srcLen / n
			copyLine(out, src, i, s, horizontal, sw, sh)
		}
		return out
	}
	// weights: destination cell i covers source [i*srcLen/n, (i+1)*srcLen/n)
	scale := float64(srcLen) / float64(n)
	lines := sh
	if !horizontal {
		lines = sw
	}
	acc := make([]float64, lines*4)
	for i := 0; i < n; i++ {
		start := float64(i) * scale
		end := start + scale
		for j := range acc {
			acc[j] = 0
		}
		s0 := int(start)
		s1 := int(end)
		if s1 >= srcLen {
			s1 = srcLen - 1
		}
		total := 0.0
		for s := s0; s <= s1; s++ {
			lo, hi := float64(s), float64(s+1)
			if lo < start {
				lo = start
			}
			if hi > end {
				hi = end
			}
			wgt := hi - lo
			if wgt <= 0 {
				continue
			}
			total += wgt
			accumulate(acc, src, s, horizontal, sw, sh, wgt)
		}
		if total == 0 {
			total = 1
		}
		writeLine(out, acc, i, horizontal, total)
	}
	return out
}

func copyLine(dst, src *image.RGBA, di, si int, horizontal bool, sw, sh int) {
	if horizontal {
		for y := 0; y < sh; y++ {
			so := src.PixOffset(src.Rect.Min.X+si, src.Rect.Min.Y+y)
			do := dst.PixOffset(di, y)
			copy(dst.Pix[do:do+4], src.Pix[so:so+4])
		}
		return
	}
	so := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y+si)
	do := dst.PixOffset(0, di)
	copy(dst.Pix[do:do+sw*4], src.Pix[so:so+sw*4])
}

func accumulate(acc []float64, src *image.RGBA, s int, horizontal bool, sw, sh int, wgt float64) {
	if horizontal {
		for y := 0; y < sh; y++ {
			so := src.PixOffset(src.Rect.Min.X+s, src.Rect.Min.Y+y)
			p := src.Pix[so : so+4]
			a := acc[y*4 : y*4+4]
			a[0] += float64(p[0]) * wgt
			a[1] += float64(p[1]) * wgt
			a[2] += float64(p[2]) * wgt
			a[3] += float64(p[3]) * wgt
		}
		return
	}
	so := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y+s)
	row := src.Pix[so : so+sw*4]
	for i := 0; i < sw*4; i++ {
		acc[i] += float64(row[i]) * wgt
	}
}

func writeLine(dst *image.RGBA, acc []float64, di int, horizontal bool, total float64) {
	inv := 1 / total
	if horizontal {
		h := dst.Rect.Dy()
		for y := 0; y < h; y++ {
			do := dst.PixOffset(di, y)
			a := acc[y*4 : y*4+4]
			dst.Pix[do] = clamp(a[0] * inv)
			dst.Pix[do+1] = clamp(a[1] * inv)
			dst.Pix[do+2] = clamp(a[2] * inv)
			dst.Pix[do+3] = clamp(a[3] * inv)
		}
		return
	}
	w := dst.Rect.Dx()
	do := dst.PixOffset(0, di)
	for i := 0; i < w*4; i++ {
		dst.Pix[do+i] = clamp(acc[i] * inv)
	}
}

func clamp(v float64) uint8 {
	v += 0.5
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// ToRGBA converts any decoded image to *image.RGBA (no-op when it already is).
func ToRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Rect, img, b.Min, draw.Src)
	return out
}

// FitSize is the size (fw, fh) that fits (w, h) inside (bw, bh) keeping the
// aspect ratio, never growing beyond 2x.
// Scale produces the bitmap for a box: fitted with the aspect kept; or,
// with stretch, filling the box exactly; or, when native is asked and the
// picture is no more than 15% larger than the box in either direction, the
// picture pixel for pixel cropped to the box (a 240p shot on a 240p screen).
func Scale(src *image.RGBA, bw, bh int, native, stretch bool) *image.RGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if stretch {
		return Resample(src, bw, bh)
	}
	if native && w <= bw*115/100 && h <= bh*115/100 {
		cw, ch := min(w, bw), min(h, bh)
		r := image.Rect((w-cw)/2, (h-ch)/2, (w-cw)/2+cw, (h-ch)/2+ch).Add(src.Rect.Min)
		out := image.NewRGBA(image.Rect(0, 0, cw, ch))
		for y := 0; y < ch; y++ {
			copy(out.Pix[y*out.Stride:y*out.Stride+cw*4], src.Pix[src.PixOffset(r.Min.X, r.Min.Y+y):src.PixOffset(r.Max.X, r.Min.Y+y)])
		}
		return out
	}
	fw, fh := FitSize(w, h, bw, bh)
	return Resample(src, fw, fh)
}

func FitSize(w, h, bw, bh int) (int, int) {
	if w <= 0 || h <= 0 || bw <= 0 || bh <= 0 {
		return 0, 0
	}
	fw, fh := bw, h*bw/w
	if fh > bh {
		fh = bh
		fw = w * bh / h
	}
	if fw > 2*w || fh > 2*h {
		fw, fh = 2*w, 2*h
	}
	if fw < 1 {
		fw = 1
	}
	if fh < 1 {
		fh = 1
	}
	return fw, fh
}

var _ = color.RGBA{}
