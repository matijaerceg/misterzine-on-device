package mister

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Main keeps the alternative INI choice (OSD "Alt INI", 0..3) in a marker
// it writes to HPS memory: three magic bytes and the index, at this physical
// address. The marker survives core reloads, and Main's cfg_parse reads it
// on every core start, so the word names the INI Main itself is using.
const (
	altcfgPage   = 0x1FFFF000
	altcfgOffset = 0xF04
)

var altcfgMagic = [3]byte{0x34, 0x99, 0xBA}

// altcfgIndex decodes the marker bytes: the index when the magic matches,
// otherwise 0 (the primary MiSTer.ini).
func altcfgIndex(b []byte) int {
	if len(b) < 4 || b[0] != altcfgMagic[0] || b[1] != altcfgMagic[1] || b[2] != altcfgMagic[2] {
		return 0
	}
	return int(b[3])
}

// IniName mirrors Main's cfg_get_name: alt 0 is MiSTer.ini; alt 1..3 picks
// among the first three MiSTer_*.ini files in directory order, sorted
// case-insensitively. Only the name is returned; callers join it with card.
func IniName(card string, alt int) string {
	if alt <= 0 || alt > 3 {
		return "MiSTer.ini"
	}
	var names []string
	if d, err := os.Open(card); err == nil {
		entries, _ := d.Readdirnames(-1) // directory order, like readdir
		d.Close()
		for _, n := range entries {
			if len(n) > 11 && strings.EqualFold(n[:7], "MiSTer_") && strings.EqualFold(n[len(n)-4:], ".ini") {
				names = append(names, n)
				if len(names) == 3 {
					break
				}
			}
		}
	}
	sort.SliceStable(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
	if alt > len(names) {
		return "MiSTer.ini"
	}
	return names[alt-1]
}

// ActiveIni returns the INI file Main is currently using on this card and
// the alternative index it read (0 for the primary file).
func ActiveIni(card string) (string, int) {
	alt := activeAltcfg()
	return filepath.Join(card, IniName(card, alt)), alt
}
