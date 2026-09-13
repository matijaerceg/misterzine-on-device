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
					} else if p == (color.RGBA{R: 50, G: 40, B: 30, A: 255}) {
						continue
					} else if sx := x - (c.W() - travel%(c.W()+mask.Rect.Dx())); sx < 0 || sx >= mask.Rect.Dx() || mask.Pix[y*mask.Stride+sx] == 0 || mask.Pix[y*mask.Stride+sx] == saverInk {
						t.Fatalf("pixel not black, dimmed or a lit outline: %v", p)
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

func TestSaverRimGlints(t *testing.T) {
	a, _ := saverApp()
	c := a.logical
	m := a.saverMask(c.H())
	w, h := m.Rect.Dx(), m.Rect.Dy()
	rim, interior := 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch v := m.Pix[y*m.Stride+x]; {
			case v == 0:
			case v == saverInk:
				interior++
			case v < 1 || v > 9 || v == 5:
				t.Fatalf("bad facing code %d at %d,%d", v, x, y)
			default:
				rim++
				open := x == 0 || x == w-1
				for _, d := range []image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx, ny := x+d.X, y+d.Y
					if nx >= 0 && nx < w && ny >= 0 && ny < h && m.Pix[ny*m.Stride+nx] == 0 {
						open = true
					}
				}
				if !open {
					t.Fatalf("outline pixel %d,%d is not on the edge", x, y)
				}
			}
		}
	}
	if rim == 0 || rim*10 > interior {
		t.Fatalf("outline is %d pixels against %d interior", rim, interior)
	}
	// Sample frames across a full pass of the word: the glint must reach its
	// cap somewhere, while any given edge pixel is black most of the time.
	lit, total, peak := 0, 0, uint8(0)
	for travel := 0; travel < c.W()+w; travel += 7 {
		c.Fill(c.Rect, color.RGBA{R: 200, G: 160, B: 120, A: 255})
		a.saver.travel = travel
		a.paintSaver(c)
		x0 := c.W() - travel%(c.W()+w)
		for y := 0; y < c.H(); y++ {
			for x := max(x0, 0); x < min(x0+w, c.W()); x++ {
				if v := m.Pix[y*m.Stride+x-x0]; v != 0 && v != saverInk {
					total++
					if q := c.RGBAAt(x, y); q.R != 0 || q.G != 0 || q.B != 0 {
						lit++
						peak = max(peak, q.R, q.G, q.B)
					}
				}
			}
		}
	}
	if peak != saverGlintMax || peak > 102 {
		t.Fatalf("glint peak %d; it must reach exactly the cap and stay at or under 40%% of white", peak)
	}
	if lit*2 > total {
		t.Fatalf("outline lit for %d of %d pixel-frames; it should be black most of the time", lit, total)
	}
}

// The saver fades the screen to its quarter brightness over saverFade and
// only then starts the lettering across; a paint mid-fade is between the
// two levels.
func TestSaverFadesBeforeTheWordEnters(t *testing.T) {
	a, clock := saverApp()
	t0 := *clock
	a.startSaver(t0)
	if a.saver.shade != 256 || a.saver.travel != 0 {
		t.Fatalf("start: shade %d travel %d", a.saver.shade, a.saver.travel)
	}
	a.Tick(t0.Add(saverFade / 2))
	if a.saver.shade <= saverShade || a.saver.shade >= 256 || a.saver.travel != 0 {
		t.Fatalf("mid-fade: shade %d travel %d", a.saver.shade, a.saver.travel)
	}
	c := a.logical
	c.Fill(c.Rect, color.RGBA{R: 200, G: 160, B: 120, A: 255})
	a.paintSaver(c)
	if p := c.RGBAAt(0, 0); p.R <= 50 || p.R >= 200 {
		t.Fatalf("mid-fade pixel %v is not between full and a quarter", p)
	}
	a.Tick(t0.Add(saverFade))
	if a.saver.shade != saverShade || a.saver.travel != 0 {
		t.Fatalf("fade end: shade %d travel %d", a.saver.shade, a.saver.travel)
	}
	c.Fill(c.Rect, color.RGBA{R: 200, G: 160, B: 120, A: 255})
	a.paintSaver(c)
	if p := c.RGBAAt(0, 0); p != (color.RGBA{R: 50, G: 40, B: 30, A: 255}) {
		t.Fatalf("dark pixel %v is not a quarter", p)
	}
	a.Tick(t0.Add(saverFade + 10*saverFrame))
	if a.saver.travel != 10 {
		t.Fatalf("ten frames after the fade: travel %d", a.saver.travel)
	}
}

// The picture under the lettering is blurred: once the fade is done a
// hard edge has spread into a ramp, while a flat area is untouched and
// only dimmed; before the fade starts nothing is blurred.
func TestSaverBlursThePictureUnderTheLettering(t *testing.T) {
	a, clock := saverApp()
	t0 := *clock
	a.startSaver(t0)
	c := a.logical
	edge := c.W() / 2
	paintHalves := func() {
		c.Fill(c.Rect, color.RGBA{A: 255})
		c.Fill(image.Rect(0, 0, edge, c.H()), color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
	paintHalves()
	a.paintSaver(c)
	if p := c.RGBAAt(edge-1, 10); p.R != 255 {
		t.Fatalf("before the fade the edge pixel is %v, not sharp", p)
	}
	a.Tick(t0.Add(saverFade))
	paintHalves()
	a.paintSaver(c)
	r := c.W() / saverBlurDiv
	if r < 4 {
		t.Fatalf("radius %d too small to test", r)
	}
	if p := c.RGBAAt(edge-1, 10); p.R <= 8 || p.R >= 60 {
		t.Fatalf("edge pixel %v is not a ramp", p)
	}
	if p := c.RGBAAt(edge-4*r, 10); p != (color.RGBA{R: 63, G: 63, B: 63, A: 255}) {
		t.Fatalf("flat pixel %v is not just dimmed", p)
	}
	if p := c.RGBAAt(edge+4*r, 10); p != (color.RGBA{A: 255}) {
		t.Fatalf("flat black pixel %v changed", p)
	}
	// the ramp is monotonic across the edge
	last := 256
	for x := edge - 2*r; x <= edge+2*r; x++ {
		v := int(c.RGBAAt(x, 10).R)
		if v > last {
			t.Fatalf("ramp rises again at x=%d: %d after %d", x, v, last)
		}
		last = v
	}
}
