package app

import (
	"image"
	"math"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

const pageWipeDuration = 200 * time.Millisecond
const pageWipeBandDiv = 5

type pageIdentity struct {
	screen  Screen
	view    data.SortMode
	support string
}

type pageTransition struct {
	enabled, painted bool
	page             pageIdentity
	bounds           image.Rectangle
	from, to         []byte
	at, next         time.Time
	progress         float64
	back             bool
	lastProgress     float64
	fresh, pending   bool
}

// PageTransitions reports the saved navigation animation preference.
func (a *App) PageTransitions() bool { return !a.cfg.TransitionsDisabled }

// EnablePageTransitions enables the animation clock for an interactive host.
// Static renderers keep drawing the final page without a clock.
func (a *App) EnablePageTransitions() { a.transition.enabled = true }

func (a *App) pageIdentity() pageIdentity {
	p := pageIdentity{screen: a.screen}
	if a.screen == ScreenList {
		p.view = a.mode
	}
	if a.screen == ScreenTroubleshooting {
		p.support = a.support.mode
	}
	return p
}

func (p pageIdentity) depth() int {
	switch p.screen {
	case ScreenList:
		return 0
	case ScreenDetails, ScreenFilter:
		return 1
	case ScreenShot:
		return 2
	case ScreenOptions:
		return 3
	case ScreenTroubleshooting:
		if p.support != "menu" && p.support != "" {
			return 5
		}
	}
	return 4
}

// Snapshot before painting the destination. During interrupted navigation this
// is the actual composed frame on screen, so no intermediate page flashes up.
func (a *App) preparePageTransition() {
	t := &a.transition
	if !t.enabled || !a.PageTransitions() {
		if len(t.from) > 0 {
			a.cfg.Images.SetPaused(false)
		}
		t.from, t.to, t.next, t.painted = nil, nil, time.Time{}, false
		return
	}
	p, bounds := a.pageIdentity(), a.logical.Rect
	if a.saver.active || !t.painted || t.bounds != bounds {
		if len(t.from) > 0 {
			a.cfg.Images.SetPaused(false)
		}
		t.from, t.to, t.next = nil, nil, time.Time{}
		t.painted = !a.saver.active
	} else if p != t.page {
		a.cfg.Images.SetPaused(true)
		t.pending = false
		t.from = append(t.from[:0], a.logical.Pix...)
		t.at = a.cfg.TimerNow()
		t.next = t.at.Add(frameDur)
		t.progress, t.lastProgress, t.fresh = 0, 0, true
		t.back = p.depth() < t.page.depth()
		// The splash belongs to startup, not to the arriving page.
		a.splash.enabled, a.splash.next = false, time.Time{}
	}
	t.page, t.bounds = p, bounds
}

func (a *App) PageTransitionRunning() bool { return len(a.transition.from) != 0 }

func (a *App) tickPageTransition(now time.Time) bool {
	t := &a.transition
	if !a.PageTransitionRunning() {
		if !t.next.IsZero() {
			t.next = time.Time{}
			a.Invalidate()
			return true
		}
		return false
	}
	t.progress = min(1, max(0, float64(now.Sub(t.at))/float64(pageWipeDuration)))
	if a.saver.active {
		a.cfg.Images.SetPaused(false)
		a.all = true
		t.from, t.to, t.next = nil, nil, time.Time{}
	} else {
		t.next = now.Add(frameDur)
		if end := t.at.Add(pageWipeDuration); t.next.After(end) {
			t.next = end
		}
	}
	return true
}

func (a *App) paintPageTransition(full bool) {
	t := &a.transition
	if len(t.from) != len(a.logical.Pix) {
		return
	}
	c := a.logical
	composePageWipe(c.Pix, t.from, c.W(), c.H(), c.Stride, t.progress, t.back)
	c.TakeDirty()
	if t.fresh {
		// Drawing the destination precedes the animation clock. The old
		// physical frame is already visible, so don't copy it again.
		t.fresh = false
		t.at = a.cfg.TimerNow()
		t.next = t.at.Add(frameDur)
	} else if full {
		c.DirtyAll() // explicit input changed the page during the wipe
	} else {
		c.Dirty(pageWipeDirty(c.W(), c.H(), t.lastProgress, t.progress, t.back))
	}
	t.lastProgress = t.progress
	if t.progress >= 1 {
		a.cfg.Images.SetPaused(a.RepeatActive())
		t.from, t.to, t.next = nil, nil, time.Time{}
		if t.pending {
			t.pending = false
			t.next = a.cfg.TimerNow().Add(frameDur)
		}
	}
}

// Only the moving band changes between cached frames. Bound its two positions
// across every row; rotated canvases then copy just this strip to the display.
func pageWipeDirty(w, h int, previous, current float64, back bool) image.Rectangle {
	if current == previous {
		return image.Rectangle{}
	}
	band := max(2.0, float64(w)/pageWipeBandDiv)
	amp := float64(w) / 32 * 1.35
	margin := float64(w)/32*1.4 + band
	a := -margin + previous*(float64(w)+2*margin)
	b := -margin + current*(float64(w)+2*margin)
	lo := int(math.Floor(min(a, b)-amp-band/2)) - 1
	hi := int(math.Ceil(max(a, b)+amp+band/2)) + 2
	if !back {
		lo, hi = w-hi, w-lo
	}
	return image.Rect(lo, 0, hi, h).Intersect(image.Rect(0, 0, w, h))
}

// The incoming page is already in dst. A broad boundary dissolves; either
// side stays sharp. The edge uses the screensaver's fine pixel dissolve.
// Forward reveals from the right, Back from the left.
func composePageWipe(dst, from []byte, w, h, stride int, progress float64, back bool) {
	band := max(2.0, float64(w)/pageWipeBandDiv)
	amp := float64(w) / 32
	margin := amp*1.4 + band
	travel := -margin + progress*(float64(w)+2*margin)
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h)
		edge := travel + amp*(math.Sin(2*math.Pi*v+0.5)+0.35*math.Sin(4*math.Pi*v+1.2))
		center := float64(w-1) - edge
		if back {
			center = edge
		}
		lo := max(0, min(w, int(math.Floor(center-band/2))))
		hi := max(0, min(w, int(math.Ceil(center+band/2))))
		row := y * stride
		// Copy the solid outgoing span in one operation. Only the few pixels
		// along the soft edge need arithmetic on these ARM boards.
		if back {
			copy(dst[row+hi*4:row+w*4], from[row+hi*4:row+w*4])
		} else {
			copy(dst[row:row+lo*4], from[row:row+lo*4])
		}
		centerFixed := int(center * 256)
		bandPixels := max(2, w/pageWipeBandDiv)
		for x := lo; x < hi; x++ {
			distance := x*256 - centerFixed
			if back {
				distance = -distance
			}
			if distance <= (int(saverBayer[y&7][x&7])-128)*bandPixels {
				i := row + x*4
				copy(dst[i:i+4], from[i:i+4])
			}
		}
	}
}
