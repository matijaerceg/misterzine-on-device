package app

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func saverApp() (*App, *time.Time) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	a := New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 40, SafeInsetY: 40,
		Now: func() time.Time { return now }}, data.Ingest(nil, "test", now), nil)
	return a, &now
}

func TestSaverIdleAndWake(t *testing.T) {
	for _, key := range []platform.Key{platform.KeyStart, platform.KeyEnter, platform.KeyBack, platform.KeySpace, platform.KeyDown} {
		a, clock := saverApp()
		launches := 0
		a.cfg.Status = func(int) data.Status { return data.StatusCurrent }
		a.cfg.Exists = func(string) bool { return true }
		a.cfg.Launch = func(string) { launches++ }
		a.SetData(data.Ingest([]data.Row{{K: "test", Title: "Test", Base: "Arcade", MRA: "_Arcade/Test.mra"}}, "test", *clock), nil)
		if a.Screensaver() != "1" || !a.NextTick().Equal(clock.Add(time.Minute)) {
			t.Fatal("new session must schedule the default one-minute timeout")
		}
		a.Tick(clock.Add(time.Minute - time.Millisecond))
		if a.ScreensaverActive() {
			t.Fatal("started early")
		}
		*clock = clock.Add(time.Minute)
		a.Tick(*clock)
		if !a.ScreensaverActive() || !a.NextTick().After(*clock) {
			t.Fatal("saver did not start or scheduled a busy loop")
		}
		for i := 0; i < 2; i++ {
			a.Handle(platform.Event{Key: key, Pressed: true, At: *clock})
		}
		a.Tick(clock.Add(5 * time.Second))
		if a.ScreensaverActive() || a.Screen() != ScreenList || a.Sort() != data.SortUpdated || a.Repeating() || launches != 0 {
			t.Fatalf("wake key %v leaked into browsing", key)
		}
		a.Handle(platform.Event{Key: key, At: clock.Add(5 * time.Second)})
		if key == platform.KeyStart {
			a.Handle(platform.Event{Key: key, Pressed: true, At: clock.Add(6 * time.Second)})
			a.Handle(platform.Event{Key: key, At: clock.Add(6 * time.Second)})
			if launches != 1 {
				t.Fatal("deliberate Start no longer launches")
			}
		}
		a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: clock.Add(6 * time.Second)})
		if a.Screen() != ScreenOptions {
			t.Fatal("fresh press after wake was lost")
		}
	}
}

func TestSaverActivityAndCalendarCorrection(t *testing.T) {
	timer := time.Unix(100, 0)
	calendar := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	a := New(Config{PhysW: 320, PhysH: 240,
		Now: func() time.Time { return calendar }, TimerNow: func() time.Time { return timer }}, data.Ingest(nil, "test", calendar), nil)
	calendar = calendar.AddDate(10, 0, 0)
	a.SetNet("data just updated")
	a.Tick(timer.Add(59 * time.Second))
	if a.ScreensaverActive() || !a.NextTick().Equal(timer.Add(time.Minute)) {
		t.Fatal("calendar correction moved idle deadline")
	}
	timer = timer.Add(59 * time.Second)
	a.Handle(platform.Event{Key: platform.KeyDown, Pressed: true, At: timer})
	a.Tick(timer.Add(2 * time.Minute))
	if a.ScreensaverActive() {
		t.Fatal("held input is not idle")
	}
	timer = timer.Add(2 * time.Minute)
	a.Handle(platform.Event{Key: platform.KeyDown, At: timer})
	a.SetNet("data just updated again")
	a.Tick(timer.Add(time.Minute))
	if !a.ScreensaverActive() {
		t.Fatal("background refresh prevented idle saver")
	}
}

func TestSaverTextWakeDoesNotSearch(t *testing.T) {
	a, clock := saverApp()
	a.startSaver(*clock)
	ev := platform.Event{Key: platform.KeyOther, Code: 30, Text: 'a', Pressed: true, At: *clock}
	a.Handle(ev)
	a.Handle(ev)
	if a.Search() != "" {
		t.Fatal("wake text reached search")
	}
	ev.Pressed = false
	a.Handle(ev)
	ev.Pressed = true
	a.Handle(ev)
	if a.Search() != "a" {
		t.Fatal("search did not resume after release")
	}
}

func TestSaverPreviewAndOff(t *testing.T) {
	a, clock := saverApp()
	a.cfg.Screensaver = "off"
	a.Tick(clock.Add(24 * time.Hour))
	if a.ScreensaverActive() || !a.nextSaverTick().IsZero() {
		t.Fatal("Off still started or scheduled the saver")
	}
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "screensaver" {
			a.panel.cursor = i
		}
	}
	a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyEnter, At: *clock})
	if !a.ScreensaverActive() || a.Repeating() || a.Screensaver() != "off" {
		t.Fatal("preview must work while Off and survive releasing A")
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	if a.Screen() != ScreenOptions {
		t.Fatal("preview wake navigated away")
	}
}

func TestSaverWakeCannotCancelUpdate(t *testing.T) {
	a, clock := saverApp()
	cancels := 0
	a.cfg.Action = func(kind, arg string) {
		if kind == "update-cancel" {
			cancels++
		}
	}
	a.SetUpdate(updater.State{ID: "run", Status: "running", Started: *clock}, true)
	*clock = clock.Add(time.Minute)
	a.Tick(*clock)
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	*clock = clock.Add(3 * time.Second)
	a.Tick(*clock)
	if cancels != 0 || a.Screen() != ScreenUpdate || a.ScreensaverActive() {
		t.Fatal("wake hold affected the update")
	}
	a.Handle(platform.Event{Key: platform.KeyBack, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	a.Tick(clock.Add(3 * time.Second))
	if cancels != 1 {
		t.Fatal("a new deliberate hold must still cancel")
	}
}

func TestSaverSweepsEveryPixel(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a, _ := saverApp()
		a.SetRotation(rot)
		c := a.logical
		covered := make([]bool, c.W()*c.H())
		remaining := len(covered)
		mask := a.saverMask(c.H())
		for travel := 0; travel < c.W()+mask.Rect.Dx() && remaining > 0; travel++ {
			c.Fill(c.Rect, color.RGBA{R: 200, G: 160, B: 120, A: 255})
			a.saver.travel = travel
			a.paintSaver(c)
			for y := 0; y < c.H(); y++ {
				for x := 0; x < c.W(); x++ {
					p := c.RGBAAt(x, y)
					if p == (color.RGBA{A: 255}) {
						i := y*c.W() + x
						if !covered[i] {
							covered[i] = true
							remaining--
						}
					} else if p != (color.RGBA{R: 50, G: 40, B: 30, A: 255}) {
						t.Fatalf("pixel not black or correctly dimmed: %v", p)
					}
				}
			}
		}
		if remaining != 0 {
			t.Fatalf("rotation %v: %d pixels never swept black", rot, remaining)
		}
		// Include the physical frame bounds after tate rotation.
		physical := image.NewRGBA(image.Rect(0, 0, 320, 240))
		if got := gfx.RotateRect(physical, c.RGBA, c.Rect, rot); got != physical.Rect {
			t.Fatalf("saver left physical margins uncovered: %v", got)
		}
	}
}
