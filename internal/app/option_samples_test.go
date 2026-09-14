package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"testing"
	"time"
)

func TestSampleListTiming(t *testing.T) {
	for _, speed := range ScrollValues {
		pace := scrollPace(speed)
		for _, ms := range []int{200, 300, 500} {
			delay := time.Duration(ms) * time.Millisecond
			for _, c := range []struct {
				at   time.Duration
				want int
			}{
				{0, 0}, {delay - time.Nanosecond, 0}, {delay + pace, 1},
				{delay + 12*pace, 12}, {2*delay + 12*pace - time.Nanosecond, 12},
				{2*delay + 13*pace, 11}, {2 * (delay + 12*pace), 0},
			} {
				if got := sampleListOffset(c.at, pace, delay); got != c.want {
					t.Fatalf("%s/%d at %v: %d want %d", speed, ms, c.at, got, c.want)
				}
			}
		}
		if got := sampleListOffset(6*pace, pace, 0); got != 6 {
			t.Fatal(got)
		}
	}
}
func TestOptionSampleScheduling(t *testing.T) {
	now := time.Unix(100, 0)
	a := New(Config{PhysW: 320, PhysH: 240, TimerNow: func() time.Time { return now }}, data.Ingest(nil, "", now), nil)
	a.screen = ScreenOptions
	a.buildPanel()
	for i, e := range a.panel.entries {
		if e.kind == "scroll" {
			a.panel.cursor = i
		}
	}
	if a.nextOptionSampleTick().IsZero() {
		t.Fatal("animation not scheduled")
	}
	if !a.Tick(now) || !a.nextOptionSampleTick().After(now) {
		t.Fatal("animation did not advance")
	}
	before := a.optionSamples.now
	if a.Tick(now.Add(time.Millisecond)) || a.optionSamples.now != before {
		t.Fatal("early animation frame")
	}
	a.screen = ScreenList
	if !a.nextOptionSampleTick().IsZero() {
		t.Fatal("animation continues off Options")
	}
	a.tickOptionSamples(now)
	if a.optionSamples.kind != "" {
		t.Fatal("clock not cleared")
	}
}
