package updater

import "bytes"

type outputEvent uint8

const (
	outputErrors outputEvent = iota
	linuxStart
	pocketStart
	linuxDone
	pocketDone
	rebootNeeded
	outputSuccess
	outputExtras
	outputCheck
	outputFiles
)

var outputMarkers = []struct {
	text   string
	prefix bool
	event  outputEvent
}{
	{"there were some errors in the updaters", false, outputErrors},
	{"linux will be updated", false, linuxStart},
	{"fetching the new linux image", false, linuxStart},
	{"hold your breath", false, linuxStart},
	{"installing analogue pocket firmware", false, pocketStart},
	{"linux has been updated", false, linuxDone},
	{"your pocket firmware is on", false, pocketDone},
	{"your pocket firmware could not", false, pocketDone},
	{"rebooting ", false, rebootNeeded},
	{"you should reboot", false, rebootNeeded},
	{"success! log", true, outputSuccess},
	{"success! more details at:", true, outputSuccess},
	{"running arcade organizer", false, outputExtras},
	{"backing up analogue pocket", false, outputExtras},
	{"running mister downloader", false, outputCheck},
	{"reading sections from", true, outputCheck},
	{"downloading ", true, outputFiles},
	{"fetching ", true, outputFiles},
	{"installing ", true, outputFiles},
	{"removing ", true, outputFiles},
	{"updates found", true, outputFiles},
}

// outputParser observes each marker once, when its last byte arrives. Its
// bounded look-behind survives pipe reads and display flushes, but resets at
// real line endings. Terminal escape state also survives split pipe reads.
type outputParser struct {
	tail   [64]byte // longer than any known marker
	n      int
	column int // capped once prefix matching is no longer possible
	escape byte
}

func (p *outputParser) write(s *State, b []byte) {
	for _, c := range b {
		switch p.escape {
		case 1: // ESC
			p.escape = 0
			if c == '[' {
				p.escape = 2
			} else if c == ']' {
				p.escape = 3
			}
			continue
		case 2: // CSI: parameters through the final command byte
			if c >= 0x40 && c <= 0x7e {
				p.escape = 0
			}
			continue
		case 3: // OSC: terminal title/link, terminated by BEL or ESC-backslash
			if c == '\a' {
				p.escape = 0
			} else if c == '\x1b' {
				p.escape = 4
			}
			continue
		case 4:
			p.escape = 3
			if c == '\\' || c == '\a' {
				p.escape = 0
			} else if c == '\x1b' {
				p.escape = 4
			}
			continue
		}
		if c == '\x1b' {
			p.escape = 1
			continue
		}
		if c == '\n' || c == '\r' {
			p.n, p.column = 0, 0
			continue
		}
		if c == '\t' {
			c = ' '
		}
		if c < 32 || c == 127 || p.column == 0 && c == ' ' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if p.n == len(p.tail) {
			copy(p.tail[:], p.tail[1:])
			p.n--
		}
		p.tail[p.n] = c
		p.n++
		p.column = min(p.column+1, len(p.tail)+1)
		for _, m := range outputMarkers {
			if c != m.text[len(m.text)-1] || m.prefix && p.column != len(m.text) {
				continue
			}
			if bytes.HasSuffix(p.tail[:p.n], []byte(m.text)) {
				s.observe(m.event)
			}
		}
	}
}
