//go:build linux

package mister

// Printable keys on a US-layout keyboard. Letters stay lowercase because
// title matching ignores case. Gamepad events from Main's virtual keyboard
// remain navigation, particularly Space/Y (sort) and Enter/A (details).
func (d *device) keyboardText(code uint16) rune {
	if d.pad || d.name == "MiSTer virtual input" {
		return 0
	}
	for _, mod := range []uint16{29, 97, 56, 100, 125, 126} { // Ctrl, Alt, Meta
		if _, held := d.held[mod]; held {
			return 0
		}
	}
	for _, row := range []struct {
		first uint16
		keys  string
	}{{16, "qwertyuiop"}, {30, "asdfghjkl"}, {44, "zxcvbnm"}} {
		if code >= row.first && int(code-row.first) < len(row.keys) {
			return rune(row.keys[code-row.first])
		}
	}
	_, leftShift := d.held[42]
	_, rightShift := d.held[54]
	shift := leftShift || rightShift
	if code >= 2 && code <= 11 {
		if shift {
			return rune("!@#$%^&*()"[code-2])
		}
		return rune("1234567890"[code-2])
	}
	if pair := punctuation[code]; pair != "" {
		if shift {
			return rune(pair[1])
		}
		return rune(pair[0])
	}
	return 0
}

var punctuation = map[uint16]string{
	12: "-_", 13: "=+", 26: "[{", 27: "]}", 39: ";:", 40: "'\"",
	41: "`~", 43: "\\|", 51: ",<", 52: ".>", 53: "/?", 57: "  ",
	55: "**", 74: "--", 78: "++", 98: "//",
}
