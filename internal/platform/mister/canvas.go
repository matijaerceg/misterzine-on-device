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

// FullCanvas fills both axes with square, integer-scaled pixels. Keep low
// resolution CRT modes on the existing path. Bound the canvas for unusual
// timings rather than allocating a native-resolution UI just to remove bars.
func FullCanvas(nativeW, nativeH int) (int, int) {
	if nativeH < 480 || nativeW <= 0 {
		return FitCanvas(nativeW, nativeH)
	}
	for k := nativeH / 240; k >= 1; k-- {
		if nativeW%k != 0 || nativeH%k != 0 {
			continue
		}
		w, h := nativeW/k, nativeH/k
		if w >= 320 && w <= 960 && h <= 600 {
			return w, h
		}
	}
	return FitCanvas(nativeW, nativeH)
}
