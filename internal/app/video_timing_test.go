package app

import (
	"testing"
	"time"
)

func TestVideoTimingCountsFramesRatherThanRepaints(t *testing.T) {
	var v videoTiming
	now := time.Unix(100, 0)
	v.observe(0, false, true, now)
	v.observe(0, false, true, now.Add(10*time.Millisecond))
	v.observe(0, true, false, now.Add(40*time.Millisecond))
	v.observe(0, true, false, now.Add(50*time.Millisecond))
	v.observe(1, true, false, now.Add(67*time.Millisecond))
	v.observe(1, true, false, now.Add(70*time.Millisecond))
	if v.requested != 2 || v.available != 2 || v.immediate != 1 || v.waited != 1 || v.waitTotal != 40*time.Millisecond {
		t.Fatalf("timing counted repeated paints or measured the wrong delay: %+v", v)
	}
	if got := v.snapshot()["wait_avg_ms"]; got != 40.0 {
		t.Fatal(got)
	}
}
