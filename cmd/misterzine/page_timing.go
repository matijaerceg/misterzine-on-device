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
