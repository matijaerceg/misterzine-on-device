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
	repeatDelay = 200 * time.Millisecond
	repeatSlow  = 48 * time.Millisecond
	repeatMid   = 28 * time.Millisecond
	repeatFast  = 16 * time.Millisecond
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
	// anchor to the schedule, not to when the loop got round to it, so the
	// average rate is exact; after a long stall (a data swap) restart from now
	if now.Sub(r.next) > 4*d {
		r.next = now.Add(d)
	} else {
		r.next = r.next.Add(d)
	}
	return r.key
}

// nextAt reports when the loop should wake for a repeat; zero when idle.
func (r *repeater) nextAt() time.Time {
	if !r.held {
		return time.Time{}
	}
	return r.next
}

// Scroll speeds for a held Up/Down in lists: a steady rate with a short
// warm-up. "fast" is about 30 rows a second, which is what a CRT list wants.
var scrollSpeeds = map[string]time.Duration{"normal": 50 * time.Millisecond, "fast": 33 * time.Millisecond, "turbo": 20 * time.Millisecond}

// accel is the list scrolling ladder for the chosen speed: two slower steps
// so a single tap never overshoots, then the steady rate.
func accel(speed string, count int) time.Duration {
	d, ok := scrollSpeeds[speed]
	if !ok {
		d = scrollSpeeds["fast"]
	}
	if count <= 2 {
		return d * 2
	}
	return d
}
