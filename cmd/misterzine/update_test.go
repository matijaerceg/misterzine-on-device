//go:build linux

package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/images"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func recoveryHost(t *testing.T, root string) *host {
	t.Helper()
	h := &host{root: root, quit: make(chan struct{}), lg: log.New(io.Discard, "", 0)}
	h.img = images.New(filepath.Join(root, "images"), nil, h.lg, 1<<20)
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Action: func(kind, id string) {
		if kind == "update-dismiss" {
			h.dismissUpdate(id)
		}
	}}, data.Ingest(nil, "test", time.Now()), nil)
	h.initUpdates()
	t.Cleanup(func() { close(h.quit); h.img.Close() })
	return h
}

func TestRecoveryDismissalSurvivesRelaunch(t *testing.T) {
	for _, success := range []bool{false, true} {
		status := "interrupted"
		if success {
			status = "restarted"
		}
		t.Run(status, func(t *testing.T) {
			root := t.TempDir()
			stored := updater.State{ID: "old-run", Status: "running", Boot: "previous-boot", SawSuccess: success, Lines: []string{"original output"}}
			if err := store.Save(updater.StatePath(root), stored); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(updater.StatePath(root))
			if err != nil {
				t.Fatal(err)
			}
			first := recoveryHost(t, root)
			if first.a.Screen() != app.ScreenUpdate || first.a.UpdateState().Status != status {
				t.Fatal("unseen recovery warning did not open")
			}
			// Simply opening the app is not acknowledgement.
			again := recoveryHost(t, root)
			if again.a.Screen() != app.ScreenUpdate {
				t.Fatal("warning vanished before the user dismissed it")
			}
			again.a.Handle(platform.Event{Key: platform.KeyBack, Pressed: true, At: time.Now()})
			if again.a.Screen() != app.ScreenOptions {
				t.Fatal("B did not return to Options")
			}
			next := recoveryHost(t, root)
			if next.a.Screen() != app.ScreenList || next.a.UpdateState().Status != status {
				t.Fatalf("dismissed warning reopened: screen=%s status=%s", next.a.Screen(), next.a.UpdateState().Status)
			}
			after, err := os.ReadFile(updater.StatePath(root))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("dismissal modified the original recovery evidence")
			}
			stored.ID = "later-run"
			if err := store.Save(updater.StatePath(root), stored); err != nil {
				t.Fatal(err)
			}
			if later := recoveryHost(t, root); later.a.Screen() != app.ScreenUpdate {
				t.Fatal("an old acknowledgement hid a later interrupted run")
			}
		})
	}
}
