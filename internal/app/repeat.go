package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// repeater turns a held key into repeated actions. Which keys repeat, and
// how fast, depends on the screen (see App.repeatStep): list scrolling
// uses the chosen pace, screenshot paging is slower, and action keys never
// repeat, so a held Enter cannot open details and then launch. Main's
// virtual keyboard autorepeats too, but those events are dropped by the
// platform layer so the feel is ours.
type repeater struct {
	key    platform.Key
	held   bool
	next   time.Time
	count  int
	frames int // frames since the last step (frame-driven mode)
	every  int // frames per step (frame-driven mode)
}

// frameDur is one frame of the 60 Hz framebuffer.
const frameDur = 16667 * time.Microsecond

const (
	repeatDelay = 500 * time.Millisecond // a tap held a little long is still one step
	repeatPage  = 150 * time.Millisecond
	repeatStep  = 200 * time.Millisecond // flat pace for row/slot walking
	repeatErase = 60 * time.Millisecond  // a held Backspace clears a search quickly
	repeatCalib = 60 * time.Millisecond
)

func (r *repeater) press(k platform.Key, now time.Time) {
	r.key, r.held, r.count = k, true, 0
	r.frames, r.every = 0, 0
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
	// anchored to the schedule, but never more than one step per call: a
	// late frame slides the schedule rather than skipping rows
	r.next = r.next.Add(d)
	if r.next.Before(now) {
		r.next = now
	}
	return r.key
}

// frameDue is the vsync-driven counterpart of due, called once per frame:
// a step happens every N frames, N from the pace the screen asks for, so
// the rate is exact and no frame ever moves more than one step.
func (r *repeater) frameDue(now time.Time, step func(platform.Key, int) time.Duration) platform.Key {
	if !r.held || now.Before(r.next) {
		return platform.KeyNone
	}
	r.frames++
	if r.frames < r.every {
		return platform.KeyNone
	}
	r.frames = 0
	r.count++
	d := step(r.key, r.count)
	if d == 0 {
		r.held = false
		return platform.KeyNone
	}
	r.every = int((d + frameDur/2) / frameDur)
	if r.every < 1 {
		r.every = 1
	}
	r.next = now
	return r.key
}

// nextAt reports when the loop should wake for a repeat; zero when idle.
func (r *repeater) nextAt() time.Time {
	if !r.held {
		return time.Time{}
	}
	return r.next
}

// Scroll speeds for a held Up/Down in lists, keyed by rows per second:
// every third frame, every second frame, every frame.
var scrollSpeeds = map[string]time.Duration{"20": 3 * frameDur, "30": 2 * frameDur, "60": frameDur}

// ScrollValues are the setting's choices in order.
var ScrollValues = []string{"20", "30", "60"}

// scrollPace applies the chosen speed from the first repeat.
func scrollPace(speed string) time.Duration {
	d, ok := scrollSpeeds[speed]
	if !ok {
		d = scrollSpeeds["30"]
	}
	return d
}
