package app

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"time"
)

//go:embed startup_logo.png
var startupLogoPNG []byte

const startupFade = 2 * time.Second
const startupFrame = time.Second / 30

type startupSplash struct {
	enabled, started bool
	at, next         time.Time
	opacity          uint8
	logo             *image.RGBA
}

// StartSplash enables the once-per-launch branding on the first list frame.
// It is only a paint effect; input continues through the normal event loop.
func (a *App) StartSplash() {
	if a.splash.enabled || a.splash.started {
		return
	}
	a.splash.enabled = true
	a.all = true
}

func (a *App) tickSplash(now time.Time) bool {
	s := &a.splash
	if !s.enabled || !s.started {
		return false
	}
	elapsed := max(time.Duration(0), now.Sub(s.at))
	if elapsed >= startupFade || a.screen != ScreenList {
		s.enabled = false
		s.next = time.Time{}
		s.logo = nil
	} else {
		s.opacity = uint8(255 * (startupFade - elapsed) / startupFade)
		s.next = now.Add(startupFrame)
		if end := s.at.Add(startupFade); s.next.After(end) {
			s.next = end
		}
	}
	a.all = true
	return true
}

func (a *App) paintSplash() {
	s := &a.splash
	if !s.enabled || a.screen != ScreenList || a.saver.active {
		return
	}
	if !s.started {
		s.started = true
		s.at = a.cfg.TimerNow()
		s.next = s.at.Add(startupFrame)
		s.opacity = 255
	}
	r := a.lay.Body
	w := r.Dx() * 3 / 4
	h := w * 354 / 532
	if h > r.Dy()*3/4 {
		h = r.Dy() * 3 / 4
		w = h * 532 / 354
	}
	if s.logo == nil || s.logo.Rect.Dx() != w || s.logo.Rect.Dy() != h {
		src, err := png.Decode(bytes.NewReader(startupLogoPNG))
		if err != nil {
			s.enabled = false
			s.next = time.Time{}
			return
		}
		s.logo = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				s.logo.Set(x, y, src.At(x*src.Bounds().Dx()/w, y*src.Bounds().Dy()/h))
			}
		}
	}
	rect := image.Rect(0, 0, w, h).Add(image.Pt(r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2))
	draw.DrawMask(a.logical.RGBA, rect, s.logo, image.Point{}, image.NewUniform(color.Alpha{A: s.opacity}), image.Point{}, draw.Over)
	a.logical.DirtyAll()
}
