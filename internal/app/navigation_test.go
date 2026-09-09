package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestOptionsHoldStopsAtEndsAndFreshPressWraps(t *testing.T) {
	for _, drive := range []string{"tick", "frame"} {
		for _, key := range []platform.Key{platform.KeyUp, platform.KeyDown} {
			t.Run(key.String()+"/"+drive, func(t *testing.T) {
				a, clock := saverApp()
				a.openPanel(ScreenOptions)
				last := len(a.panel.entries) - 1
				end, other, start := 0, last, 1
				if key == platform.KeyDown {
					end, other, start = last, 0, last-1
				}
				a.panel.cursor = start
				press := func() { a.Handle(platform.Event{Key: key, Pressed: true, At: *clock}) }
				hold := func() {
					for n := 0; n < 180; n++ {
						*clock = clock.Add(frameDur)
						if drive == "frame" {
							a.Frame(*clock)
						} else {
							a.Tick(*clock)
						}
					}
				}
				press()
				if a.panel.cursor != end {
					t.Fatalf("first press: got %d, want end %d", a.panel.cursor, end)
				}
				hold()
				press() // duplicate keydown is still the same hold
				if a.panel.cursor != end {
					t.Fatalf("holding crossed end %d to %d", end, a.panel.cursor)
				}
				a.Handle(platform.Event{Key: key, At: *clock})
				press()
				if a.panel.cursor != other {
					t.Fatalf("new press at end: got %d, want wrap to %d", a.panel.cursor, other)
				}
				hold()
				if a.panel.cursor != end {
					t.Fatalf("holding after a wrap: got %d, want stop at %d", a.panel.cursor, end)
				}
				a.Handle(platform.Event{Key: key, At: clock.Add(time.Second)})
			})
		}
	}
}
