package mister

import (
	"strconv"
	"strings"
)

// MayRepeatPixels reports whether MiSTer.ini's video_mode could name a mode
// the HDMI transmitter sends with pixel repetition: above 2048 pixels the
// dot clock is out of reach, so Main halves the active width and asks the
// display to double every pixel back. 2560x1440 is sent as 1280x1440 that
// way, and the picture the scaler draws is then half as wide as the one the
// display shows.
//
// This only decides whether asking the hardware is worth a round trip. The
// INI is not evidence of what the scaler is doing now: Main reads it once at
// startup, so a file edited since boot describes a mode that is not running,
// and a video_mode under [MisterZine] never ran at all - Main fixes the mode
// in video_init before the MGL's setname is parsed (user_io.cpp, video_init
// at 1545 and mgl_parse at 1550). A wrong yes costs one probe; a wrong no
// leaves the picture as it was. So this leans towards yes, and FB.probeFrame
// decides.
//
// An absent video_mode is the one certain no: Main then falls back on the
// display's preferred mode, and get_edid_vmode refuses anything wider than
// 2048 (video.cpp).
func MayRepeatPixels(raw string) bool {
	tok := strings.Split(strings.TrimSpace(raw), ",")
	for i := range tok {
		tok[i] = strings.TrimSpace(tok[i])
	}
	if len(tok) == 0 || tok[0] == "" {
		return false
	}

	// Main reads leading numbers and treats the rest as named flags
	// (video.cpp, parse_custom_video_mode), accepting a fractional
	// refresh rate as the third number.
	var val []uint64
	for _, t := range tok {
		n, err := strconv.ParseUint(t, 0, 32)
		if err != nil {
			break
		}
		val = append(val, n)
	}
	if len(val) == 2 && len(tok) > 2 {
		if _, err := strconv.ParseFloat(tok[2], 64); err == nil {
			val = append(val, 0)
		}
	}
	for _, f := range tok[len(val):] {
		if strings.EqualFold(f, "pr") {
			return true // a modeline that asks for repetition outright
		}
	}

	switch len(val) {
	case 1:
		// The preset table's only repeating entry is 14, 2560x1440.
		// Main replaces an index past the table with 0.
		return val[0] == 14
	case 3:
		// video_calculate_cvt repeats anything wider than 2048.
		return val[0] > 2048
	case 9, 11:
		return false // a modeline repeats only when it says so
	}
	if len(val) >= 21 {
		return false
	}
	// Main either rejects the line or reads it in a way this does not
	// model. Ask the hardware rather than guess.
	return true
}

// repeatsPixels compares the framebuffer Main built for itself with the
// frame the scaler is driving, and reports whether every pixel is doubled
// horizontally on the way out.
//
// Main divides the frame by fb_size for its own framebuffer and divides the
// height once more when the display repeats pixels, so that its menu keeps
// square pixels (video.cpp, video_fb_config: fb_scale_y is fb_scale*2 when
// pr is set). That second division is the whole signal: the frame is as many
// times taller than the framebuffer as it is wide, unless pixels repeat, in
// which case it is twice as tall again. The probe's divisor cancels out of
// the comparison, and a tolerance absorbs its rounding.
func repeatsPixels(nativeW, nativeH, probeW, probeH int) bool {
	if nativeW <= 0 || nativeH <= 0 || probeW <= 0 || probeH <= 0 {
		return false
	}
	num, den := probeH*nativeW, probeW*nativeH
	return num*10 >= den*19 && num*10 <= den*21 // 2.0, within a tenth
}
