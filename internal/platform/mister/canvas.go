package mister

// FitCanvas picks the canvas for a display of nativeW x nativeH. Main places
// a framebuffer in the video mode at one integer factor for both axes, the
// largest that fits, centred (video.cpp, fb_cmd1): a 320x240 canvas on
// 1080p lands at x4 as 1280x960 with 60 px bars above and below, whatever
// vscale_mode says. The smallest height from 240 up that divides the mode's
// height, 4:3 wide, fills it instead: 360x270 on 1080p, 340x256 on
// 1024x768, 400x300 on 1600x900 and 800x600. Where 320x240 already fits
// (240p and 480 CRT modes, 720p, 1440p, 4K) nothing changes, and a mode no
// candidate divides keeps the classic size too.
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
