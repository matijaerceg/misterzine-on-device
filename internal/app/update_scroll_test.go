package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func TestUpdateLogHoldDelayPaceAndRelease(t *testing.T) {
	for _, status := range []string{"running", "completed"} {
		for _, delay := range []int{200, 300, 500} {
			for _, speed := range []string{"20", "30", "60"} {
				for _, key := range []platform.Key{platform.KeyUp, platform.KeyDown, platform.KeyPageUp, platform.KeyPageDown} {
					t.Run(fmt.Sprintf("%s/%d/%s/%v", status, delay, speed, key), func(t *testing.T) {
						now := time.Unix(100, 0)
						a := New(Config{PhysW: 320, PhysH: 240, HoldDelay: delay, Scroll: speed, Screensaver: "off", Now: func() time.Time { return now }}, data.Ingest(nil, "", now), nil)
						s := updater.State{ID: "run", Status: status}
						for i := 0; i < 200; i++ {
							s.Lines = append(s.Lines, fmt.Sprintf("output line %d", i))
						}
						a.SetUpdate(s, true)
						a.Paint()
						a.updateView.scroll = 80
						step := 1
						if key == platform.KeyPageUp || key == platform.KeyPageDown {
							step = a.updateView.lines
						}
						if key == platform.KeyDown || key == platform.KeyPageDown {
							step = -step
						}
						a.Handle(platform.Event{Key: key, Pressed: true, At: now})
						deadline := now.Add(time.Duration(delay) * time.Millisecond)
						if a.updateView.scroll != 80+step || !a.NextTick().Equal(deadline) || a.Repeating() {
							t.Fatal("first press/delay wrong, or updater entered exclusive frame loop")
						}
						a.Tick(deadline.Add(-time.Millisecond))
						if a.updateView.scroll != 80+step {
							t.Fatal("repeated before hold delay")
						}
						a.Tick(deadline)
						if a.updateView.scroll != 80+2*step {
							t.Fatal("did not repeat at hold delay")
						}
						next := deadline.Add(scrollPace(speed))
						a.Tick(next.Add(-time.Microsecond))
						if a.updateView.scroll != 80+2*step {
							t.Fatal("repeat faster than selected pace")
						}
						a.Tick(next)
						if a.updateView.scroll != 80+3*step {
							t.Fatal("repeat slower than selected pace")
						}
						a.Handle(platform.Event{Key: key, At: next})
						a.Tick(next.Add(time.Second))
						if a.updateView.scroll != 80+3*step || !a.NextTick().IsZero() {
							t.Fatal("release left scroll or wake timer running")
						}
					})
				}
			}
		}
	}
}

func TestUpdateHoldKeepsCancelAndResultTransitions(t *testing.T) {
	now := time.Unix(100, 0)
	cancels := 0
	a := New(Config{PhysW: 320, PhysH: 240, HoldDelay: 200, Now: func() time.Time { return now }, Action: func(kind, arg string) {
		if kind == "update-cancel" {
			cancels++
		}
	}}, data.Ingest(nil, "", now), nil)
	a.SetUpdate(updater.State{ID: "run", Status: "running"}, true)
	a.Handle(platform.Event{Key: platform.KeyUp, Pressed: true, At: now})
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
	a.Tick(now.Add(1999 * time.Millisecond))
	if cancels != 0 {
		t.Fatal("scroll hold changed cancellation delay")
	}
	a.Tick(now.Add(2 * time.Second))
	a.Tick(now.Add(3 * time.Second))
	if cancels != 1 {
		t.Fatalf("cancellation calls = %d", cancels)
	}
	a.SetUpdate(updater.State{ID: "run", Status: "completed"}, false)
	a.Tick(now.Add(4 * time.Second))
	if a.Screen() != ScreenUpdate || a.UpdateState().Status != "completed" {
		t.Fatal("held scrolling blocked completion")
	}
	a.Handle(platform.Event{Key: platform.KeyBack, At: now})
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
	if a.Screen() != ScreenOptions || a.rep.held {
		t.Fatal("leaving result retained log repeat")
	}
	a.Handle(platform.Event{Key: platform.KeyUp, At: now})
}
