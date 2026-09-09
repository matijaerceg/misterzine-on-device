//go:build linux

package main

import "time"

// The local sample retains Go's monotonic clock. Advancing the server date by
// elapsed time avoids adding the old boot-time offset again when NTP jumps.
type serverClockSample struct {
	wall, local time.Time
}

func (s *serverClockSample) at(system time.Time, elapsed time.Duration) (time.Time, bool) {
	estimated := s.wall.Add(elapsed)
	delta := system.Sub(estimated)
	if delta >= -5*time.Second && delta <= 5*time.Second {
		return system, true
	}
	return estimated, false
}

func (h *host) now() time.Time {
	system := time.Now()
	sample := h.timeSample.Load()
	if sample == nil {
		return system
	}
	corrected, caughtUp := sample.at(system, system.Sub(sample.local))
	if caughtUp {
		h.timeSample.CompareAndSwap(sample, nil)
	}
	return corrected
}

// UI-owned: record this visit when a reliable date first becomes available.
func (h *host) trustClock() {
	if h.clock.Trusted {
		return
	}
	h.clock.Trusted = true
	h.a.SetClockTrusted(true)
	h.dirty = true
}
