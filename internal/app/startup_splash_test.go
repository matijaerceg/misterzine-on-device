package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

func TestStartupSplashTimingAndInput(t *testing.T) {
	a, clock := saverApp()
	a.StartSplash()
	// Loading before the first visible frame must not consume the fade.
	*clock = clock.Add(5 * time.Second)
	a.Paint()
	if a.splash.opacity != 255 || !a.splash.at.Equal(*clock) {
		t.Fatal("first list frame must start fully visible")
	}
	if !a.NextTick().Equal(clock.Add(startupFrame)) {
		t.Fatal("fade frame not scheduled")
	}
	*clock = clock.Add(time.Second)
	a.Frame(*clock)
	if a.splash.opacity != 127 {
		t.Fatalf("halfway opacity = %d", a.splash.opacity)
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *clock})
	if a.screen == ScreenList {
		t.Fatal("splash blocked opening Options")
	}
	a.Tick(*clock)
	if a.splash.enabled {
		t.Fatal("splash remained after leaving list")
	}
	a.StartSplash()
	if a.splash.enabled {
		t.Fatal("splash replayed in same session")
	}
}

func TestStartupSplashExpiresCleanly(t *testing.T) {
	a, clock := saverApp()
	a.StartSplash()
	a.Paint()
	*clock = clock.Add(startupFade)
	if !a.Tick(*clock) {
		t.Fatal("expiry must repaint")
	}
	frame, _ := a.Paint()
	if a.splash.enabled || !a.splash.next.IsZero() || a.splash.logo != nil {
		t.Fatal("fade retained work after two seconds")
	}
	want := append([]byte(nil), frame.Pix...)
	a.all = true
	frame, _ = a.Paint()
	for i, p := range want {
		if p != frame.Pix[i] {
			t.Fatal("splash left pixels behind")
		}
	}
}
