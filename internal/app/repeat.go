package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// repeater turns a held key into repeated actions with acceleration. Main's
// virtual keyboard autorepeats too, but those events are dropped by the
// platform layer so the feel is ours: a short first delay, then faster and
// faster, and an immediate stop on release.
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

// due returns the key to repeat if one is due at now, else KeyNone.
func (r *repeater) due(now time.Time) platform.Key {
	if !r.held || now.Before(r.next) {
		return platform.KeyNone
	}
	r.count++
	var step time.Duration
	switch r.key {
	case platform.KeyPageUp, platform.KeyPageDown:
		step = repeatPage
	case platform.KeyLeft, platform.KeyRight:
		step = repeatSlow
	default:
		switch {
		case r.count > 30:
			step = repeatFast
		case r.count > 10:
			step = repeatMid
		default:
			step = repeatSlow
		}
	}
	r.next = now.Add(step)
	return r.key
}

// nextAt reports when the loop should wake for a repeat; zero when idle.
func (r *repeater) nextAt() time.Time {
	if !r.held {
		return time.Time{}
	}
	return r.next
}
