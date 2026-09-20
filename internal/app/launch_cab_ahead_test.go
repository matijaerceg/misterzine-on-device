package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// With render-ahead on, the frames come from the producer goroutine:
// every paint hands back a full physical frame, the cabinet is drawn,
// the last frame before the launch is black and the producer has
// stopped by then.
func TestLaunchCabRendersAhead(t *testing.T) {
	a, clock, launched := cabApp(true)
	a.cfg.LaunchTransition = "always"
	a.EnableRenderAhead()
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	if a.cab.ahead == nil {
		t.Fatal("no frame producer started")
	}
	lit := 0
	for a.LaunchCabRunning() {
		a.Tick(*clock)
		frame, dirty := a.Paint()
		if a.LaunchCabRunning() {
			if len(dirty) != 1 || dirty[0] != a.physical.Rect || frame.Rect != a.physical.Rect {
				t.Fatalf("frame %v dirty %v, want the whole physical frame %v", frame.Rect, dirty, a.physical.Rect)
			}
			for i := 0; i < len(frame.Pix); i += 4 {
				if frame.Pix[i] != 0 || frame.Pix[i+1] != 0 {
					lit++
					break
				}
			}
		}
		*clock = clock.Add(frameDur)
	}
	if lit < 20 {
		t.Fatalf("only %d frames drew the cabinet", lit)
	}
	if len(*launched) != 1 {
		t.Fatalf("launched %v after the animation", *launched)
	}
	select {
	case <-a.cab.ahead.done:
	default:
		t.Fatal("the producer is still running after the launch")
	}
	frame, _ := a.Paint()
	for i := 0; i < len(frame.Pix); i += 4 {
		if frame.Pix[i] != 0 || frame.Pix[i+1] != 0 || frame.Pix[i+2] != 0 {
			t.Fatal("the frame left on screen for the core is not black")
		}
	}
}

// Back stops the producer along with the animation.
func TestLaunchCabBackStopsTheProducer(t *testing.T) {
	a, clock, launched := cabApp(true)
	a.cfg.LaunchTransition = "always"
	a.EnableRenderAhead()
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	a.Tick(*clock)
	a.Paint()
	ah := a.cab.ahead
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	if a.cab.active || a.cab.ahead != nil {
		t.Fatal("Back did not drop the animation and its producer")
	}
	select {
	case <-ah.done:
	default:
		t.Fatal("the producer is still running after Back")
	}
	if len(*launched) != 0 {
		t.Fatal("Back launched the game")
	}
}

// The display's blanks jitter around the 60 Hz step and drift against it.
// The frames still go out one at a time, in order: no frame shown twice,
// none skipped.
func TestLaunchCabFramesFollowTheBlanks(t *testing.T) {
	a, clock, _ := cabApp(true)
	a.cfg.LaunchTransition = "always"
	a.EnableRenderAhead()
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	// 16.65 ms average, alternating a millisecond either side
	steps := []time.Duration{15650 * time.Microsecond, 17650 * time.Microsecond}
	last := -1
	for n := 0; a.LaunchCabRunning(); n++ {
		a.Tick(*clock)
		waitCabFrames(a, 1) // the test clock outruns the renderer; the boards do not
		a.Paint()
		if a.LaunchCabRunning() {
			shown := a.cab.ahead.taken - 1
			if shown != last+1 {
				t.Fatalf("paint %d showed frame %d after frame %d", n, shown, last)
			}
			last = shown
		}
		*clock = clock.Add(steps[n%2])
	}
	if last < 150 {
		t.Fatalf("only %d frames shown", last+1)
	}
}

// A stall skips ahead instead of playing the missed frames late.
func TestLaunchCabStallSkipsAhead(t *testing.T) {
	a, clock, _ := cabApp(true)
	a.cfg.LaunchTransition = "always"
	a.EnableRenderAhead()
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyStart, At: *clock})
	for i := 0; i < 5; i++ {
		a.Tick(*clock)
		a.Paint()
		*clock = clock.Add(frameDur)
	}
	*clock = clock.Add(10 * frameDur) // the display stalled for ten blanks
	a.Tick(*clock)
	waitCabFrames(a, cabAheadDepth)
	a.Paint()
	if shown := a.cab.ahead.taken - 1; shown < 10 {
		t.Fatalf("frame %d shown after a ten-frame stall, want the clock's frame", shown)
	}
	a.cab.stopAhead()
}

// waitCabFrames lets the producer get n frames ahead.
func waitCabFrames(a *App, n int) {
	for len(a.cab.ahead.frames) < n && !a.cab.ahead.closed {
		select {
		case <-a.cab.ahead.done:
			return
		case <-time.After(time.Millisecond):
		}
	}
}
