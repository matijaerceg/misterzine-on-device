package gfx

import (
	"image"
	"image/color"
	"testing"
)

// propFont is a 4x5 cell font with glyphs of different ink widths.
func propFont() *Font {
	f := &Font{W: 4, H: 5, descent: 1}
	f.SetGlyph('?', []byte{0xF0, 0x10, 0x20, 0x00, 0x20})
	f.SetGlyph('i', []byte{0x40, 0x00, 0x40, 0x40, 0x40}) // one column
	f.SetGlyph('m', []byte{0x00, 0xF0, 0xB0, 0xB0, 0xB0}) // four columns
	f.SetGlyph('j', []byte{0x00, 0x20, 0x00, 0x20, 0x20}) // ink starts in column 2
	f.SetGlyph(' ', make([]byte, 5))
	f.SetGlyph(Ellipsis[0], []byte{0, 0, 0, 0, 0xA0})
	return f
}

func TestProportionalMetrics(t *testing.T) {
	f := propFont()
	for ch, want := range map[byte]int{'i': 2, 'm': 5, 'j': 2, ' ': 2, '?': 5, 0xC3: 5} {
		if got := f.Advance(ch); got != want {
			t.Errorf("advance %q = %d, want %d", ch, got, want)
		}
	}
	if f.PropWidth("im j") != 2+5+2+2 {
		t.Fatal("PropWidth must sum the advances")
	}
	if got := FitProp(f, "im", 7); got != "im" {
		t.Fatalf("FitProp kept %q, want the whole string", got)
	}
	if got := FitProp(f, "iiiii", 7); got != "i"+Ellipsis {
		t.Fatalf("FitProp cut to %q, want an ellipsis after what fits", got)
	}
	if FitProp(f, "im", 0) != "" {
		t.Fatal("no room draws nothing")
	}
}

func TestTextPropShiftsInkToThePen(t *testing.T) {
	f := propFont()
	c := New(20, 5)
	white := color.RGBA{255, 255, 255, 255}
	if w := c.TextProp(2, 0, f, "ji", white); w != 4 {
		t.Fatalf("drawn width %d, want 4", w)
	}
	// j's ink column 2 lands on x=2, i's on x=4; nothing before the pen
	for x := 0; x < 8; x++ {
		lit := c.RGBA.RGBAAt(x, 1).R != 0
		if lit != (x == 2) {
			t.Fatalf("row 1 pixel %d lit=%v", x, lit)
		}
		lit = c.RGBA.RGBAAt(x, 2).R != 0
		if lit != (x == 4) {
			t.Fatalf("row 2 pixel %d lit=%v", x, lit)
		}
	}
}

func TestTextClipStaysInsideTheWindow(t *testing.T) {
	f := propFont()
	c := New(20, 5)
	white := color.RGBA{255, 255, 255, 255}
	clip := image.Rect(4, 0, 8, 5)
	c.TextClip(-3, 0, f, "mmmmm", white, clip)
	for x := 0; x < 20; x++ {
		lit := c.RGBA.RGBAAt(x, 1).R != 0
		if lit != (x >= 4 && x < 8) {
			t.Fatalf("pixel %d lit=%v outside the clip", x, lit)
		}
	}
	dirty := c.TakeDirty()
	for _, r := range dirty {
		if !r.In(clip) {
			t.Fatalf("dirty rect %v leaves the clip %v", r, clip)
		}
	}
}
