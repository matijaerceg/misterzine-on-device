package images

import (
	"image"
	"image/color"
	"testing"
)

func flat(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestResampleFlatIsExact(t *testing.T) {
	src := flat(384, 224, color.RGBA{200, 100, 50, 255})
	out := Resample(src, 96, 56)
	if out.Rect.Dx() != 96 || out.Rect.Dy() != 56 {
		t.Fatalf("size %v", out.Rect)
	}
	for y := 0; y < 56; y += 7 {
		for x := 0; x < 96; x += 5 {
			if got := out.RGBAAt(x, y); got != (color.RGBA{200, 100, 50, 255}) {
				t.Fatalf("pixel %d,%d = %v", x, y, got)
			}
		}
	}
}

func TestResampleCheckerAverages(t *testing.T) {
	// 2x2 checker of black/white halves to mid grey when halved
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := uint8(0)
			if (x+y)%2 == 0 {
				v = 255
			}
			src.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	out := Resample(src, 4, 4)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			p := out.RGBAAt(x, y)
			if p.R < 126 || p.R > 129 {
				t.Fatalf("pixel %d,%d = %v, want ~128", x, y, p)
			}
		}
	}
}

func TestResampleFractional(t *testing.T) {
	// 240x292 vertical shot into a 96x72 box keeps aspect: 59x72
	fw, fh := FitSize(240, 292, 96, 72)
	if fw != 59 || fh != 72 {
		t.Fatalf("fit = %dx%d", fw, fh)
	}
	src := flat(240, 292, color.RGBA{10, 20, 30, 255})
	out := Resample(src, fw, fh)
	if out.Rect.Dx() != 59 || out.Rect.Dy() != 72 || out.RGBAAt(58, 71) != (color.RGBA{10, 20, 30, 255}) {
		t.Fatalf("fractional resample wrong: %v %v", out.Rect, out.RGBAAt(58, 71))
	}
	// never more than 2x up
	if w, h := FitSize(40, 30, 400, 300); w != 80 || h != 60 {
		t.Fatalf("upscale cap = %dx%d", w, h)
	}
}

func BenchmarkResample384to96(b *testing.B) {
	src := flat(384, 224, color.RGBA{1, 2, 3, 255})
	for i := 0; i < b.N; i++ {
		Resample(src, 96, 56)
	}
}

func BenchmarkResample320toFull(b *testing.B) {
	src := flat(384, 224, color.RGBA{1, 2, 3, 255})
	for i := 0; i < b.N; i++ {
		Resample(src, 304, 177)
	}
}
