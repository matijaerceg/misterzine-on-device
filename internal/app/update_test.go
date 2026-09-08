package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func TestUpdateModalAndLongCancel(t *testing.T) {
	now := time.Now()
	calls := 0
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }, Action: func(kind, arg string) {
		if kind == "update-cancel" {
			if arg != "run" {
				t.Fatal(arg)
			}
			calls++
		}
	}}, data.Ingest(nil, "test", now), nil)
	a.SetUpdate(updater.State{ID: "run", Status: "running", Label: "Checking", Started: now}, true)
	for _, key := range []platform.Key{platform.KeyTab, platform.KeyEnter, platform.KeyStart, platform.KeySpace, platform.KeyLeft, platform.KeyRight} {
		a.Handle(platform.Event{Key: key, Pressed: true, At: now})
		a.Handle(platform.Event{Key: key, At: now})
		if a.Screen() != ScreenUpdate {
			t.Fatal("navigation escaped modal")
		}
	}
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
	a.Tick(now.Add(time.Second))
	a.Handle(platform.Event{Key: platform.KeyBack, At: now.Add(time.Second)})
	a.Tick(now.Add(3 * time.Second))
	if calls != 0 {
		t.Fatal("short B cancelled")
	}
	now = now.Add(4 * time.Second)
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
	a.Tick(now.Add(2 * time.Second))
	a.Tick(now.Add(4 * time.Second))
	if calls != 1 {
		t.Fatalf("cancel calls=%d", calls)
	}
	a.Handle(platform.Event{Key: platform.KeyBack, At: now.Add(4 * time.Second)})
	a.SetUpdate(updater.State{ID: "run", Status: "cancelled"}, false)
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now.Add(5 * time.Second)})
	if a.Screen() != ScreenOptions {
		t.Fatal("B did not return to Options")
	}
}

// Run this on ARM as well: converting a full epoch time to int before reducing
// the animation index overflows on the MiSTer's 32-bit CPU.
func TestUpdatePaintOnDeviceClock(t *testing.T) {
	now := time.Unix(1788900000, 0)
	for _, rotation := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a := New(Config{PhysW: 320, PhysH: 240, Rotation: rotation, SafeInsetX: 15, SafeInsetY: 15, Now: func() time.Time { return now }}, data.Ingest(nil, "test", now), nil)
		a.SetUpdate(updater.State{ID: "run", Status: "running", Stage: 2, Reboot: true, Protected: true, Started: now, Heartbeat: now, Lines: []string{"Linux will be updated"}}, true)
		for i := 0; i < 4; i++ {
			a.Tick(now.Add(time.Duration(i) * 250 * time.Millisecond))
			frame, dirty := a.Paint()
			if frame == nil || dirty == nil {
				t.Fatal("missing update frame")
			}
		}
	}
}
