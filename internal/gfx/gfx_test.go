package gfx

import (
	"image"
	"image/color"
	"os"
	"testing"
)

func loadFont(t *testing.T, name string) *Font {
	t.Helper()
	b, err := os.ReadFile("../fonts/" + name)
	if err != nil {
		t.Fatal(err)
	}
	f, err := ParseBDF(b)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParseBDF(t *testing.T) {
	f := loadFont(t, "spleen-6x12.bdf")
	if f.W != 6 || f.H != 12 {
		t.Fatalf("6x12 cell = %dx%d", f.W, f.H)
	}
	s := loadFont(t, "spleen-5x8.bdf")
	if s.W != 5 || s.H != 8 {
		t.Fatalf("5x8 cell = %dx%d", s.W, s.H)
	}
	for _, ch := range []byte{'A', 'g', '0', '?', '.'} {
		set := 0
		for _, r := range f.Glyph(ch) {
			if r != 0 {
				set++
			}
		}
		if set == 0 {
			t.Errorf("glyph %q is blank", ch)
		}
	}
	for _, r := range f.Glyph(' ') {
		if r != 0 {
			t.Fatal("space glyph is not blank")
		}
	}
	// glyphs must stay inside the 6 px cell: no bits below the MSB 6
	for ch := 32; ch < 127; ch++ {
		for _, r := range f.Glyph(byte(ch)) {
			if r&0x03 != 0 {
				t.Fatalf("glyph %q uses pixels outside a 6-wide cell", rune(ch))
			}
		}
	}
	// descenders reach the bottom rows, capitals do not
	g := f.Glyph('g')
	if g[f.H-1] == 0 && g[f.H-2] == 0 {
		t.Error("'g' has no descender in the bottom two rows")
	}
	a := f.Glyph('A')
	if a[f.H-1] != 0 {
		t.Error("'A' reaches the bottom row, baseline placement looks wrong")
	}
}

func TestTextAndDirty(t *testing.T) {
	f := loadFont(t, "spleen-6x12.bdf")
	c := New(64, 32)
	c.Fill(c.Rect, color.RGBA{0, 0, 0, 255})
	c.TakeDirty()
	w := c.Text(4, 8, f, "Hi", color.RGBA{255, 0, 0, 255})
	if w != 12 {
		t.Fatalf("width = %d", w)
	}
	d := c.TakeDirty()
	if len(d) != 1 || d[0] != image.Rect(4, 8, 16, 20) {
		t.Fatalf("dirty = %v", d)
	}
	red := 0
	for y := 8; y < 20; y++ {
		for x := 4; x < 16; x++ {
			if c.RGBAAt(x, y) == (color.RGBA{255, 0, 0, 255}) {
				red++
			}
		}
	}
	if red < 10 {
		t.Fatalf("only %d red pixels drawn", red)
	}
	if got := Fit("abcdefgh", 5); got != "abc.." {
		t.Errorf("Fit = %q", got)
	}
	if got := Fit("abc", 5); got != "abc" {
		t.Errorf("Fit short = %q", got)
	}
	lines := Wrap("Street Fighter III 3rd Strike: Fight for the Future", 17, 2)
	if len(lines) != 2 || lines[0] != "Street Fighter" || len(lines[1]) > 17 {
		t.Errorf("Wrap = %q", lines)
	}
	if got := Wrap("short", 17, 2); len(got) != 1 || got[0] != "short" {
		t.Errorf("Wrap short = %q", got)
	}
}

func TestDirtyMerge(t *testing.T) {
	c := New(100, 100)
	for i := 0; i < 20; i++ {
		c.Dirty(image.Rect(i*4, i*4, i*4+2, i*4+2))
	}
	d := c.TakeDirty()
	if len(d) > MaxDirty {
		t.Fatalf("%d dirty rects, cap is %d", len(d), MaxDirty)
	}
	u := d[0]
	for _, r := range d[1:] {
		u = u.Union(r)
	}
	if !u.Eq(image.Rect(0, 0, 78, 78)) {
		t.Fatalf("union = %v", u)
	}
}

func TestRotate(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	cols := []color.RGBA{{1, 0, 0, 255}, {2, 0, 0, 255}, {3, 0, 0, 255}, {4, 0, 0, 255}, {5, 0, 0, 255}, {6, 0, 0, 255}, {7, 0, 0, 255}, {8, 0, 0, 255}}
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			src.SetRGBA(x, y, cols[y*4+x])
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, 2, 4)) // physical W=2, H=4
	r := RotateAll(dst, src, RotLeft)
	if !r.Eq(dst.Rect) {
		t.Fatalf("left rect = %v", r)
	}
	// left: px = W-1-ly, py = lx. src(0,0) -> (1,0); src(3,1) -> (0,3)
	if dst.RGBAAt(1, 0) != cols[0] || dst.RGBAAt(0, 3) != cols[7] || dst.RGBAAt(1, 3) != cols[3] {
		t.Fatalf("left mapping wrong: %v", dst.Pix)
	}
	dst2 := image.NewRGBA(image.Rect(0, 0, 2, 4))
	RotateAll(dst2, src, RotRight)
	// right: px = ly, py = H-1-lx. src(0,0) -> (0,3); src(3,1) -> (1,0)
	if dst2.RGBAAt(0, 3) != cols[0] || dst2.RGBAAt(1, 0) != cols[7] {
		t.Fatalf("right mapping wrong: %v", dst2.Pix)
	}
	// partial rect: only its pixels move, returned rect covers them
	dst3 := image.NewRGBA(image.Rect(0, 0, 2, 4))
	pr := RotateRect(dst3, src, image.Rect(1, 0, 3, 1), RotLeft) // src (1,0),(2,0) -> (1,1),(1,2)
	if !pr.Eq(image.Rect(1, 1, 2, 3)) || dst3.RGBAAt(1, 1) != cols[1] || dst3.RGBAAt(1, 2) != cols[2] {
		t.Fatalf("partial left: rect %v pix %v", pr, dst3.Pix)
	}
}

func TestBlitAlpha(t *testing.T) {
	c := New(4, 4)
	c.Fill(c.Rect, color.RGBA{0, 0, 0, 255})
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{200, 100, 0, 255})
	src.SetRGBA(1, 1, color.RGBA{100, 50, 0, 128}) // premultiplied half
	c.BlitAlpha(image.Pt(1, 1), src)
	if c.RGBAAt(1, 1) != (color.RGBA{200, 100, 0, 255}) {
		t.Fatalf("opaque pixel = %v", c.RGBAAt(1, 1))
	}
	p := c.RGBAAt(2, 2)
	if p.R < 95 || p.R > 105 || p.A != 255 {
		t.Fatalf("blended pixel = %v", p)
	}
	if c.RGBAAt(2, 1) != (color.RGBA{0, 0, 0, 255}) {
		t.Fatalf("transparent pixel changed: %v", c.RGBAAt(2, 1))
	}
}

func BenchmarkTextScreen(b *testing.B) {
	bts, _ := os.ReadFile("../fonts/spleen-6x12.bdf")
	f, _ := ParseBDF(bts)
	c := New(320, 240)
	line := "Street Fighter III 3rd Strike: Fight for the Future 2026-09-05"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for y := 0; y < 20; y++ {
			c.Text(0, y*12, f, line[:53], color.RGBA{255, 255, 255, 255})
		}
		c.TakeDirty()
	}
}

func BenchmarkRotateFrame(b *testing.B) {
	src := image.NewRGBA(image.Rect(0, 0, 240, 320))
	dst := image.NewRGBA(image.Rect(0, 0, 320, 240))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RotateAll(dst, src, RotLeft)
	}
}
