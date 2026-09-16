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

func TestSmoothScrollOption(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(nil, "", time.Now()), nil)
	a.screen = ScreenOptions
	for _, speed := range []string{"20", "30", "60", "30"} {
		a.cfg.Scroll = speed
		a.buildPanel()
		expandOptionsForTest(a)
		found := false
		for i, e := range a.panel.entries {
			if e.kind != "smooth-scroll" {
				continue
			}
			found = true
			if !e.child || i == 0 || a.panel.entries[i-1].kind != "scroll" {
				t.Fatal("not a child of speed")
			}
			a.panel.cursor = i
			a.stepValue(-1)
			if a.SmoothScrolling() {
				t.Fatal("toggle failed")
			}
			if !a.OptionSamplesRunning() {
				t.Fatal("samples not animated")
			}
		}
		if found != (speed != "60") {
			t.Fatal("wrong visibility", speed)
		}
	}
	if a.SmoothScrolling() {
		t.Fatal("hidden setting lost")
	}
}

func TestSmoothSampleFrames(t *testing.T) {
	for _, speed := range []string{"20", "30"} {
		pace := scrollPace(speed)
		every := int(pace / frameDur)
		for frame := 0; frame < 12; frame++ {
			at := time.Duration(frame) * frameDur
			if got := smoothSampleTravel(at, pace, true, 12); got != frame*12/every {
				t.Fatal("uneven smooth travel")
			}
			if got := smoothSampleTravel(at, pace, false, 12); got != frame/every*12 {
				t.Fatal("off sample interpolated")
			}
		}
	}
}
