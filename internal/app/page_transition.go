package app

import (
	"image"
	"math"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

const pageWipeDuration = 200 * time.Millisecond

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
}

// EnablePageTransitions enables navigation animation for an interactive host.
// Static renderers can keep drawing the final page without an animation clock.
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
	if !t.enabled {
		return
	}
	p, bounds := a.pageIdentity(), a.logical.Rect
	if a.saver.active || !t.painted || t.bounds != bounds {
		t.from, t.to, t.next = nil, nil, time.Time{}
		t.painted = !a.saver.active
	} else if p != t.page {
		t.from = append(t.from[:0], a.logical.Pix...)
		t.at = a.cfg.TimerNow()
		t.next = t.at.Add(frameDur)
		t.progress = 0
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
		return false
	}
	t.progress = max(0, float64(now.Sub(t.at))/float64(pageWipeDuration))
	if t.progress >= 1 || a.saver.active {
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

func (a *App) paintPageTransition() {
	t := &a.transition
	if len(t.from) != len(a.logical.Pix) {
		return
	}
	composePageWipe(a.logical.Pix, t.from, a.logical.W(), a.logical.H(), a.logical.Stride, t.progress, t.back)
	a.logical.DirtyAll()
}

// The incoming page is already in dst. Only a narrow boundary dissolves; either
// side stays sharp. The edge uses the screensaver's fine pixel dissolve.
// Forward reveals from the right, Back from the left.
func composePageWipe(dst, from []byte, w, h, stride int, progress float64, back bool) {
	band := max(2.0, float64(w)/saverShotBandDiv)
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
		for x := lo; x < hi; x++ {
			amount := (float64(x)-center)/band + 0.5
			if back {
				amount = 1 - amount
			}
			alpha := int(max(0, min(1, amount)) * 256)
			i := row + x*4
			if alpha <= int(saverBayer[y&7][x&7]) {
				copy(dst[i:i+4], from[i:i+4])
			}
		}
	}
}
