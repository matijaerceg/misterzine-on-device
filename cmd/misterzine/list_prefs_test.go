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
	rows := []data.Row{{Base: "Arcade", K: "a", Title: "Alpha", Updated: "2026-09-10"}}
	open := func() {
		h.a = app.New(app.Config{PhysW: 320, PhysH: 240, RememberSort: h.settings.RememberSort,
			TitleFont: h.settings.TitleFont, ListShot: h.settings.ListShot, DateFormat: h.settings.DateFormat,
			SettingsChanged: func() { h.setDirty = true },
		}, data.Ingest(rows, "test", time.Now()), nil)
	}
	open()
	if h.a.TitleFont() != "tall" || h.a.ListShot() != "gameplay" || h.a.DateFormat() != "mm-dd" {
		t.Fatal("fresh defaults: narrow tall titles, gameplay shots, MM-DD")
	}
	at := time.Now()
	tap := func(key platform.Key) {
		at = at.Add(50 * time.Millisecond) // taps spaced past the bounce guard
		h.a.Handle(platform.Event{Key: key, Pressed: true, At: at})
		h.a.Handle(platform.Event{Key: key, At: at})
	}
	tap(platform.KeyBack)
	for i := 0; i < 12; i++ {
		tap(platform.KeyDown)
	}
	tap(platform.KeyLeft) // Title font: narrow, in the List group after Recents view
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

func TestArcadePreferenceAndIntroPersistThroughHost(t *testing.T) {
	h := favoritesHost(t.TempDir())
	h.settings.ArcadeIntroPending = true
	now := time.Now()
	open := func() {
		h.a = app.New(app.Config{PhysW: 320, PhysH: 240, ShowNonArcade: h.settings.ShowNonArcade, ArcadeIntro: h.settings.ArcadeIntroPending, SettingsChanged: func() { h.setDirty = true }}, data.Ingest([]data.Row{{K: "s", Title: "System", Base: "Console"}}, "", now), nil)
	}
	tap := func(k platform.Key) {
		now = now.Add(time.Second)
		h.a.Handle(platform.Event{Key: k, Pressed: true, At: now})
		h.a.Handle(platform.Event{Key: k, At: now.Add(time.Millisecond)})
	}
	reopen := func() {
		t.Helper()
		h.saveAll(true)
		var err error
		h.settings, err = store.LoadSettings(filepath.Join(h.root, "settings.json"))
		if err != nil {
			t.Fatal(err)
		}
		open()
	}
	open()
	reopen()
	if !h.a.ArcadeIntroPending() {
		t.Fatal("saving before acknowledgement lost explanation")
	}
	tap(platform.KeyBack)
	reopen()
	if h.a.ArcadeIntroPending() || h.a.ShowNonArcade() {
		t.Fatal("acknowledgement did not persist")
	}
	for _, show := range []bool{true, false} {
		tap(platform.KeyBack)
		for i := 0; i < 6; i++ {
			tap(platform.KeyDown)
		}
		if show {
			tap(platform.KeyRight)
		} else {
			tap(platform.KeyLeft)
		}
		reopen()
		if h.a.ShowNonArcade() != show || h.a.ArcadeIntroPending() {
			t.Fatal("preference did not survive restart")
		}
		if (h.a.CursorKey() == "s") != show {
			t.Fatal("restored catalogue does not follow preference")
		}
	}
}
