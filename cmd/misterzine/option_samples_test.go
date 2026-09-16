//go:build linux

package main

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
)

func TestOptionPreviewExitPaintsBeforeWaitingForMoreInput(t *testing.T) {
	for _, holdDelay := range []bool{false, true} {
		name, key := "scroll-up", platform.KeyUp
		if holdDelay {
			name, key = "hold-delay-down", platform.KeyDown
		}
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			a := app.New(app.Config{PhysW: 320, PhysH: 240, TransitionsDisabled: true}, data.Ingest(nil, "test", now), nil)
			tap := func(k platform.Key) {
				a.Handle(platform.Event{Key: k, Pressed: true, At: now})
				a.Handle(platform.Event{Key: k, At: now})
			}
			tap(platform.KeyBack)
			tap(platform.KeyPageDown)
			tap(platform.KeyPageDown)
			tap(platform.KeyPageDown)
			tap(platform.KeyRight)
			for i := 0; i < 60 && !a.OptionSamplesRunning(); i++ {
				tap(platform.KeyDown)
			}
			if holdDelay {
				tap(platform.KeyDown)
			}
			if !a.OptionSamplesRunning() {
				t.Fatal("expected an animated preview")
			}
			a.Paint()
			// A closed framebuffer needs no hardware; Paint still consumes the
			// pending redraw before PresentWait reports that it is closed.
			h := &host{a: a, fb: &mister.FB{}, events: make(chan platform.Event, 1)}
			h.events <- platform.Event{Key: key, Pressed: true, At: now}
			h.optionSampleLoop()
			if a.OptionSamplesRunning() || !a.Repeating() {
				t.Fatal("expected to leave the preview with the navigation key still held")
			}
			if _, dirty := a.Paint(); dirty != nil {
				t.Fatal("preview loop returned with the new selection still waiting to be painted")
			}
		})
	}
}
