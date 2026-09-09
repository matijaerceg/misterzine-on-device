package app

import (
	"image"
	"math"
	"sort"
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
	w := -3
	for _, ch := range []byte(word) {
		w += saverLetters[ch].width + 3
	}
	m := image.NewAlpha(image.Rect(0, 0, w*h/24, h))
	x := 0
	for _, ch := range []byte(word) {
		letter := saverLetters[ch]
		fillSaverPolygon(m, x, letter.outline, 255)
		for _, counter := range letter.counters {
			fillSaverPolygon(m, x, counter, 0)
		}
		x += letter.width + 3
	}
	a.saver.mask = m
	return m
}

// Broad geometric capitals drawn for this overlay, in 24-unit-high outlines.
// The full-height M and I strokes ensure a pass sweeps every row solid black.
type saverLetter struct {
	width    int
	outline  []image.Point
	counters [][]image.Point
}

var saverLetters = map[byte]saverLetter{
	'M': {24, []image.Point{{0, 24}, {0, 0}, {6, 0}, {12, 10}, {18, 0}, {24, 0}, {24, 24}, {18, 24}, {18, 10}, {12, 19}, {6, 10}, {6, 24}}, nil},
	'I': {6, []image.Point{{0, 0}, {6, 0}, {6, 24}, {0, 24}}, nil},
	'S': {20, []image.Point{{4, 0}, {16, 0}, {20, 4}, {20, 7}, {14, 7}, {14, 5}, {6, 5}, {6, 9}, {16, 9}, {20, 13}, {20, 20}, {16, 24}, {4, 24}, {0, 20}, {0, 17}, {6, 17}, {6, 19}, {14, 19}, {14, 15}, {4, 15}, {0, 11}, {0, 4}}, nil},
	'T': {20, []image.Point{{0, 0}, {20, 0}, {20, 6}, {13, 6}, {13, 24}, {7, 24}, {7, 6}, {0, 6}}, nil},
	'E': {18, []image.Point{{0, 0}, {18, 0}, {18, 6}, {6, 6}, {6, 9}, {16, 9}, {16, 15}, {6, 15}, {6, 18}, {18, 18}, {18, 24}, {0, 24}}, nil},
	'R': {22, []image.Point{{0, 0}, {17, 0}, {21, 4}, {21, 11}, {17, 15}, {22, 24}, {15, 24}, {10, 15}, {6, 15}, {6, 24}, {0, 24}}, [][]image.Point{{{6, 5}, {15, 5}, {15, 10}, {6, 10}}}},
	'Z': {20, []image.Point{{0, 0}, {20, 0}, {20, 5}, {8, 18}, {20, 18}, {20, 24}, {0, 24}, {0, 19}, {12, 6}, {0, 6}}, nil},
	'N': {22, []image.Point{{0, 24}, {0, 0}, {6, 0}, {16, 14}, {16, 0}, {22, 0}, {22, 24}, {16, 24}, {6, 10}, {6, 24}}, nil},
}

func fillSaverPolygon(m *image.Alpha, offset int, points []image.Point, ink uint8) {
	scale := float64(m.Rect.Dy()) / 24
	xs := make([]float64, 0, len(points))
	for y := 0; y < m.Rect.Dy(); y++ {
		py := (float64(y) + .5) / scale
		xs = xs[:0]
		for i, p := range points {
			q := points[(i+1)%len(points)]
			if (float64(p.Y) <= py && float64(q.Y) > py) || (float64(q.Y) <= py && float64(p.Y) > py) {
				x := float64(p.X) + (py-float64(p.Y))*float64(q.X-p.X)/float64(q.Y-p.Y)
				xs = append(xs, (float64(offset)+x)*scale)
			}
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			start := max(0, int(math.Ceil(xs[i]-.5)))
			end := min(m.Rect.Dx(), int(math.Ceil(xs[i+1]-.5)))
			for x := start; x < end; x++ {
				m.Pix[y*m.Stride+x] = ink
			}
		}
	}
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
