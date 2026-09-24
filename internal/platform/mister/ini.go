package mister

import (
	"bufio"
	"os"
	"strings"
)

// IniSettings are the MiSTer.ini values the app cares about, read the way
// Main reads them for the menu core loaded through MisterZine.mgl: the
// global [MiSTer] section, [Menu] (the core's own name) and [MisterZine]
// (the MGL's setname) all apply, later sections in the file overriding
// earlier ones.
type IniSettings struct {
	OSDRotate   int    // 0 none, 1 right (+90), 2 left (-90)
	DirectVideo int    // 0/1/2
	VGAScaler   int    // 0/1
	FBTerminal  int    // 1 default
	VideoMode   string // raw value; a preset number, a modeline or empty
	MenuPal     int    // 0/1: direct video runs the menu core at 50 Hz, 288 lines
	Found       bool   // the file was read
}

// iniSectionApplies reports whether Main applies a section of that name
// while MisterZine.mgl is loaded. Main matches [MiSTer], the original core
// name (Menu) and the setname override (misterzine), case-insensitively; a
// trailing * matches a prefix of either name. Values above the first section
// header are accepted as global.
func iniSectionApplies(name string) bool {
	name = strings.ToLower(name)
	if name == "" || name == "mister" {
		return true
	}
	for _, core := range []string{"menu", "misterzine"} {
		if name == core {
			return true
		}
		if p, ok := strings.CutSuffix(name, "*"); ok && strings.HasPrefix(core, p) {
			return true
		}
	}
	return false
}

// ReadIni parses the few keys we need from a MiSTer.ini file.
func ReadIni(path string) IniSettings {
	s := IniSettings{FBTerminal: 1}
	f, err := os.Open(path)
	if err != nil {
		return s
	}
	defer f.Close()
	s.Found = true
	applies := true
	values := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.IndexByte(line, ';'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			applies = iniSectionApplies(strings.TrimSpace(strings.Trim(line, "[]")))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || !applies {
			continue
		}
		values[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	get := func(key string, def int) int {
		v, ok := values[key]
		if !ok {
			return def
		}
		n := 0
		for _, c := range v {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	s.OSDRotate = get("osd_rotate", 0)
	s.DirectVideo = get("direct_video", 0)
	s.VGAScaler = get("vga_scaler", 0)
	s.FBTerminal = get("fb_terminal", 1)
	s.VideoMode = values["video_mode"]
	s.MenuPal = get("menu_pal", 0)
	return s
}

// AnalogVisible reports whether the framebuffer can reach the analog port
// on this configuration: the scaler is routed to VGA, or direct video
// carries the framebuffer while a script runs. HDMI always shows it.
// direct_video=2 is Main's auto mode: it only turns direct video on when
// an HDMI DAC reporting 1024x768 is attached, so on its own it promises
// nothing. MiSTercade's shipped INI uses 2 with nothing on HDMI, and the
// framebuffer then stays HDMI-only while the cabinet shows the menu core.
func (s IniSettings) AnalogVisible() bool {
	return s.VGAScaler == 1 || s.DirectVideo == 1
}

// DirectVideoAuto reports the auto direct-video setting, whose analog
// reach depends on a DAC that MisterZine cannot see.
func (s IniSettings) DirectVideoAuto() bool { return s.DirectVideo == 2 }
