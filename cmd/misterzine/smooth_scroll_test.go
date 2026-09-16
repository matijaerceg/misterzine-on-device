//go:build linux

package main

import (
	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"path/filepath"
	"testing"
	"time"
)

func TestSmoothScrollPreferencePersists(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		root := t.TempDir()
		h := favoritesHost(root)
		h.a = app.New(app.Config{PhysW: 320, PhysH: 240, SmoothScrollDisabled: disabled}, data.Ingest(nil, "test", time.Now()), nil)
		h.saveAll(true)
		saved, err := store.LoadSettings(filepath.Join(root, "settings.json"))
		if err != nil {
			t.Fatal(err)
		}
		if saved.SmoothScrollDisabled != disabled {
			t.Fatal("smooth scroll preference not saved")
		}
		reopened := app.New(app.Config{PhysW: 320, PhysH: 240, SmoothScrollDisabled: saved.SmoothScrollDisabled}, data.Ingest(nil, "test", time.Now()), nil)
		if reopened.SmoothScrolling() == disabled {
			t.Fatal("smooth scroll preference not restored")
		}
	}
}
