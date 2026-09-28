package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// Scroll speed offers 10, 20, 30 and 60 rows a second; 10 moves a row every
// sixth frame.

func TestScrollValues(t *testing.T) {
	if want := []string{"10", "20", "30", "60"}; !reflect.DeepEqual(ScrollValues, want) {
		t.Fatalf("offered %v, want %v", ScrollValues, want)
	}
	for _, v := range ScrollValues {
		if scrollSpeeds[v] <= 0 {
			t.Fatalf("offered speed %s has no pace", v)
		}
	}
}

// Every saved value, garbage included, reads as an offered speed with a
// real pace.
func TestSavedScrollReadsAsOfferedSpeed(t *testing.T) {
	frames := map[string]int{"10": 6, "20": 3, "30": 2, "60": 1}
	for saved, want := range map[string]string{
		"10": "10", "20": "20", "30": "30", "60": "60",
		"": "30", "fast": "30", "warp": "30", "0": "30", "5": "30", "15": "30", "-10": "30", "10 Hz": "30", " 10": "30",
	} {
		if got := offeredScroll(saved); got != want {
			t.Errorf("saved %q reads as %q, want %q", saved, got, want)
		}
		if got := scrollPace(saved); got != time.Duration(frames[want])*frameDur {
			t.Errorf("saved %q paces %v, want %d frames", saved, got, frames[want])
		}
		a := New(Config{PhysW: 320, PhysH: 240, Scroll: saved}, data.Ingest(nil, "", time.Now()), nil)
		if got := a.ScrollSpeed(); got != want {
			t.Errorf("saved %q saves back as %q, want %q", saved, got, want)
		}
	}
}

// scrollRow opens Options on the Scroll speed row.
func scrollRow(t *testing.T, a *App) panelEntry {
	t.Helper()
	a.screen = ScreenOptions
	a.buildPanel()
	expandOptionsForTest(a)
	for i, e := range a.panel.entries {
		if e.kind == "scroll" {
			a.panel.cursor = i
			return e
		}
	}
	t.Fatal("no Scroll speed row")
	return panelEntry{}
}

// The row offers the speeds in order and always shows one of them; Left and
// Right walk them end to end.
func TestScrollRowOffersSpeeds(t *testing.T) {
	labels := []string{"10 Hz", "20 Hz", "30 Hz", "60 Hz"}
	for _, saved := range []string{"10", "20", "30", "60", "", "warp"} {
		a := New(Config{PhysW: 320, PhysH: 240, Scroll: saved}, data.Ingest(nil, "", time.Now()), nil)
		e := scrollRow(t, a)
		if !reflect.DeepEqual(e.vals, labels) {
			t.Fatalf("saved %q: row offers %v, want %v", saved, e.vals, labels)
		}
		if e.idx < 0 || e.idx >= len(e.vals) || e.vals[e.idx] != offeredScroll(saved)+" Hz" {
			t.Fatalf("saved %q: row shows index %d of %v", saved, e.idx, e.vals)
		}
		for a.stepValue(-1) {
		}
		walked := []string{a.ScrollSpeed()}
		for a.stepValue(1) {
			walked = append(walked, a.ScrollSpeed())
		}
		if !reflect.DeepEqual(walked, ScrollValues) {
			t.Fatalf("saved %q: Right walks %v, want %v", saved, walked, ScrollValues)
		}
	}
}

// A settings file that says 10 loads as 10 and saves back as 10; garbage
// becomes the default 30.
func TestScrollSettingSurvivesASave(t *testing.T) {
	for saved, want := range map[string]string{"10": "10", "60": "60", "warp": "30", "": "30"} {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.WriteFile(path, []byte(`{"schema":1,"inset_x":15,"inset_y":15,"scroll":"`+saved+`"}`), 0644); err != nil {
			t.Fatal(err)
		}
		s, err := store.LoadSettings(path)
		if err != nil {
			t.Fatal(err)
		}
		a := New(Config{PhysW: 320, PhysH: 240, Scroll: s.Scroll}, data.Ingest(nil, "", time.Now()), nil)
		if e := scrollRow(t, a); e.vals[e.idx] != want+" Hz" {
			t.Fatalf("saved %q: row shows %q, want %s Hz", saved, e.vals[e.idx], want)
		}
		// the host saves the app's speed back, as cmd/misterzine does
		s.Scroll = a.ScrollSpeed()
		if err := store.Save(path, s); err != nil {
			t.Fatal(err)
		}
		if s, err = store.LoadSettings(path); err != nil || s.Scroll != want {
			t.Fatalf("saved %q: file now says %q (%v), want %q", saved, s.Scroll, err, want)
		}
	}
}

// A held direction on the list waits the hold delay, then moves one row
// every sixth frame at 10 Hz, as the other speeds keep their own pace.
func TestHeldListScrollAtEverySpeed(t *testing.T) {
	frames := map[string]int{"10": 6, "20": 3, "30": 2, "60": 1}
	rows := make([]data.Row, 200)
	for i := range rows {
		rows[i] = data.Row{K: fmt.Sprint(i), Title: fmt.Sprintf("Game %03d", i), Base: "Arcade"}
	}
	for _, speed := range ScrollValues {
		for _, delay := range []int{200, 300, 500} {
			now := time.Unix(100, 0)
			a := New(Config{PhysW: 320, PhysH: 240, HoldDelay: delay, Scroll: speed, TimerNow: func() time.Time { return now }}, data.Ingest(rows, "", now), nil)
			a.cursor = 10
			a.Paint()
			a.Handle(platform.Event{Key: platform.KeyDown, Pressed: true, At: now})
			if a.cursor != 11 {
				t.Fatalf("%s/%d: the press itself moved to %d", speed, delay, a.cursor)
			}
			deadline := now.Add(time.Duration(delay) * time.Millisecond)
			a.Frame(deadline.Add(-time.Millisecond))
			if a.cursor != 11 {
				t.Fatalf("%s/%d: repeated before the hold delay", speed, delay)
			}
			every := frames[speed]
			for f := 0; f < 4*every; f++ {
				a.Frame(deadline.Add(time.Duration(f) * frameDur))
				if want := 12 + f/every; a.cursor != want {
					t.Fatalf("%s/%d frame %d: row %d, want %d (a row every %d frames)", speed, delay, f, a.cursor, want, every)
				}
			}
			held := a.cursor
			a.Handle(platform.Event{Key: platform.KeyDown, At: deadline.Add(time.Duration(4*every) * frameDur)})
			a.Frame(deadline.Add(time.Second))
			if a.cursor != held || a.Repeating() {
				t.Fatalf("%s/%d: release kept scrolling", speed, delay)
			}
		}
	}
}

// The update log's timer-driven repeat keeps the same pace: 100 ms a line
// at 10 Hz after the hold delay.
func TestUpdateLogScrollAtTenHz(t *testing.T) {
	pace := 6 * frameDur
	now := time.Unix(100, 0)
	a := New(Config{PhysW: 320, PhysH: 240, HoldDelay: 300, Scroll: "10", Screensaver: "off", Now: func() time.Time { return now }}, data.Ingest(nil, "", now), nil)
	s := updater.State{ID: "run", Status: "completed"}
	for i := 0; i < 200; i++ {
		s.Lines = append(s.Lines, fmt.Sprintf("output line %d", i))
	}
	a.SetUpdate(s, true)
	a.Paint()
	a.updateView.scroll = 80
	a.Handle(platform.Event{Key: platform.KeyUp, Pressed: true, At: now})
	deadline := now.Add(300 * time.Millisecond)
	a.Tick(deadline)
	for step := 1; step <= 3; step++ {
		next := deadline.Add(time.Duration(step) * pace)
		a.Tick(next.Add(-time.Microsecond))
		if a.updateView.scroll != 81+step {
			t.Fatalf("step %d came early", step)
		}
		a.Tick(next)
		if a.updateView.scroll != 82+step {
			t.Fatalf("step %d came late", step)
		}
	}
}

// The preview runs a sample per speed, each at its own pace.
func TestScrollSamplesRunEverySpeed(t *testing.T) {
	now := time.Unix(100, 0)
	a := New(Config{PhysW: 320, PhysH: 240, Scroll: "10", TimerNow: func() time.Time { return now }}, data.Ingest(nil, "", now), nil)
	e := scrollRow(t, a)
	if len(e.vals) != len(ScrollValues) {
		t.Fatalf("%d samples for %d speeds", len(e.vals), len(ScrollValues))
	}
	a.Tick(now)
	if !a.OptionSamplesRunning() {
		t.Fatal("preview not animated")
	}
	frames := map[string]int{"10": 6, "20": 3, "30": 2, "60": 1}
	for frame := 1; frame <= 24; frame++ {
		a.OptionSampleFrame()
		a.Paint()
		for _, speed := range ScrollValues {
			if got := sampleListSelection(a.optionSamples.elapsed, scrollPace(speed), 0, 3); got != (frame/frames[speed])%3 {
				t.Fatalf("frame %d speed %s selection %d", frame, speed, got)
			}
		}
	}
}
