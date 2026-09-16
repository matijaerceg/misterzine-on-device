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

func TestFullCanvas(t *testing.T) {
	for _, c := range []struct{ nw, nh, w, h int }{
		{1920, 1080, 480, 270}, {1280, 720, 640, 360}, {2560, 1440, 512, 288},
		{3840, 2160, 480, 270}, {1920, 1200, 384, 240}, {1024, 768, 512, 384},
		{640, 480, 320, 240}, {640, 240, 320, 240}, {768, 288, 384, 288},
		{1366, 768, 683, 384}, {3440, 1440, 688, 288}, {1919, 1079, 320, 240},
	} {
		w, h := FullCanvas(c.nw, c.nh)
		if w != c.w || h != c.h {
			t.Errorf("FullCanvas(%d,%d)=%dx%d want %dx%d", c.nw, c.nh, w, h, c.w, c.h)
		}
		if c.nh >= 480 && c.nw != 1919 && (c.nw%w != 0 || c.nh%h != 0 || c.nw/w != c.nh/h) {
			t.Errorf("nonuniform or incomplete fill: %+v", c)
		}
	}
}
