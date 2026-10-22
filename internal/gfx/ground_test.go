package gfx

import (
	"image"
	"image/color"
	"testing"
)

var groundBg = color.RGBA{29, 19, 48, 255}

func testGround(w, h int) *image.RGBA {
	g := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 7, 255})
		}
	}
	return g
}

func TestFillCopiesTheGround(t *testing.T) {
	c := New(40, 30)
	SetGround(groundBg, testGround(40, 30))
	defer SetGround(groundBg, nil)
	c.Fill(image.Rect(5, 6, 20, 12), groundBg)
	if got := c.RGBAAt(9, 8); got != (color.RGBA{9, 8, 7, 255}) {
		t.Fatalf("ground pixel %v", got)
	}
	// a sub-canvas keeps screen coordinates, so the pattern lines up
	sub := &Canvas{RGBA: c.Sub(image.Rect(10, 10, 30, 25))}
	sub.Fill(sub.Rect, groundBg)
	if got := c.RGBAAt(29, 24); got != (color.RGBA{29, 24, 7, 255}) {
		t.Fatalf("sub-canvas pixel %v", got)
	}
	// other colours stay flat
	c.Fill(image.Rect(0, 0, 3, 3), color.RGBA{1, 2, 3, 255})
	if got := c.RGBAAt(2, 2); got != (color.RGBA{1, 2, 3, 255}) {
		t.Fatalf("flat pixel %v", got)
	}
}

// The ground fill against the flat one, over a whole 320x240 screen: run
// the cross-compiled test binary on a board, Windows timings do not carry.
func BenchmarkFillFlat(b *testing.B) {
	c := New(320, 240)
	SetGround(groundBg, nil)
	for b.Loop() {
		c.Fill(c.Rect, groundBg)
		c.TakeDirty()
	}
}

func BenchmarkFillGround(b *testing.B) {
	c := New(320, 240)
	SetGround(groundBg, testGround(320, 240))
	defer SetGround(groundBg, nil)
	for b.Loop() {
		c.Fill(c.Rect, groundBg)
		c.TakeDirty()
	}
}

// a row at a time, as the list repaints its lines
func BenchmarkFillGroundRows(b *testing.B) {
	c := New(320, 240)
	SetGround(groundBg, testGround(320, 240))
	defer SetGround(groundBg, nil)
	for b.Loop() {
		for y := 0; y < 240; y += 12 {
			c.Fill(image.Rect(0, y, 320, y+12), groundBg)
		}
		c.TakeDirty()
	}
}

func BenchmarkFillFlatRows(b *testing.B) {
	c := New(320, 240)
	SetGround(groundBg, nil)
	for b.Loop() {
		for y := 0; y < 240; y += 12 {
			c.Fill(image.Rect(0, y, 320, y+12), groundBg)
		}
		c.TakeDirty()
	}
}
