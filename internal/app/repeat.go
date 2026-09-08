package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// repeater turns a held key into repeated actions. Which keys repeat, and
// how fast, depends on the screen (see App.repeatStep): list scrolling
// accelerates, screenshot paging is slow and flat, and action keys never
// repeat, so a held Enter cannot open details and then launch. Main's
// virtual keyboard autorepeats too, but those events are dropped by the
// platform layer so the feel is ours.
type repeater struct {
	key   platform.Key
	held  bool
	next  time.Time
	count int
}

const (
	repeatDelay = 280 * time.Millisecond
	repeatSlow  = 70 * time.Millisecond
	repeatMid   = 40 * time.Millisecond
	repeatFast  = 24 * time.Millisecond
	repeatPage  = 150 * time.Millisecond
	repeatStep  = 200 * time.Millisecond // flat pace for row/slot walking
	repeatCalib = 60 * time.Millisecond
)

func (r *repeater) press(k platform.Key, now time.Time) {
	r.key, r.held, r.count = k, true, 0
	r.next = now.Add(repeatDelay)
}

func (r *repeater) release(k platform.Key) {
	if r.key == k {
		r.held = false
	}
}

// due returns the held key when a repeat is due; step decides the pace for
// the next one (0 = this key does not repeat here, so stop).
func (r *repeater) due(now time.Time, step func(platform.Key, int) time.Duration) platform.Key {
	if !r.held || now.Before(r.next) {
		return platform.KeyNone
	}
	r.count++
	d := step(r.key, r.count)
	if d == 0 {
		r.held = false
		return platform.KeyNone
	}
	r.next = now.Add(d)
	return r.key
}

// nextAt reports when the loop should wake for a repeat; zero when idle.
func (r *repeater) nextAt() time.Time {
	if !r.held {
		return time.Time{}
	}
	return r.next
}

// accel is the list scrolling ladder: slow, then faster after 10 and 30 steps.
func accel(count int) time.Duration {
	switch {
	case count > 30:
		return repeatFast
	case count > 10:
		return repeatMid
	}
	return repeatSlow
}
