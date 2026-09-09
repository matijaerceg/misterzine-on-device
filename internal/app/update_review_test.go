package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func TestUpdateHoldAcrossStartup(t *testing.T) {
	for _, releaseEarly := range []bool{false, true} {
		t.Run(fmt.Sprint(releaseEarly), func(t *testing.T) {
			now := time.Unix(100, 0)
			calls := []string{}
			a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }, Action: func(kind, id string) {
				if kind == "update-cancel" {
					calls = append(calls, id)
				}
			}}, data.Ingest(nil, "", now), nil)
			a.OpenUpdate()
			a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: now})
			if releaseEarly {
				a.Handle(platform.Event{Key: platform.KeyBack, At: now.Add(time.Second)})
			}
			now = now.Add(3 * time.Second)
			a.Tick(now)
			if len(calls) != 0 {
				t.Fatal("sent cancellation without run ID")
			}
			a.SetUpdate(updater.State{ID: "real-run", Status: "running"}, true)
			a.Tick(now)
			a.Tick(now.Add(time.Second))
			if releaseEarly {
				if len(calls) != 0 {
					t.Fatal("released startup hold cancelled run")
				}
			} else if !reflect.DeepEqual(calls, []string{"real-run"}) {
				t.Fatalf("lost/duplicated startup cancellation: %v", calls)
			}
		})
	}
}

func TestUpdateLogStaysStillWhileReading(t *testing.T) {
	now := time.Unix(100, 0)
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(nil, "", now), nil)
	s := updater.State{ID: "run", Status: "running"}
	for i := 0; i < 60; i++ {
		s.Lines = append(s.Lines, fmt.Sprintf("original line %d", i))
	}
	a.SetUpdate(s, true)
	a.Paint()
	a.handleUpdate(platform.Event{Key: platform.KeyPageUp, Pressed: true, At: now})
	a.handleUpdate(platform.Event{Key: platform.KeyPageUp, At: now})
	a.Paint()
	held := append([]string(nil), a.updateView.log...)
	offset := a.updateView.scroll
	s.Lines = []string{"newest output replaces the entire tail"}
	a.SetUpdate(s, false)
	a.Paint()
	if !reflect.DeepEqual(held, a.updateView.log) || a.updateView.scroll != offset {
		t.Fatal("incoming output moved paused log")
	}
	a.handleUpdate(platform.Event{Key: platform.KeyPageDown, Pressed: true, At: now})
	a.Paint()
	if a.updateView.scroll != 0 || !reflect.DeepEqual(a.updateView.log, s.Lines) {
		t.Fatal("returning to bottom did not resume live output")
	}
}

func TestUpdateProtectedRenders(t *testing.T) {
	now := time.Unix(100, 0)
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a := New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15, Rotation: rot, Now: func() time.Time { return now }}, data.Ingest(nil, "", now), nil)
		a.SetUpdate(updater.State{ID: "run", Status: "cancelling", Protected: true, CancelRequested: true, Reboot: true, Label: "Updating system", Stage: 2, Heartbeat: now, Lines: []string{"Writing system image", "Waiting for the system write to finish"}}, true)
		a.Paint()
		if a.updateView.lines < 1 {
			t.Fatal("no log space")
		}
		if dir := os.Getenv("MZ_RENDER_DIR"); dir != "" {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("update-protected-%d.png", rot)))
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, a.Logical()); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
}

func TestOptionsReviewsResultWithoutStartingUpdate(t *testing.T) {
	calls := 0
	now := time.Now()
	a := New(Config{PhysW: 320, PhysH: 240, Action: func(string, string) { calls++ }}, data.Ingest(nil, "", now), nil)
	a.SetUpdate(updater.State{ID: "old", Status: "completed", Lines: []string{"saved output"}}, false)
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "update-result" {
			a.panel.cursor = i
			break
		}
	}
	a.actPanel(platform.KeyEnter)
	if calls != 0 || a.Screen() != ScreenUpdate || a.update.ID != "old" {
		t.Fatal("result review started/replaced update")
	}
}
