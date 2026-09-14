package app

import (
	"image"
	"math"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

const pageWipeDuration = 150 * time.Millisecond

type pageIdentity struct {
	screen  Screen
	view    data.SortMode
	support string
}

type pageTransition struct {
	enabled, painted bool
	page             pageIdentity
	bounds           image.Rectangle
	from             []byte
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
		t.from, t.next = nil, time.Time{}
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
		t.from, t.next = nil, time.Time{}
	} else {
		t.next = now.Add(frameDur)
		if end := t.at.Add(pageWipeDuration); t.next.After(end) {
			t.next = end
		}
	}
	a.all = true
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

// The incoming page is already in dst. Only a narrow boundary blends; either
// side stays sharp. Forward reveals from the right, Back from the left.
func composePageWipe(dst, from []byte, w, h, stride int, progress float64, back bool) {
	band := max(2.0, float64(w)/50)
	amp := float64(w) / 32
	margin := amp*1.4 + band
	travel := -margin + progress*(float64(w)+2*margin)
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h)
		edge := travel + amp*(math.Sin(2*math.Pi*v+0.5)+0.35*math.Sin(4*math.Pi*v+1.2))
		for x := 0; x < w; x++ {
			distance := float64(w - 1 - x)
			if back {
				distance = float64(x)
			}
			alpha := int(max(0, min(1, (edge-distance)/band+0.5)) * 256)
			i := y*stride + x*4
			if alpha == 256 {
				continue
			}
			if alpha == 0 {
				copy(dst[i:i+4], from[i:i+4])
				continue
			}
			for c := 0; c < 3; c++ {
				dst[i+c] = byte((int(dst[i+c])*alpha + int(from[i+c])*(256-alpha) + 128) >> 8)
			}
			dst[i+3] = 255
		}
	}
}
