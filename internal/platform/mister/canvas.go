package mister

// FitCanvas picks the canvas for a display of nativeW x nativeH. Main places
// a framebuffer in the video mode at one integer factor for both axes, the
// largest that fits, centred (video.cpp, fb_cmd1): a 320x240 canvas on
// 1080p lands at x4 as 1280x960 with 60 px bars above and below, whatever
// vscale_mode says. The smallest height from 240 up that divides the mode's
// height, 4:3 wide, fills it instead: 360x270 on 1080p, 340x256 on
// 1024x768, 400x300 on 1600x900 and 800x600. Where 320x240 already fits
// (240p and 480 CRT modes, 720p, and the 1280x720 framebuffer a 1440p
// display asks with) nothing changes, and a mode no candidate divides keeps
// the classic size too. A 4K display asks with 960x540 and gets 360x270.
func FitCanvas(nativeW, nativeH int) (int, int) {
	for _, h := range []int{240, 256, 270, 288, 300} {
		if nativeH%h != 0 {
			continue
		}
		k := nativeH / h
		w := (h * 4 / 3) &^ 3 // 4:3, a multiple of 4
		if w*k <= nativeW {
			return w, h
		}
	}
	return 320, 240
}

// DirectVideoCanvas picks the canvas for a direct-video mode under 300
// lines, where Main scales the framebuffer with a separate integer factor
// per axis (video.cpp, fb_cmd1: divx and divy) instead of the one factor
// FitCanvas assumes. The canvas keeps the mode's full height and takes the
// widest width from 320 up to 400 that Main can multiply into the mode's
// width: 320x288 on the 640x288 mode menu_pal=1 gives, doubled across to
// fill the line. FitCanvas chose 384x288 there, which only fits 640 once and
// left the picture centred at 60% of the width (report 9CPF, September
// 2026). The 640x240 mode keeps 320x240, as before. A mode outside 240-300
// lines, or too narrow for a 320-wide canvas, keeps FitCanvas.
func DirectVideoCanvas(nativeW, nativeH int) (int, int) {
	if nativeH < 240 || nativeH > 300 {
		return FitCanvas(nativeW, nativeH)
	}
	for k := 1; nativeW/k >= 320; k++ {
		if w := nativeW / k; w <= 400 {
			return w &^ 3, nativeH
		}
	}
	return FitCanvas(nativeW, nativeH)
}

// fbRequest is the framebuffer to ask Main for so that a canvas of
// canvasW x canvasH keeps square pixels on the screen.
//
// Main sizes the framebuffer window with one integer factor for both axes
// (video.cpp, video_cmd) and takes no account of pixel repetition, so on a
// repeating display every framebuffer pixel covers two screen pixels
// across and one down: the picture comes out twice as wide as it should
// and cannot reach the full height of the mode. Asking for twice the
// height cancels that. The canvas itself is unchanged - Present writes
// each of its rows to two framebuffer rows - so the layout, the text size
// and the safe zone stay exactly as they are on every other display.
func fbRequest(canvasW, canvasH int, pr bool) (int, int) {
	if pr {
		return canvasW, canvasH * 2
	}
	return canvasW, canvasH
}

// fullTargetH is the canvas height Full display aims for. Text keeps its
// pixel size on every canvas, so the canvas height is what sets how large
// it looks: 270 rows is 1080p's exact fit, and FitCanvas stays within
// 240-300 everywhere, so both choices read at much the same size.
const fullTargetH = 270

// FullCanvas fills both axes with square, integer-scaled pixels at a text
// size close to every other display's: of the factors that leave at least
// 240 rows, it takes the one whose height lands nearest fullTargetH.
//
// The canvas is the framebuffer divided by that factor and rounded down, so
// it can fall short by less than one canvas pixel on each axis, and Main
// centres it: 426x240 covers 1278 of 720p's 1280 columns. Demanding an
// exact division is what used to let the text size wander, since 1280 has
// no factor of 3: 720p fell back to 640x360 and 1600x900 to 800x450, text
// at three quarters of 1080p's size or less, and up to three times the
// drawing.
//
// Low resolution CRT modes keep FitCanvas. The bounds keep an unusual
// timing from allocating a native-resolution UI just to remove bars.
func FullCanvas(nativeW, nativeH int) (int, int) {
	if nativeH < 480 || nativeW <= 0 {
		return FitCanvas(nativeW, nativeH)
	}
	bestW, bestH, bestD := 0, 0, -1
	for k := 1; nativeH/k >= 240; k++ {
		w, h := nativeW/k, nativeH/k
		if w < 320 || w > 960 || h > 600 {
			continue
		}
		d := h - fullTargetH
		if d < 0 {
			d = -d
		}
		if bestD < 0 || d < bestD { // k rises, so a tie keeps the taller
			bestW, bestH, bestD = w, h, d
		}
	}
	if bestD < 0 {
		return FitCanvas(nativeW, nativeH)
	}
	return bestW, bestH
}
