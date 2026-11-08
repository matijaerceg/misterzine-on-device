package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestFilterCountsFollowFirstFrame(t *testing.T) {
	for _, tc := range []struct {
		name string
		wipe bool
	}{{"wipe", true}, {"instant", false}} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			reads := 0
			a := New(Config{PhysW: 320, PhysH: 240, TransitionsDisabled: !tc.wipe,
				Now: func() time.Time { return now }, Status: func(int) data.Status { reads++; return data.StatusCurrent },
			}, data.Ingest([]data.Row{{K: "game", Title: "Game", Base: "Arcade"}}, "", now), nil)
			a.SetFilters(data.Filters{Install: data.InstallFound})
			a.EnablePageTransitions()
			a.Paint()
			reads = 0
			a.actList(platform.KeyTab)
			found := func() panelEntry {
				t.Helper()
				for _, row := range a.panel.entries {
					if row.kind == "install" && row.value == data.InstallFound {
						return row
					}
				}
				t.Fatal("saved choice missing from first frame")
				return panelEntry{}
			}
			if row := found(); !row.checked || row.showCount || reads != 0 {
				t.Fatal("opening read the catalogue or hid the saved choice")
			}
			a.Tick(now)
			a.Paint()
			if tc.wipe {
				now = now.Add(pageWipeDuration)
				a.Tick(now)
				if reads != 0 || found().showCount {
					t.Fatal("counted during the page transition")
				}
				a.Paint()
			}
			a.actPanel(platform.KeyDown)
			cursor := a.panel.cursor
			now = now.Add(frameDur)
			a.Tick(now)
			if row := found(); !row.showCount || row.count != 1 || a.panel.cursor != cursor {
				t.Fatal("settled count or cursor is wrong")
			}

			// Leaving immediately must discard the work from a fresh visit.
			a.actPanel(platform.KeyTab)
			a.actList(platform.KeyTab)
			a.Paint()
			a.actPanel(platform.KeyTab)
			reads = 0
			now = now.Add(time.Second)
			a.Tick(now)
			if a.screen != ScreenList || reads != 0 {
				t.Fatal("closed Filters still performed a catalogue count")
			}
		})
	}
}
