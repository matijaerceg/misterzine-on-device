package app

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// With render-ahead on, the frames come from the producer goroutine:
// every paint hands back a full physical frame, the cabinet is drawn,
// the last frame before the launch is black and the producer has
// stopped by then.
func TestLaunchCabRendersAhead(t *testing.T) {
	a, clock, launched := cabApp(true)
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
