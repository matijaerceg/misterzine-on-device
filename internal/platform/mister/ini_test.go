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
			IniSettings{OSDRotate: 2, DirectVideo: 1, FBTerminal: 1, Found: true}},
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
