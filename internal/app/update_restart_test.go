package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
	"testing"
)

func TestUpdateRestartRequiresFinishedChangedProgramAndFreshPress(t *testing.T) {
	a, now := saverApp()
	requests := 0
	a.cfg.Action = func(kind, id string) {
		if kind == "update-restart" {
			requests++
			if id != "run" {
				t.Fatal(id)
			}
		}
	}
	press := func() { a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: *now}) }
	release := func() { a.Handle(platform.Event{Key: platform.KeyEnter, At: *now}) }
	a.SetUpdate(updater.State{ID: "run", Status: "running"}, true)
	a.SetUpdateRestart("run", true)
	press()
	a.SetUpdate(updater.State{ID: "run", Status: "completed"}, false)
	a.SetUpdateRestart("run", true)
	press()
	if requests != 0 {
		t.Fatal("active run or held key requested restart")
	}
	release()
	press()
	press()
	if requests != 1 {
		t.Fatalf("restart requests: %d", requests)
	}
	release()
	a.SetUpdate(updater.State{ID: "next", Status: "completed"}, true)
	a.SetUpdateRestart("run", true)
	press()
	if a.UpdateRestartAvailable() || requests != 1 {
		t.Fatal("stale check offered restart for another run")
	}
}

func TestUpdateRestartCanBeDeferred(t *testing.T) {
	a, now := saverApp()
	a.SetUpdate(updater.State{ID: "run", Status: "errors"}, true)
	a.SetUpdateRestart("run", true)
	a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: *now})
	if a.Screen() != ScreenOptions {
		t.Fatal("Back did not leave the result")
	}
	a.SetUpdate(a.UpdateState(), true)
	if !a.UpdateRestartAvailable() {
		t.Fatal("reviewing the result lost the restart offer")
	}
}
