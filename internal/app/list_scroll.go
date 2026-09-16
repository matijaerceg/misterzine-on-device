package app

import "time"

// offset is the distance still to travel to the already selected destination.
// A repeat is divided into display frames; release finishes only that repeat.
type listMotion struct {
	offset, frames int
	dirty          bool
}

func (a *App) ListScrollRunning() bool {
	return a.screen == ScreenList && !a.saver.active && a.listMotion.offset != 0
}

func (a *App) ListScrollFrame(_ time.Time) bool {
	if !a.ListScrollRunning() {
		return false
	}
	m := &a.listMotion
	m.offset -= m.offset / max(1, m.frames)
	m.frames--
	m.dirty = true
	return true
}

// SmoothScrolling is retained when the 60 Hz choice hides its control.
func (a *App) SmoothScrolling() bool { return !a.cfg.SmoothScrollDisabled }
