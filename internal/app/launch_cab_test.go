package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func cabApp(motion bool) (*App, *time.Time, *[]string) {
	a, clock := saverApp()
	var launched []string
	a.cfg.Status = func(int) data.Status { return data.StatusCurrent }
	a.cfg.Exists = func(string) bool { return true }
	a.cfg.Launch = func(p string) { launched = append(launched, p) }
	a.SetData(data.Ingest([]data.Row{{K: "test", Title: "Test", Base: "Arcade", MRA: "_Arcade/Test.mra", Img: "test", ImgSlots: []string{"snap", "title"}}}, "test", *clock), nil)
	if motion {
		a.EnablePageTransitions()
	}
	a.Paint()
	return a, clock, &launched
}

func TestLaunchCabPlaysThenLaunches(t *testing.T) {
	a, clock, launched := cabApp(true)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	if !a.LaunchCabRunning() || len(*launched) != 0 {
		t.Fatal("Start must begin the animation, not launch at once")
	}
	if a.cab.req.Slot != "title" {
		t.Fatalf("monitor shows %q, want the title shot", a.cab.req.Slot)
	}
	lit := 0
	for a.LaunchCabRunning() {
		if next := a.NextTick(); next.After(*clock) {
			t.Fatalf("next tick %v is past the frame clock %v", next, *clock)
		}
		a.Tick(*clock)
		frame, dirty := a.Paint()
		if a.LaunchCabRunning() {
			if len(dirty) == 0 {
				t.Fatal("animation frame painted nothing")
			}
			for i := 0; i < len(frame.Pix); i += 4 {
				if frame.Pix[i] != 0 || frame.Pix[i+1] != 0 {
					lit++
					break
				}
			}
			// input other than Back is swallowed while it plays
			a.Handle(platform.Event{Key: platform.KeyDown, Pressed: true, At: *clock})
			a.Handle(platform.Event{Key: platform.KeyDown, At: *clock})
			if a.screen != ScreenList || a.cursor != 0 {
				t.Fatal("a key moved the list during the animation")
			}
		}
		*clock = clock.Add(frameDur)
	}
	if lit < 20 {
		t.Fatalf("only %d frames drew the cabinet", lit)
	}
	if len(*launched) != 1 || (*launched)[0] != "_Arcade/Test.mra" {
		t.Fatalf("launched %v after the animation", *launched)
	}
}

func TestLaunchCabBackCancels(t *testing.T) {
	a, clock, launched := cabApp(true)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	*clock = clock.Add(300 * time.Millisecond)
	a.Tick(*clock)
	a.Paint()
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	if a.LaunchCabRunning() {
		t.Fatal("Back must cancel the launch")
	}
	*clock = clock.Add(2 * time.Second)
	a.Tick(*clock)
	if len(*launched) != 0 {
		t.Fatal("a cancelled launch still fired")
	}
	if _, dirty := a.Paint(); len(dirty) == 0 {
		t.Fatal("the list was not repainted after the cancel")
	}
}

func TestLaunchCabOffWithoutMotion(t *testing.T) {
	a, clock, launched := cabApp(false)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if a.LaunchCabRunning() || len(*launched) != 1 {
		t.Fatalf("static hosts must launch at once: running=%v launched=%v", a.LaunchCabRunning(), *launched)
	}
}
