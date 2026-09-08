package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// A scan notice can predate opening Update All or arrive during a run/result.
// Once it expires, the host must return to its normal polling delay instead of
// receiving the same past deadline and repainting as fast as vsync allows.
func TestUpdateNoticeExpiryReleasesWakeDeadline(t *testing.T) {
	for _, status := range []string{"starting", "running", "completed", "cancelled"} {
		for _, inherited := range []bool{true, false} {
			name := status + "/during"
			if inherited {
				name = status + "/inherited"
			}
			t.Run(name, func(t *testing.T) {
				now := time.Unix(1788900000, 0)
				a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(nil, "test", now), nil)
				if inherited {
					a.Notice("scanning card", time.Second)
					a.OpenUpdate()
				}
				a.SetUpdate(updater.State{ID: "run", Status: status, Started: now, Heartbeat: now}, true)
				if !inherited {
					a.Notice("card scan complete", time.Second)
				}
				deadline := now.Add(time.Second)
				a.Paint()
				now = deadline.Add(-time.Millisecond)
				a.Tick(now)
				if a.notice == "" || !a.NextTick().Equal(deadline) {
					t.Fatal("notice expired before its deadline")
				}
				for _, elapsed := range []time.Duration{0, time.Millisecond, 250 * time.Millisecond, time.Second} {
					now = deadline.Add(elapsed)
					a.Tick(now)
					a.Paint()
					if next := a.NextTick(); !next.IsZero() || a.notice != "" {
						t.Errorf("at deadline + %v: stale notice %q keeps wake deadline %v", elapsed, a.notice, next)
					}
					if a.Screen() != ScreenUpdate || a.UpdateState().Status != status {
						t.Fatal("notice expiry changed the update screen or run status")
					}
				}
				if status == "completed" || status == "cancelled" {
					a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
					if a.Screen() != ScreenOptions || a.notice != "" {
						t.Fatal("B did not return to Options with the expired notice cleared")
					}
				}
			})
		}
	}
}

func TestNoticeExpiresAtDeadline(t *testing.T) {
	for _, frame := range []bool{false, true} {
		now := time.Unix(1788900000, 0)
		a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(nil, "test", now), nil)
		a.Notice("message", time.Second)
		a.Paint()
		now = a.NextTick()
		tick := a.Tick
		if frame {
			tick = a.Frame
		}
		if !tick(now) || a.notice != "" || !a.NextTick().IsZero() {
			t.Fatalf("frame=%v: notice did not expire when the host woke at its deadline", frame)
		}
		if _, dirty := a.Paint(); len(dirty) == 0 {
			t.Fatal("expired message was not repainted")
		}
	}
}

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
