package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"testing"
	"time"
)

func TestSampleListTiming(t *testing.T) {
	for _, rows := range []int{3, 4} {
		for _, speed := range ScrollValues {
			pace := scrollPace(speed)
			for step := 0; step < rows*3; step++ {
				if got := sampleListSelection(time.Duration(step)*pace, pace, 0, rows); got != step%rows {
					t.Fatal("selection does not wrap", got)
				}
			}
			for _, ms := range []int{200, 300, 500} {
				delay := time.Duration(ms) * time.Millisecond
				end := delay + time.Duration(rows-2)*pace
				for _, c := range []struct {
					at   time.Duration
					want int
				}{
					{0, 0}, {delay - time.Nanosecond, 0}, {delay, 1}, {end, rows - 1},
					{end + delay - time.Nanosecond, rows - 1}, {end + delay, rows - 2}, {2 * end, 0},
				} {
					if got := sampleListSelection(c.at, pace, delay, rows); got != c.want {
						t.Fatalf("rows %d speed %s delay %d at %v: got %d want %d", rows, speed, ms, c.at, got, c.want)
					}
				}
			}
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
