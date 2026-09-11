//go:build linux

package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// The list preferences (title font, list shots, date format) round-trip
// through settings.json like the other Options rows.
func TestListPreferencesPersist(t *testing.T) {
	root := t.TempDir()
	h := favoritesHost(root)
	rows := []data.Row{{K: "a", Title: "Alpha", Updated: "2026-09-10"}}
	open := func() {
		h.a = app.New(app.Config{PhysW: 320, PhysH: 240,
			TitleFont: h.settings.TitleFont, ListShot: h.settings.ListShot, DateFormat: h.settings.DateFormat,
			SettingsChanged: func() { h.setDirty = true },
		}, data.Ingest(rows, "test", time.Now()), nil)
	}
	open()
	if h.a.TitleFont() != "tall" || h.a.ListShot() != "gameplay" || h.a.DateFormat() != "mm-dd" {
		t.Fatal("fresh defaults: narrow tall titles, gameplay shots, MM-DD")
	}
	tap := func(key platform.Key) {
		now := time.Now()
		h.a.Handle(platform.Event{Key: key, Pressed: true, At: now})
		h.a.Handle(platform.Event{Key: key, At: now})
	}
	tap(platform.KeyBack)
	for i := 0; i < 9; i++ {
		tap(platform.KeyDown)
	}
	tap(platform.KeyLeft) // Title font: narrow
	tap(platform.KeyDown)
	tap(platform.KeyRight) // List shots: title
	tap(platform.KeyDown)
	tap(platform.KeyRight)
	tap(platform.KeyRight) // Date format: Mon D
	now := time.Now()
	h.autosave(now, false)
	h.autosave(now.Add(500*time.Millisecond), false)
	var err error
	if h.settings, err = store.LoadSettings(filepath.Join(root, "settings.json")); err != nil {
		t.Fatal(err)
	}
	if h.settings.TitleFont != "narrow" || h.settings.ListShot != "title" || h.settings.DateFormat != "mon-d" {
		t.Fatalf("saved %+v", h.settings)
	}
	open()
	if h.a.TitleFont() != "narrow" || h.a.ListShot() != "title" || h.a.DateFormat() != "mon-d" {
		t.Fatal("preferences did not survive a restart")
	}
}
