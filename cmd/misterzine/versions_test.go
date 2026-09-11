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

// The remembered version per game rides along in state.json and comes back
// on the next run.
func TestRememberedVersionsPersist(t *testing.T) {
	root := t.TempDir()
	h := favoritesHost(root)
	rows := []data.Row{{K: "game", Title: "Example", MRA: "_Arcade/Example.mra", Updated: "2026-09-10"}}
	alts := []string{"_Arcade/_alternatives/Example A.mra"}
	open := func() {
		h.a = app.New(app.Config{PhysW: 320, PhysH: 240,
			Alternatives:   func(*data.Row) []string { return alts },
			Versions:       h.state.Versions,
			VersionChanged: func() { h.dirty = true },
		}, data.Ingest(rows, "test", time.Now()), nil)
	}
	open()
	tap := func(key platform.Key) {
		now := time.Now()
		h.a.Handle(platform.Event{Key: key, Pressed: true, At: now})
		h.a.Handle(platform.Event{Key: key, At: now})
	}
	h.dirty = false
	tap(platform.KeyEnter)
	tap(platform.KeyRight)
	if !h.dirty {
		t.Fatal("choosing a version did not mark the state for saving")
	}
	now := time.Now()
	h.autosave(now, false)
	h.autosave(now.Add(500*time.Millisecond), false)
	var st store.State
	if err := store.Load(filepath.Join(root, "state.json"), &st); err != nil || st.Versions["game"] != alts[0] {
		t.Fatalf("state.json versions: %v, %v", st.Versions, err)
	}
	h.state = st
	open()
	tap(platform.KeyEnter)
	if h.a.Screen() != app.ScreenDetails {
		t.Fatal("Enter did not open Details")
	}
	if got := h.a.Versions()["game"]; got != alts[0] {
		t.Fatalf("reopened app remembers %q", got)
	}
}
