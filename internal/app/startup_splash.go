package app

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	"image/png"
	"time"
)

//go:embed startup_logo.png
var startupLogoPNG []byte

const startupFade = time.Second
const startupFrame = time.Second / 30

type startupSplash struct {
	enabled, started bool
	at, next         time.Time
	coverage         uint8
	logo             *image.RGBA
	mask             *image.Alpha
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
		s.mask = nil
	} else {
		s.coverage = uint8(255 * (startupFade - elapsed) / startupFade)
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
		s.coverage = 255
	}
	r := a.lay.Body
	if s.logo == nil {
		src, err := png.Decode(bytes.NewReader(startupLogoPNG))
		if err != nil {
			s.enabled = false
			s.next = time.Time{}
			return
		}
		s.logo = image.NewRGBA(src.Bounds())
		s.mask = image.NewAlpha(src.Bounds())
		draw.Draw(s.logo, s.logo.Rect, src, src.Bounds().Min, draw.Src)
	}
	w, h := s.logo.Rect.Dx(), s.logo.Rect.Dy()
	// Keep one asset pixel per logical screen pixel in every orientation.
	// An unusually small safe area omits the effect instead of scaling it.
	if w > r.Dx() || h > r.Dy() {
		return
	}
	rect := image.Rect(0, 0, w, h).Add(image.Pt(r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2))
	// A fixed ordered pattern removes whole pixels as time passes. Surviving
	// pixels retain the source colour and its original edge antialiasing.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			value := uint8(0)
			if int(s.coverage) > int(saverBayer[y&7][x&7]) {
				value = 255
			}
			s.mask.Pix[y*s.mask.Stride+x] = value
		}
	}
	draw.DrawMask(a.logical.RGBA, rect, s.logo, image.Point{}, s.mask, image.Point{}, draw.Over)
	a.logical.DirtyAll()
}
