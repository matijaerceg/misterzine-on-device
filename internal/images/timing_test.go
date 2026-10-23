package images

import (
	"io"
	"log"
	"testing"
	"time"
)

func TestVideoTimingIsOptInAndSeparatesStages(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 1024)
	defer s.Close()
	s.recordVideoTiming(time.Millisecond, time.Millisecond, time.Millisecond)
	if s.VideoStats()["decoded_frames"] != uint64(0) {
		t.Fatal("normal mode collected timing")
	}
	s.SetVideoTiming(true)
	s.recordVideoTiming(2*time.Millisecond, 3*time.Millisecond, 4*time.Millisecond)
	s.recordVideoTiming(4*time.Millisecond, 5*time.Millisecond, 6*time.Millisecond)
	stats := s.VideoStats()
	if stats["decoded_frames"] != uint64(2) || stats["jpeg"].(map[string]float64)["avg_ms"] != 3 || stats["resize"].(map[string]float64)["max_ms"] != 6 {
		t.Fatalf("wrong stages: %v", stats)
	}
	s.SetVideoTiming(false)
	if s.VideoStats()["enabled"] != false || s.VideoStats()["decoded_frames"] != uint64(0) {
		t.Fatal("disabling retained measurements")
	}
}
