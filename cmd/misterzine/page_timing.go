//go:build linux

package main

import "time"

// Debug-only, in-memory timing avoids logging to the card between frames.
type pageFrameTiming struct {
	Screen     string `json:"screen"`
	PaintUS    int64  `json:"paint_us"`
	WaitUS     int64  `json:"wait_us"`
	CopyUS     int64  `json:"copy_us"`
	IntervalUS int64  `json:"interval_us"`
}

func (h *host) recordPageFrame(active bool, paint, wait, copyTime time.Duration, at time.Time) {
	if !active {
		h.pageLastAt = time.Time{}
		return
	}
	screen := h.a.Screen().String()
	interval := time.Duration(0)
	if screen == h.pageLastScreen && !h.pageLastAt.IsZero() {
		interval = at.Sub(h.pageLastAt)
	}
	h.pageLastAt, h.pageLastScreen = at, screen
	if len(h.pageFrames) == 128 {
		copy(h.pageFrames, h.pageFrames[1:])
		h.pageFrames = h.pageFrames[:127]
	}
	h.pageFrames = append(h.pageFrames, pageFrameTiming{screen, paint.Microseconds(), wait.Microseconds(), copyTime.Microseconds(), interval.Microseconds()})
}

// layoutFrameTiming is one presented frame of a Select+Y layout motion: what
// it cost, the interval since the motion's previous frame, and how far the
// motion had got (1 is the settled layout). Steps of 1/12 at 16.7 ms each
// are an even motion; a 33 ms interval is a missed blank, and a bigger step
// than the one before is part of the motion skipped.
type layoutFrameTiming struct {
	Progress   float64 `json:"progress"`
	PaintUS    int64   `json:"paint_us"`
	WaitUS     int64   `json:"wait_us"`
	CopyUS     int64   `json:"copy_us"`
	IntervalUS int64   `json:"interval_us"`
}

func (h *host) recordLayoutFrame(active bool, paint, wait, copyTime time.Duration, at time.Time) {
	if !active {
		h.layoutLastAt = time.Time{}
		return
	}
	interval := time.Duration(0)
	if !h.layoutLastAt.IsZero() {
		interval = at.Sub(h.layoutLastAt)
	}
	h.layoutLastAt = at
	if !h.a.LayoutTransitionRunning() {
		h.layoutLastAt = time.Time{} // settled: the next press starts afresh
	}
	if len(h.layoutFrames) == 128 {
		copy(h.layoutFrames, h.layoutFrames[1:])
		h.layoutFrames = h.layoutFrames[:127]
	}
	h.layoutFrames = append(h.layoutFrames, layoutFrameTiming{h.a.LayoutMotionProgress(), paint.Microseconds(), wait.Microseconds(), copyTime.Microseconds(), interval.Microseconds()})
}

// Keep scrolling measurements separate from page changes and idle gaps.
func (h *host) recordDetailFrame(active bool, paint, wait, copyTime time.Duration, at time.Time) {
	if !active {
		h.detailLastAt = time.Time{}
		return
	}
	interval := time.Duration(0)
	if !h.detailLastAt.IsZero() {
		interval = at.Sub(h.detailLastAt)
	}
	h.detailLastAt = at
	if len(h.detailFrames) == 1024 {
		copy(h.detailFrames, h.detailFrames[1:])
		h.detailFrames = h.detailFrames[:1023]
	}
	h.detailFrames = append(h.detailFrames, pageFrameTiming{h.a.Screen().String(), paint.Microseconds(), wait.Microseconds(), copyTime.Microseconds(), interval.Microseconds()})
	if !h.a.DetailScrollRunning() {
		h.detailLastAt = time.Time{}
	}
}
