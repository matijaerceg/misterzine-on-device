package app

import "time"

// Counts frame availability once per requested frame, not once per repaint.
// The state is reset with the video when its row, geometry or screen changes.
type videoTiming struct {
	seen, delivered                         bool
	frame                                   int
	requested, available, immediate, waited uint64
	pending                                 time.Time
	waitTotal, waitMax                      time.Duration
}

func (v *videoTiming) observe(frame int, ready, loading bool, now time.Time) {
	if !v.seen || v.frame != frame {
		v.seen, v.delivered, v.frame = true, false, frame
		v.pending = time.Time{}
		v.requested++
		if ready {
			v.immediate++
		}
	}
	if v.delivered {
		return
	}
	if !ready {
		if loading && v.pending.IsZero() {
			v.pending = now
		}
		return
	}
	v.delivered = true
	v.available++
	if !v.pending.IsZero() {
		d := now.Sub(v.pending)
		v.waited++
		v.waitTotal += d
		v.waitMax = max(v.waitMax, d)
	}
}

func (v *videoTiming) snapshot() map[string]any {
	avg := 0.0
	if v.waited != 0 {
		avg = float64(v.waitTotal) / float64(time.Millisecond) / float64(v.waited)
	}
	return map[string]any{
		"requested_frames": v.requested, "available_frames": v.available,
		"immediate_frames": v.immediate, "waited_frames": v.waited,
		"wait_avg_ms": avg, "wait_max_ms": float64(v.waitMax) / float64(time.Millisecond),
	}
}
