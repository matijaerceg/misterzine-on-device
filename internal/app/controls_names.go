package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"image"
	"strings"
	"time"
)

type nameGroup struct {
	title  string
	labels []string
}

var nameGroups = []nameGroup{
	{"Letters", strings.Fields("A B C D E F G H I J K L M N O P Q R S T U V W X Y Z")},
	{"Numbers", strings.Fields("1 2 3 4 5 6 7 8 9 0")},
	{"PlayStation", strings.Fields("Cross Circle Square Triangle L1 R1 L2 R2 L3 R3 Opt Share PS")},
	{"Colors", strings.Fields("Red Blue Green Yellow White Black Orange Purple Pink Gray")},
	{"Fighting", strings.Fields("LP MP HP LK MK HK P K PP KK PPP KKK A B C D S HS Dust Taunt")},
	{"Arcade", strings.Fields("Coin Start Service Test Tilt P1 P2 1P 2P Fire Jump Bomb Shot Turbo Auto Weapon Special Credit")},
	{"Pad", strings.Fields("Start Select Back Guide Home Menu L R LB RB LT RT LS RS Z ZL ZR C")},
	{"Keyboard", []string{"Enter", "Esc", "Tab", "Space", "Ctrl", "Shift", "Alt", "PgUp", "PgDn", "Home", "End", "Ins", "Del", "Bksp", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"}},
	{"Directions", strings.Fields("Up Down Left Right")},
}

func (a *App) holdText(c *gfx.Canvas, x, y, cols, line int, text string, at time.Time) int {
	lines := gfx.Wrap(text, cols, 4)
	total := 0
	for _, s := range lines {
		total += a.sm.Width(s)
	}
	remaining := 0
	if !at.IsZero() {
		remaining = int(min(int64(total), max(int64(0), a.controlNow().Sub(at).Nanoseconds()*int64(total)/int64(controlHoldDuration))))
	}
	for _, s := range lines {
		c.Text(x, y, a.sm, s, pal.Muted)
		w := a.sm.Width(s)
		if remaining > 0 {
			c.TextClip(x, y, a.sm, s, pal.Accent, image.Rect(x, y, x+min(w, remaining), y+a.sm.H))
		}
		remaining -= w
		y += line
	}
	return y
}
func displayButtonName(name string) string {
	switch name {
	case "Cross":
		return gfx.Cross
	case "Circle":
		return gfx.Circle
	case "Square":
		return gfx.Square
	case "Triangle":
		return gfx.Triangle
	}
	return name
}

func storedButtonName(s string) string {
	switch s {
	case gfx.Cross:
		return "Cross"
	case gfx.Circle:
		return "Circle"
	case gfx.Square:
		return "Square"
	case gfx.Triangle:
		return "Triangle"
	}
	return s
}
