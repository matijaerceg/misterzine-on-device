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
	expandOptionsForTest(a)
	for i, e := range a.panel.entries {
		if e.kind == "scroll" {
			a.panel.cursor = i
		}
	}
	a.Tick(now)
	if !a.OptionSamplesRunning() {
		t.Fatal("no frame loop requested")
	}
	for frame := 1; frame <= 12; frame++ {
		// Irregular timer calls, including a long stall, cannot advance the preview.
		a.Tick(now.Add(time.Duration(frame) * time.Second))
		if a.optionSamples.elapsed != time.Duration(frame-1)*frameDur {
			t.Fatal("timer moved selection")
		}
		a.OptionSampleFrame()
		for i, speed := range ScrollValues {
			got := sampleListSelection(a.optionSamples.elapsed, scrollPace(speed), 0, 3)
			every := []int{3, 2, 1}[i]
			if got != (frame/every)%3 {
				t.Fatalf("frame %d speed %s selection %d", frame, speed, got)
			}
		}
	}
	a.screen = ScreenList
	if a.OptionSamplesRunning() || a.OptionSampleFrame() {
		t.Fatal("animation continues outside Options")
	}
	if a.optionSamples.kind != "" {
		t.Fatal("clock not cleared")
	}
}
