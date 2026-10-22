package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// The list wraps like Options: a fresh Up on the first row goes to the last
// and a fresh L on the first letter to the last letter, but a held key
// stops at either end.
func TestListHoldStopsAtEndsAndFreshPressWraps(t *testing.T) {
	var rows []data.Row
	for i := 0; i < 26; i++ {
		for j := 0; j < 2; j++ {
			title := fmt.Sprintf("%c %d", 'A'+i, j)
			rows = append(rows, data.Row{Base: "Arcade", K: title, Title: title})
		}
	}
	for _, drive := range []string{"tick", "frame"} {
		for _, c := range []struct {
			key               platform.Key
			start, end, other string
		}{
			{platform.KeyUp, "A 1", "A 0", "Z 1"},
			{platform.KeyDown, "Z 0", "Z 1", "A 0"},
			{platform.KeyPageUp, "B 0", "A 0", "Z 0"},
			{platform.KeyPageDown, "Y 0", "Z 0", "A 0"},
		} {
			t.Run(c.key.String()+"/"+drive, func(t *testing.T) {
				clock := time.Unix(100, 0)
				a := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortAlphabetical, Now: func() time.Time { return clock }},
					data.Ingest(rows, "test", clock), nil)
				a.MoveToKey(c.start)
				press := func() { a.Handle(platform.Event{Key: c.key, Pressed: true, At: clock}) }
				hold := func() {
					for n := 0; n < 180; n++ {
						clock = clock.Add(frameDur)
						if drive == "frame" {
							a.Frame(clock)
						} else {
							a.Tick(clock)
						}
					}
				}
				press()
				if a.CursorKey() != c.end {
					t.Fatalf("first press: got %q, want end %q", a.CursorKey(), c.end)
				}
				hold()
				press() // duplicate keydown is still the same hold
				if a.CursorKey() != c.end {
					t.Fatalf("holding crossed end %q to %q", c.end, a.CursorKey())
				}
				a.Handle(platform.Event{Key: c.key, At: clock})
				press()
				if a.CursorKey() != c.other {
					t.Fatalf("new press at end: got %q, want wrap to %q", a.CursorKey(), c.other)
				}
				hold()
				if a.CursorKey() != c.end {
					t.Fatalf("holding after a wrap: got %q, want stop at %q", a.CursorKey(), c.end)
				}
				a.Handle(platform.Event{Key: c.key, At: clock.Add(time.Second)})
			})
		}
	}
}

func TestOptionsHoldStopsAtEndsAndFreshPressWraps(t *testing.T) {
	for _, drive := range []string{"tick", "frame"} {
		for _, key := range []platform.Key{platform.KeyUp, platform.KeyDown} {
			t.Run(key.String()+"/"+drive, func(t *testing.T) {
				a, clock := saverApp()
				a.openPanel(ScreenOptions)
				first := a.panel.cursor // past the Data heading
				last := len(a.panel.entries) - 1
				for a.panel.entries[last].kind != "quit" {
					last-- // the greyed build and data rows after Quit are not stops
				}
				end, other, start := first, last, first+1
				if key == platform.KeyDown {
					end, other, start = last, first, last-1
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
