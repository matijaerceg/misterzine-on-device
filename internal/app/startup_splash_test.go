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
	if a.splash.coverage != 255 || !a.splash.at.Equal(*clock) {
		t.Fatal("first list frame must start fully visible")
	}
	if !a.NextTick().Equal(clock.Add(startupFrame)) {
		t.Fatal("fade frame not scheduled")
	}
	*clock = clock.Add(startupFade / 2)
	a.Frame(*clock)
	if a.splash.coverage != 127 {
		t.Fatalf("halfway coverage = %d", a.splash.coverage)
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
		t.Fatal("fade retained work after one second")
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

func TestStartupLogoNativeSizeAndCounters(t *testing.T) {
	a, _ := saverApp()
	a.StartSplash()
	a.Paint()
	if got := a.splash.logo.Bounds().Size(); got.X != 160 || got.Y != 106 {
		t.Fatalf("logo was resized: %v", got)
	}
	for _, point := range [][2]int{{125, 27}, {135, 71}} {
		r, g, b, alpha := a.splash.logo.At(point[0], point[1]).RGBA()
		if r > 0x2000 || g > 0x2000 || b > 0x2000 || alpha != 0xffff {
			t.Fatalf("e counter at %v must be black, got %x %x %x %x", point, r, g, b, alpha)
		}
	}
}

func TestStartupSplashDithersWithoutBlending(t *testing.T) {
	a, clock := saverApp()
	a.StartSplash()
	a.Paint()
	*clock = clock.Add(500 * time.Millisecond)
	a.Tick(*clock)
	a.Paint()
	visible := 0
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := a.splash.mask.Pix[y*a.splash.mask.Stride+x]
			if v == 255 {
				visible++
			} else if v != 0 {
				t.Fatal("fade mask blends colours")
			}
		}
	}
	if visible != 32 {
		t.Fatalf("halfway dither keeps %d of 64 pixels", visible)
	}
	*clock = clock.Add(500 * time.Millisecond)
	a.Tick(*clock)
	if a.splash.enabled || a.splash.mask != nil {
		t.Fatal("splash must finish at one second")
	}
}
