package images

import "time"

// Debug-only accumulators. Locking shares the service mutex; decoding and
// resizing themselves never run under it. Stats cover successful cache fills,
// including decode-ahead, not cache reads or UI frame presentation.
type videoTiming struct {
	enabled                bool
	frames                 uint64
	decode, convert, scale stageTiming
}

type stageTiming struct {
	total, maximum time.Duration
}

func (s *stageTiming) add(d time.Duration) {
	s.total += d
	s.maximum = max(s.maximum, d)
}

func (s stageTiming) snapshot(n uint64) map[string]float64 {
	avg := 0.0
	if n != 0 {
		avg = float64(s.total) / float64(time.Millisecond) / float64(n)
	}
	return map[string]float64{"avg_ms": avg, "max_ms": float64(s.maximum) / float64(time.Millisecond)}
}

func (s *Service) SetVideoTiming(enabled bool) {
	s.mu.Lock()
	s.videoTiming = videoTiming{enabled: enabled}
	s.mu.Unlock()
}

func (s *Service) videoTimingEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.videoTiming.enabled
}

func (s *Service) recordVideoTiming(decode, convert, scale time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := &s.videoTiming
	if !v.enabled {
		return
	}
	v.frames++
	v.decode.add(decode)
	v.convert.add(convert)
	v.scale.add(scale)
}

func (s *Service) VideoStats() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := &s.videoTiming
	return map[string]any{
		"enabled": v.enabled, "decoded_frames": v.frames,
		"jpeg": v.decode.snapshot(v.frames), "convert": v.convert.snapshot(v.frames),
		"resize": v.scale.snapshot(v.frames), "scaled_cache_bytes": s.bytes,
	}
}
