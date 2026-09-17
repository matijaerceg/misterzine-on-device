package scan

import (
	"path"
	"strconv"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// rowFromAlt builds the row for an MRA the catalogue does not list. Every
// field comes from the header; what the header lacks stays empty so the
// facets file it under "unknown". A local row carries no release dates: the
// file's own stamp says nothing about the game, so date sorts place it last
// and the last-look marker ignores it.
func rowFromAlt(a Alt) data.Row {
	sn := identity(a.Setname)
	title := strings.TrimSpace(a.Name)
	if title == "" {
		title = strings.TrimSuffix(path.Base(a.Path), path.Ext(a.Path))
	}
	r := data.Row{
		Title:        title,
		Base:         "Arcade",
		Src:          data.SrcLocal,
		K:            data.LocalKey(sn),
		SN:           sn,
		Family:       a.Parent,
		Core:         a.RBF,
		MRA:          a.Path,
		Year:         strings.TrimSpace(a.Year),
		Manufacturer: strings.TrimSpace(a.Manufacturer),
		Reg:          strings.TrimSpace(a.Region),
		Rot:          mraRotation(a.Rotation),
		Plr:          players(a.Players),
	}
	if c := strings.TrimSpace(a.Category); c != "" {
		r.Note = "MRA category: " + c
	}
	joy := strings.TrimSpace(a.Joystick)
	if n, ok := buttonCount(a); ok {
		r.Buttons = &n
		if joy != "" {
			r.Ctl = joy + " · " + buttonsWord(n)
		} else {
			r.Ctl = buttonsWord(n)
		}
	} else {
		r.Ctl = joy
	}
	return r
}

// mraRotation maps an MRA <rotation> onto the catalogue's strings, so the
// rotation facet and the rotation chip read a local row like any other.
func mraRotation(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "vertical") && strings.Contains(l, "ccw"):
		return "Vertical (CCW)"
	case strings.Contains(l, "vertical") && strings.Contains(l, "cw"):
		return "Vertical (CW)"
	case strings.Contains(l, "vertical"):
		return "Vertical"
	case strings.Contains(l, "horizontal") && strings.Contains(l, "180"):
		return "Horizontal (180)"
	case strings.Contains(l, "horizontal"):
		return "Horizontal"
	}
	return ""
}

// players keeps the leading number of "<players>2 (alternating)</players>".
func players(s string) string {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i]
}

// buttonCount prefers <num_buttons>; otherwise it counts <buttons names>
// entries that are game buttons rather than start, coin or service inputs.
func buttonCount(a Alt) (int, bool) {
	if a.NumButtons > 0 {
		return a.NumButtons, true
	}
	if a.ButtonNames == nil {
		return 0, false
	}
	n := 0
	for _, name := range a.ButtonNames {
		l := strings.ToLower(name)
		switch {
		case l == "" || l == "-" || l == "n/a" || l == "none":
		case strings.HasPrefix(l, "start"), strings.HasPrefix(l, "coin"),
			l == "pause", l == "service", l == "test", l == "tilt", l == "select":
		default:
			n++
		}
	}
	return n, true
}

func buttonsWord(n int) string {
	if n == 1 {
		return "1 button"
	}
	return strconv.Itoa(n) + " buttons"
}
