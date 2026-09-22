package mister

import "testing"

func TestMayRepeatPixels(t *testing.T) {
	for _, c := range []struct {
		raw  string
		want bool
		why  string
	}{
		{"", false, "no video_mode: Main follows the display, which it refuses above 2048"},
		{"8", false, "1080p"},
		{"0", false, "720p"},
		{"12", false, "1920x1440, wide but sent whole"},
		{"13", false, "2048x1536, the widest mode sent whole"},
		{"14", true, "the only repeating preset"},
		{" 14 ", true, "spaces around the value"},
		{"14,", true, "a trailing separator must not lose the preset"},
		{"99", false, "past the table: Main uses 0"},
		{"2560,1440,60", true, "CVT wider than 2048"},
		{"2560,1440,59.94", true, "a fractional refresh rate is still the third number"},
		{"3840,2160,60", true, "4K"},
		{"3440,1440,60", true, "ultrawide"},
		{"2049,1440,60", true, "Main tests the width before rounding it down"},
		{"2048,1536,60", false, "exactly 2048 is not wider than 2048"},
		{"1920,1080,60", false, "an ordinary CVT request"},
		{"1280,24,16,40,1440,3,5,33,120750,pr", true, "a modeline asking for repetition"},
		{"1280,24,16,40,1440,3,5,33,120750", false, "the same modeline without the flag"},
		{"640,54,56,106,224,16,0,28,13764", false, "a CRT modeline"},
		{"1280,110,40,220,720,5,5,20,74250,0,1", false, "an eleven-field modeline"},
		{"abc", true, "unreadable: ask the hardware rather than guess"},
		{"1280,720", true, "two numbers alone are not a form we model"},
		{"4294967295", false, "a preset far past the table is still not 14"},
		{"0xFFFFFFFF", false, "the same in hex"},
	} {
		if got := MayRepeatPixels(c.raw); got != c.want {
			t.Errorf("MayRepeatPixels(%q) = %v, want %v (%s)", c.raw, got, c.want, c.why)
		}
	}
}

func TestRepeatsPixels(t *testing.T) {
	for _, c := range []struct {
		name                             string
		nativeW, nativeH, probeW, probeH int
		want                             bool
	}{
		// probe is the frame divided by 4; native is what the kernel
		// reported before it.
		{"1440p repeated", 1280, 720, 320, 360, true},
		{"720p", 1280, 720, 320, 180, false},
		{"1080p", 1920, 1080, 480, 270, false},
		{"4K repeated", 960, 540, 480, 540, true},
		{"1920x1440", 960, 720, 480, 360, false},
		{"1366x768, rounding in the probe", 1366, 768, 341, 192, false},
		{"1440p repeated with fb_size=2", 640, 360, 320, 360, true},
		{"nothing landed", 1280, 720, 0, 0, false},
		{"closed framebuffer", 0, 0, 320, 360, false},
	} {
		if got := repeatsPixels(c.nativeW, c.nativeH, c.probeW, c.probeH); got != c.want {
			t.Errorf("%s: repeatsPixels(%d, %d, %d, %d) = %v, want %v",
				c.name, c.nativeW, c.nativeH, c.probeW, c.probeH, got, c.want)
		}
	}
}

func TestFBRequest(t *testing.T) {
	for _, c := range []struct {
		w, h   int
		pr     bool
		rw, rh int
	}{
		{640, 360, false, 640, 360},
		{640, 360, true, 640, 720},
		{320, 240, true, 320, 480},
		{320, 240, false, 320, 240},
	} {
		rw, rh := fbRequest(c.w, c.h, c.pr)
		if rw != c.rw || rh != c.rh {
			t.Errorf("fbRequest(%d, %d, %v) = %dx%d, want %dx%d", c.w, c.h, c.pr, rw, rh, c.rw, c.rh)
		}
	}
}

// nativeFB mirrors Main's video_fb_config: the Linux framebuffer is halved
// above 1920x1080, and halved once more vertically on a repeating mode so
// that Main's own menu keeps square pixels.
func nativeFB(hact, vact int, pr bool) (int, int) {
	s := 1
	if hact*vact > 1920*1080 {
		s = 2
	}
	sy := s
	if pr {
		sy = s * 2
	}
	return hact / s, vact / sy
}

// mainFBWindow mirrors Main's fb_cmd1 (video.cpp, video_cmd): the frame is
// clamped to the mode, one integer factor serves both axes, and the result
// is centred. It returns the rectangle in screen pixels, which is twice as
// wide as the scaler's frame when the display repeats pixels.
func mainFBWindow(hact, vact int, pr bool, w, h int) (int, int) {
	if w < 120 || w > hact {
		w = hact
	}
	if h < 120 || h > vact {
		h = vact
	}
	k := 1
	for w*(k+1) <= hact && h*(k+1) <= vact {
		k++
	}
	pw, ph := w*k, h*k
	if pr {
		pw *= 2
	}
	return pw, ph
}

// TestScreenGeometry walks the whole chain for every mode Main can drive:
// the framebuffer it builds, the probe it answers, the canvas we choose,
// the request we send and the rectangle it lands in. A repeating display
// must keep square pixels and reach the full height; an ordinary one must
// come out exactly as it did before any of this existed.
func TestScreenGeometry(t *testing.T) {
	for _, m := range []struct {
		name       string
		raw        string
		hact, vact int
		pr         bool
	}{
		{"720p", "0", 1280, 720, false},
		{"1080p", "8", 1920, 1080, false},
		{"1024x768", "1", 1024, 768, false},
		{"1920x1440", "12", 1920, 1440, false},
		{"2048x1536", "13", 2048, 1536, false},
		{"1440p", "14", 1280, 1440, true},
		{"1440p by CVT", "2560,1440,60", 1280, 1440, true},
		{"4K", "3840,2160,60", 1920, 2160, true},
		{"3440x1440 ultrawide", "3440,1440,60", 1720, 1440, true},
	} {
		fbW, fbH := nativeFB(m.hact, m.vact, m.pr)

		// A mode worth probing must probe as what it is, and a mode
		// not worth probing must not be one that repeats.
		mayProbe := MayRepeatPixels(m.raw)
		if m.pr && !mayProbe {
			t.Errorf("%s: video_mode %q would never be probed", m.name, m.raw)
		}
		pr := false
		if mayProbe {
			pr = repeatsPixels(fbW, fbH, m.hact/4, m.vact/4)
		}
		if pr != m.pr {
			t.Errorf("%s: detected repeats=%v, want %v", m.name, pr, m.pr)
		}

		for _, choice := range []struct {
			name  string
			pick  func(int, int) (int, int)
			fills bool
		}{
			{"fit", FitCanvas, true},
			{"full", FullCanvas, true},
			{"320x240", func(int, int) (int, int) { return 320, 240 }, false},
		} {
			cw, ch := choice.pick(fbW, fbH)
			rw, rh := fbRequest(cw, ch, pr)
			pw, ph := mainFBWindow(m.hact, m.vact, m.pr, rw, rh)
			if pw <= 0 || ph <= 0 {
				t.Fatalf("%s/%s: empty window", m.name, choice.name)
			}
			// Square pixels: a canvas pixel covers as many screen
			// pixels across as it does down.
			if pw%cw != 0 || ph%ch != 0 || pw/cw != ph/ch {
				t.Errorf("%s/%s: canvas %dx%d shown as %dx%d: pixels are %d:%d, want square",
					m.name, choice.name, cw, ch, pw, ph, pw/cw, ph/ch)
			}
			if ph > m.vact || pw > m.hact*2 {
				t.Errorf("%s/%s: %dx%d does not fit the %dx%d mode",
					m.name, choice.name, pw, ph, m.hact*2, m.vact)
			}
			if choice.fills && ph != m.vact {
				t.Errorf("%s/%s: %d of %d lines used; the picture should reach the full height",
					m.name, choice.name, ph, m.vact)
			}
		}
	}
}

// A display that does not repeat pixels must be asked for exactly the
// canvas, whatever the INI claims.
func TestOrdinaryDisplaysAreAskedForTheCanvasItself(t *testing.T) {
	for _, fb := range [][2]int{{1280, 720}, {1920, 1080}, {640, 240}, {960, 540}, {1366, 768}} {
		cw, ch := FullCanvas(fb[0], fb[1])
		if rw, rh := fbRequest(cw, ch, false); rw != cw || rh != ch {
			t.Errorf("%dx%d: request %dx%d changed to %dx%d", fb[0], fb[1], cw, ch, rw, rh)
		}
	}
}
