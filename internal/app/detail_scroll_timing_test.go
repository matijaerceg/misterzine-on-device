package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestDetailScrollDisplayCadence(t *testing.T) {
	for _, interval := range []time.Duration{16 * time.Millisecond, time.Second / 60, 20 * time.Millisecond} {
		t.Run(interval.String(), func(t *testing.T) {
			clock := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
			a := New(Config{PhysW: 320, PhysH: 240, TimerNow: func() time.Time { return clock }}, data.Ingest(nil, "", clock), nil)
			a.screen = ScreenDetails
			a.detail.lines = 100
			a.detail.scroll = 100
			a.detail.next, a.detail.last = clock, clock
			start := clock
			for n := 0; n < 30; n++ {
				clock = clock.Add(interval)
				before := a.detail.pixel
				if !a.DetailScrollFrame(clock) || a.detail.pixel <= before {
					t.Fatal("display refresh skipped")
				}
				// The normal tick in the animation loop must not advance a second time.
				pixel := a.detail.pixel
				a.tickDetailScroll(clock)
				if a.detail.pixel != pixel {
					t.Fatal("advanced twice on one refresh")
				}
			}
			want := int(clock.Sub(start) * detailStep / detailFrame)
			if a.detail.pixel != want {
				t.Fatalf("pixel=%d want=%d", a.detail.pixel, want)
			}
			if !a.DetailScrollRunning() || a.Repeating() {
				t.Fatal("released scroll must keep display loop alive")
			}
			a.screen = ScreenList
			if a.DetailScrollRunning() {
				t.Fatal("scroll loop survives leaving details")
			}
		})
	}
}

func TestDetailScrollReversalAndEnd(t *testing.T) {
	clock := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	a := New(Config{PhysW: 320, PhysH: 240, TimerNow: func() time.Time { return clock }}, data.Ingest([]data.Row{{Base: "Arcade", K: "game", Title: "Game"}}, "", clock), nil)
	a.screen = ScreenDetails
	a.detail.lines = 4
	a.actDetails(platform.KeyDown)
	clock = clock.Add(20 * time.Millisecond)
	a.DetailScrollFrame(clock)
	if a.detail.pixel == 0 {
		t.Fatal("did not start")
	}
	a.actDetails(platform.KeyUp)
	clock = clock.Add(time.Second)
	a.DetailScrollFrame(clock)
	if a.detail.pixel != 0 || a.DetailScrollRunning() {
		t.Fatal("reversed scroll did not stop at top")
	}
}
