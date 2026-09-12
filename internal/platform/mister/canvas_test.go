package mister

import "testing"

func TestFitCanvas(t *testing.T) {
	for _, c := range []struct{ nw, nh, w, h int }{
		{640, 240, 320, 240},   // 240p CRT (direct video)
		{768, 288, 384, 288},   // 288p CRT
		{640, 480, 320, 240},   // 480 CRT
		{1280, 720, 320, 240},  // 720p: x3 fills already
		{1920, 1080, 360, 270}, // 1080p: x4 = 1440x1080
		{2560, 1440, 320, 240}, // 1440p: x6 fills already
		{3840, 2160, 320, 240}, // 4K: x9
		{1024, 768, 340, 256},  // x3 = 1020x768
		{1366, 768, 340, 256},
		{1600, 900, 400, 300}, // x3 = 1200x900
		{800, 600, 400, 300},  // x2
		{1600, 1200, 320, 240},
		{1280, 1024, 320, 240}, // 256 rows would need 1360 px of width
		{1680, 1050, 320, 240}, // nothing divides 1050
		{720, 576, 320, 240},   // 288 rows would need 768 px of width
		{320, 240, 320, 240},   // a framebuffer left at our own size
	} {
		if w, h := FitCanvas(c.nw, c.nh); w != c.w || h != c.h {
			t.Errorf("FitCanvas(%d, %d) = %dx%d, want %dx%d", c.nw, c.nh, w, h, c.w, c.h)
		}
	}
}
