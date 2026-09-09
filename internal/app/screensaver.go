package app

import (
	"image"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

const saverFrame = time.Second / 30

var saverValues = []string{"off", "1", "2", "5", "10"}

type saverKey struct {
	key  platform.Key
	code uint16
}

type screensaver struct {
	active    bool
	lastInput time.Time
	started   time.Time
	next      time.Time
	travel    int
	mask      *image.Alpha
	waking    map[saverKey]bool
}

func (a *App) Screensaver() string {
	switch a.cfg.Screensaver {
	case "off", "2", "5", "10":
		return a.cfg.Screensaver
	default:
		return "1"
	}
}

func (a *App) ScreensaverActive() bool { return a.saver.active }

func (a *App) saverDelay() time.Duration {
	switch a.Screensaver() {
	case "off":
		return 0
	case "2":
		return 2 * time.Minute
	case "5":
		return 5 * time.Minute
	case "10":
		return 10 * time.Minute
	default:
		return time.Minute
	}
}

// Swallow the wake press, its duplicates and its release before normal actions
// see them. In particular, waking with Start must never launch a game, and
// holding B must not cancel an update until B has been released and pressed again.
func (a *App) handleSaverInput(ev platform.Event) bool {
	if ev.Key == platform.KeyNone && ev.Text == 0 {
		return false
	}
	at := ev.At
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	if at.After(a.saver.lastInput) {
		a.saver.lastInput = at
	}
	id := saverKey{key: ev.Key}
	if ev.Key == platform.KeyOther {
		id.code = ev.Code
	}
	if a.saver.waking[id] {
		if !ev.Pressed {
			delete(a.saver.waking, id)
		}
		return true
	}
	if !a.saver.active {
		return false
	}
	if !ev.Pressed {
		delete(a.down, ev.Key)
		return true // releasing the preview button doesn't dismiss the preview
	}
	a.saver.active = false
	a.rep = repeater{}
	a.down = map[platform.Key]bool{}
	a.updateView.backAt = time.Time{}
	if ev.Key != platform.KeyNone { // harness text events have no release
		a.saver.waking = map[saverKey]bool{id: true}
	}
	a.all = true
	return true
}

func (a *App) startSaver(now time.Time) {
	a.saver.active = true
	a.saver.started = now
	a.saver.next = now.Add(saverFrame)
	a.saver.travel = 0
	a.rep = repeater{}
	a.all = true
}

func (a *App) nextSaverTick() time.Time {
	if a.saver.active {
		return a.saver.next
	}
	if delay := a.saverDelay(); delay > 0 && len(a.down) == 0 && len(a.saver.waking) == 0 {
		return a.saver.lastInput.Add(delay)
	}
	return time.Time{}
}

func (a *App) tickSaver(now time.Time) bool {
	next := a.nextSaverTick()
	if next.IsZero() || now.Before(next) {
		return false
	}
	if !a.saver.active {
		a.startSaver(now)
	} else {
		a.saver.travel = int(now.Sub(a.saver.started) / saverFrame)
		a.saver.next = now.Add(saverFrame)
		a.all = true
	}
	return true
}

func (a *App) saverMask(h int) *image.Alpha {
	if a.saver.mask != nil && a.saver.mask.Rect.Dy() == h {
		return a.saver.mask
	}
	const word = "MISTERZINE"
	font := a.body
	// Crop the font's blank ascender/descender padding. M has ink on every
	// remaining row, so a complete horizontal pass blacks every screen pixel.
	top, bottom := font.H, 0
	for _, ch := range []byte(word) {
		for y, bits := range font.Glyph(ch) {
			if bits != 0 {
				top = min(top, y)
				bottom = max(bottom, y+1)
			}
		}
	}
	inkH := bottom - top
	w := len(word) * font.W * h / inkH
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := top + y*inkH/h
		for x := 0; x < w; x++ {
			sx := x * inkH / h
			if font.Glyph(word[sx/font.W])[sy]&(0x80>>uint(sx%font.W)) != 0 {
				m.Pix[y*m.Stride+x] = 255
			}
		}
	}
	a.saver.mask = m
	return m
}

func (a *App) paintSaver(c *gfx.Canvas) {
	m := a.saverMask(c.H())
	// One logical pixel per frame; the whole word enters at the right and
	// leaves at the left. Safe-zone insets don't clip this overlay.
	x0 := c.W() - a.saver.travel%(c.W()+m.Rect.Dx())
	for y := 0; y < c.H(); y++ {
		for x := 0; x < c.W(); x++ {
			i := c.PixOffset(x, y)
			sx := x - x0
			if sx >= 0 && sx < m.Rect.Dx() && m.Pix[y*m.Stride+sx] != 0 {
				c.Pix[i], c.Pix[i+1], c.Pix[i+2] = 0, 0, 0
			} else {
				c.Pix[i] /= 4
				c.Pix[i+1] /= 4
				c.Pix[i+2] /= 4
			}
		}
	}
	c.DirtyAll()
}
