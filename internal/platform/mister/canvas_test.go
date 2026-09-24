package mister

import "testing"

func TestFitCanvas(t *testing.T) {
	for _, c := range []struct{ nw, nh, w, h int }{
		{640, 240, 320, 240},   // 240p CRT (direct video)
		{640, 288, 384, 288},   // 288p CRT: one factor leaves 128 px bars; DirectVideoCanvas handles direct video
		{640, 480, 320, 240},   // 480 CRT
		{1280, 720, 320, 240},  // 720p: x3 fills already
		{1920, 1080, 360, 270}, // 1080p: x4 = 1440x1080
		// No display reaches this function with these two. Main's
		// widest frame is 2048 wide (1440p and 4K arrive halved, and
		// repeated on the way out), and it halves the framebuffer
		// again above 1920x1080, so 1440p asks with 1280x720 and 4K
		// with 960x540. Both are kept as plain arithmetic cover; the
		// rows below them are the ones those displays use.
		{2560, 1440, 320, 240},
		{3840, 2160, 320, 240},
		{1280, 720, 320, 240}, // what a 1440p display actually asks
		{960, 540, 360, 270},  // what a 4K display actually asks
		{1024, 768, 340, 256}, // x3 = 1020x768
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

// Main's direct-video modes (video.cpp tvmodes): 640x240 at 60 Hz, 640x288
// with menu_pal=1, and their scandoubled 480 and 576 line forms, which
// Main scales with one factor and so keep FitCanvas.
func TestDirectVideoCanvas(t *testing.T) {
	for _, c := range []struct{ nw, nh, w, h int }{
		{640, 240, 320, 240}, // x2 across, x1 down
		{640, 288, 320, 288}, // x2 across fills the line; FitCanvas's 384 fits only once
		{768, 288, 384, 288}, // x2 across
		{720, 240, 360, 240},
		{1024, 288, 340, 288}, // x3 = 1020
		{640, 480, 320, 240},  // scandoubled: FitCanvas
		{640, 576, 320, 240},  // scandoubled PAL: FitCanvas
		{320, 240, 320, 240},
		{300, 240, 320, 240}, // too narrow: FitCanvas
	} {
		if w, h := DirectVideoCanvas(c.nw, c.nh); w != c.w || h != c.h {
			t.Errorf("DirectVideoCanvas(%d, %d) = %dx%d, want %dx%d", c.nw, c.nh, w, h, c.w, c.h)
		}
	}
}

// Every framebuffer Main can hand us (the output mode, halved above
// 1920x1080 and halved again in height where the display repeats pixels).
func TestFullCanvas(t *testing.T) {
	for _, c := range []struct{ nw, nh, w, h int }{
		{1280, 720, 426, 240},  // 720p, and 1440p repeated
		{1920, 1080, 480, 270}, // 1080p
		{1024, 768, 341, 256},  // 1024x768, and 2048x1536 halved
		{1366, 768, 455, 256},
		{1600, 900, 533, 300},
		{1280, 1024, 320, 256},
		{1680, 1050, 420, 262},
		{960, 720, 320, 240}, // 1920x1440 halved
		{960, 540, 480, 270}, // 4K repeated
		{960, 600, 480, 300}, // 1920x1200 halved
		{800, 600, 400, 300},
		{640, 480, 320, 240},
		{1024, 600, 512, 300},
		{720, 480, 360, 240},
		{720, 576, 360, 288},
		{640, 240, 320, 240},   // 240p CRT: FitCanvas
		{640, 288, 384, 288},   // 288p CRT: FitCanvas (the host uses DirectVideoCanvas under direct video)
		{860, 360, 320, 240},   // 3440x1440 repeated: under 480, so FitCanvas
		{1919, 1079, 479, 269}, // an odd timing still fills
	} {
		w, h := FullCanvas(c.nw, c.nh)
		if w != c.w || h != c.h {
			t.Errorf("FullCanvas(%d,%d)=%dx%d want %dx%d", c.nw, c.nh, w, h, c.w, c.h)
			continue
		}
		// Every display reads at much the same text size.
		if h < 240 || h > 300 {
			t.Errorf("FullCanvas(%d,%d)=%dx%d: %d rows is outside 240-300", c.nw, c.nh, w, h, h)
		}
		if c.nh < 480 {
			continue
		}
		// One factor for both axes, falling short of the display by
		// less than one canvas pixel on each.
		filled := false
		for k := 1; k <= c.nh; k++ {
			if c.nw/k == w && c.nh/k == h {
				filled = true
				break
			}
		}
		if !filled {
			t.Errorf("FullCanvas(%d,%d)=%dx%d is not the framebuffer divided by one factor", c.nw, c.nh, w, h)
		}
	}
}
