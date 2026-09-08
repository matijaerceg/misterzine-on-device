package mister

import (
	"bufio"
	"os"
	"strings"
)

// IniSettings are the MiSTer.ini values the app cares about, with the
// [Menu] section overriding the global one the way Main applies it to the
// menu core.
type IniSettings struct {
	OSDRotate   int  // 0 none, 1 right (+90), 2 left (-90)
	DirectVideo int  // 0/1/2
	VGAScaler   int  // 0/1
	FBTerminal  int  // 1 default
	Found       bool // the file was read
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
	section := ""
	global := map[string]string{}
	menu := map[string]string{}
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
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch section {
		case "mister", "":
			global[k] = v
		case "menu":
			menu[k] = v
		}
	}
	get := func(key string, def int) int {
		v, ok := menu[key]
		if !ok {
			v, ok = global[key]
		}
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
	return s
}

// AnalogVisible reports whether the framebuffer can reach the analog port
// on this configuration: the scaler is routed to VGA, or direct video
// carries the framebuffer while a script runs. HDMI always shows it.
func (s IniSettings) AnalogVisible() bool {
	return s.VGAScaler == 1 || s.DirectVideo >= 1
}
