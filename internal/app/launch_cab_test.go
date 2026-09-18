package app

import (
	"image"
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
	// the page must not come back while the core loads
	*clock = clock.Add(frameDur)
	a.Tick(*clock)
	frame, _ := a.Paint()
	for i := 0; i < len(frame.Pix); i += 4 {
		if frame.Pix[i] != 0 || frame.Pix[i+1] != 0 || frame.Pix[i+2] != 0 {
			t.Fatal("the page was painted after the launch")
		}
	}
	if !a.NextTick().IsZero() {
		t.Fatal("nothing should be scheduled after the launch")
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

func TestLaunchCabPreviewDoesNotLaunch(t *testing.T) {
	a, clock, launched := cabApp(false) // works without the animation clock or preference
	a.cfg.TransitionsDisabled = true
	a.previewLaunchCab()
	if !a.LaunchCabRunning() {
		t.Fatal("preview did not start")
	}
	var last []image.Rectangle
	for a.LaunchCabRunning() {
		a.Tick(*clock)
		_, last = a.Paint()
		*clock = clock.Add(frameDur)
	}
	if len(*launched) != 0 {
		t.Fatalf("preview launched %v", *launched)
	}
	if len(last) == 0 {
		t.Fatal("the screen was not repainted after the preview")
	}
}

func TestLaunchCabEndsBlack(t *testing.T) {
	a, clock, _ := cabApp(true)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	var last []byte
	for a.LaunchCabRunning() {
		a.Tick(*clock)
		if a.LaunchCabRunning() {
			frame, _ := a.Paint()
			last = append(last[:0], frame.Pix...)
		}
		*clock = clock.Add(frameDur)
	}
	for i := 0; i < len(last); i += 4 {
		if last[i] != 0 || last[i+1] != 0 || last[i+2] != 0 {
			t.Fatalf("the last frame is not black at byte %d", i)
		}
	}
}

func TestLaunchCabHoldStart(t *testing.T) {
	// a tap launches plainly on the release
	a, clock, launched := cabApp(true)
	a.cfg.LaunchTransition = "hold"
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if a.LaunchCabRunning() || len(*launched) != 0 {
		t.Fatal("a press must wait")
	}
	if next := a.NextTick(); !next.Equal(clock.Add(cabHoldStart)) {
		t.Fatalf("next tick %v, want the hold deadline", next)
	}
	*clock = clock.Add(50 * time.Millisecond)
	a.Tick(*clock)
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	if a.LaunchCabRunning() || len(*launched) != 1 {
		t.Fatalf("a tap must launch at once: running=%v launched=%v", a.LaunchCabRunning(), *launched)
	}

	// a hold plays the animation, and the later release does nothing
	a, clock, launched = cabApp(true)
	a.cfg.LaunchTransition = "hold"
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	*clock = clock.Add(cabHoldStart)
	a.Tick(*clock)
	if !a.LaunchCabRunning() || len(*launched) != 0 {
		t.Fatal("a held Start must start the animation")
	}
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	if !a.LaunchCabRunning() || len(*launched) != 0 {
		t.Fatal("the release after the hold must not launch")
	}
	for a.LaunchCabRunning() {
		a.Tick(*clock)
		a.Paint()
		*clock = clock.Add(frameDur)
	}
	if len(*launched) != 1 {
		t.Fatalf("launched %v after the animation", *launched)
	}

	// "always" is untouched by the hold machinery
	a, clock, launched = cabApp(true)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if !a.LaunchCabRunning() {
		t.Fatal("always must animate on the press")
	}
}

func TestLaunchCabOffWithoutMotion(t *testing.T) {
	a, clock, launched := cabApp(false)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if a.LaunchCabRunning() || len(*launched) != 1 {
		t.Fatalf("static hosts must launch at once: running=%v launched=%v", a.LaunchCabRunning(), *launched)
	}
}

// A vertical game's shot keeps its 3:4 shape on the monitor: it sits
// centred on black bars instead of being stretched across the 4:3 screen.
func TestLaunchCabTateShotIsLetterboxed(t *testing.T) {
	shot := image.NewRGBA(image.Rect(0, 0, 72, cabTexH))
	for i := range shot.Pix {
		shot.Pix[i] = 255
	}
	tex := cabScreenImage(shot)
	if tex.Rect.Dx() != cabTexW || tex.Rect.Dy() != cabTexH {
		t.Fatalf("texture is %v, want %dx%d", tex.Rect, cabTexW, cabTexH)
	}
	bar := (cabTexW - 72) / 2
	for y := 0; y < cabTexH; y++ {
		for x := 0; x < cabTexW; x++ {
			o := tex.PixOffset(x, y)
			inside := x >= bar && x < bar+72
			if lit := tex.Pix[o] == 255; lit != inside {
				t.Fatalf("texel %d,%d lit=%v, want the shot only between the bars", x, y, lit)
			}
			if tex.Pix[o+3] != 255 {
				t.Fatalf("texel %d,%d is not opaque", x, y)
			}
		}
	}
	if cabScreenImage(shot) != tex {
		t.Fatal("the letterboxed texture is not cached")
	}
	full := image.NewRGBA(image.Rect(0, 0, cabTexW, cabTexH))
	if cabScreenImage(full) != full {
		t.Fatal("a full-size shot must be used as it is")
	}
}

// The key that starts the animation is released while it plays. The
// release must still be seen, or the next press of that key reads as a
// key still held and does nothing: the Preview row needed two presses.
func TestLaunchCabReleaseDuringAnimationIsSeen(t *testing.T) {
	a, clock, launched := cabApp(true)
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if !a.LaunchCabRunning() {
		t.Fatal("Start did not begin the animation")
	}
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock}) // released mid-animation
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyBack, At: *clock})
	if a.LaunchCabRunning() || len(*launched) != 0 {
		t.Fatal("Back did not cancel")
	}
	if a.down[platform.KeyStart] {
		t.Fatal("Start still counts as held after the animation")
	}
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if !a.LaunchCabRunning() {
		t.Fatal("the next press of Start did nothing")
	}
}
