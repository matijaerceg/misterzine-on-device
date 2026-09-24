package mister

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadIniAppliesSectionsInFileOrderLikeMain(t *testing.T) {
	p := filepath.Join(t.TempDir(), "MiSTer.ini")
	for _, tc := range []struct {
		name, content string
		want          IniSettings
	}{
		{"global only", "[MiSTer]\nosd_rotate=1\ndirect_video=1\n",
			IniSettings{OSDRotate: 1, DirectVideo: 1, FBTerminal: 1, Found: true}},
		{"menu overrides global", "[MiSTer]\nosd_rotate=1\n[Menu]\nosd_rotate=2\n",
			IniSettings{OSDRotate: 2, FBTerminal: 1, Found: true}},
		{"misterzine after menu wins", "[MiSTer]\nvideo_mode=8\n[Menu]\ndirect_video=0\nosd_rotate=0\n[MisterZine]\ndirect_video=1\nosd_rotate=2\n",
			IniSettings{OSDRotate: 2, DirectVideo: 1, FBTerminal: 1, VideoMode: "8", Found: true}},
		// Main fixes the video mode in video_init before the MGL's
		// setname is parsed, so a video_mode under [MisterZine] is not
		// the running mode whatever this returns. Only the probe in
		// fb.go decides; this records the precedence, not an effect.
		{"a later section still wins the value", "[MiSTer]\nvideo_mode=8\n[MisterZine]\nvideo_mode=14\n",
			IniSettings{FBTerminal: 1, VideoMode: "14", Found: true}},
		{"menu_pal is read", "[MiSTer]\ndirect_video=1\nmenu_pal=1\n",
			IniSettings{DirectVideo: 1, MenuPal: 1, FBTerminal: 1, Found: true}},
		{"a modeline keeps its flags", "[MiSTer]\nvideo_mode=1280,24,16,40,1440,3,5,33,120750,pr ; 1440p\n",
			IniSettings{FBTerminal: 1, VideoMode: "1280,24,16,40,1440,3,5,33,120750,pr", Found: true}},
		{"menu after misterzine wins", "[misterzine]\ndirect_video=1\n[Menu]\ndirect_video=0\n",
			IniSettings{FBTerminal: 1, Found: true}},
		{"other cores ignored", "[Menu]\ndirect_video=1\n[Genesis]\ndirect_video=0\nosd_rotate=1\n[video=640x480]\nvga_scaler=1\n",
			IniSettings{DirectVideo: 1, FBTerminal: 1, Found: true}},
		{"wildcard prefix", "[Mister*]\nosd_rotate=1\n[Zaxxon*]\nosd_rotate=2\n",
			IniSettings{OSDRotate: 1, FBTerminal: 1, Found: true}},
		{"comments and spacing", " [ Menu ] ; the menu\n fb_terminal = 0 ; console off\n",
			IniSettings{FBTerminal: 0, Found: true}},
		{"headerless values are global", "osd_rotate=2\n",
			IniSettings{OSDRotate: 2, FBTerminal: 1, Found: true}},
	} {
		if err := os.WriteFile(p, []byte(tc.content), 0600); err != nil {
			t.Fatal(err)
		}
		if got := ReadIni(p); got != tc.want {
			t.Errorf("%s: got %+v want %+v", tc.name, got, tc.want)
		}
	}
	if got := ReadIni(p + ".absent"); got != (IniSettings{FBTerminal: 1}) {
		t.Errorf("absent file: got %+v", got)
	}
}

func TestAnalogVisibleNeedsScalerOrForcedDirectVideo(t *testing.T) {
	for _, tc := range []struct {
		name          string
		s             IniSettings
		visible, auto bool
	}{
		{"hdmi defaults", IniSettings{}, false, false},
		{"vga scaler", IniSettings{VGAScaler: 1}, true, false},
		{"direct video on", IniSettings{DirectVideo: 1}, true, false},
		{"direct video auto (MiSTercade INI)", IniSettings{DirectVideo: 2}, false, true},
		{"auto with scaler", IniSettings{DirectVideo: 2, VGAScaler: 1}, true, true},
	} {
		if got := tc.s.AnalogVisible(); got != tc.visible {
			t.Errorf("%s: AnalogVisible=%v want %v", tc.name, got, tc.visible)
		}
		if got := tc.s.DirectVideoAuto(); got != tc.auto {
			t.Errorf("%s: DirectVideoAuto=%v want %v", tc.name, got, tc.auto)
		}
	}
}
