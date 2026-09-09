//go:build linux

package main

import (
	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"io"
	"log"
	"path/filepath"
	"testing"
	"time"
)

func TestServerClockSurvivesSystemJump(t *testing.T) {
	server := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	s := serverClockSample{wall: server, local: time.Unix(100, 0)}
	for _, tc := range []struct {
		name   string
		system time.Time
		caught bool
	}{
		{"unset", time.Unix(110, 0), false},
		{"ntp", server.Add(10 * time.Second), true},
		{"bad jump", server.Add(30 * 365 * 24 * time.Hour), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, caught := s.at(tc.system, 10*time.Second)
			if !got.Equal(server.Add(10*time.Second)) || caught != tc.caught {
				t.Fatalf("got %v, caught=%v", got, caught)
			}
		})
	}
}

func TestRecoveredClockPersistsWithoutInput(t *testing.T) {
	h := &host{root: t.TempDir(), lg: log.New(io.Discard, "", 0)}
	corrected := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	h.timeSample.Store(&serverClockSample{wall: corrected, local: time.Now()})
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Now: h.now, TimerNow: time.Now}, data.Ingest(nil, "", corrected), nil)
	h.trustClock()
	if !h.dirty {
		t.Fatal("clock recovery not marked for autosave")
	}
	now := time.Now()
	h.autosave(now, false)
	h.autosave(now.Add(time.Second), false)
	var state store.State
	if err := store.Load(filepath.Join(h.root, "state.json"), &state); err != nil {
		t.Fatal(err)
	}
	stamp, err := time.Parse(time.RFC3339, state.Seen.T)
	if err != nil || stamp.Sub(corrected) < 0 || stamp.Sub(corrected) > time.Minute {
		t.Fatalf("wrong persisted visit: %q", state.Seen.T)
	}
	if state.LastOpen != state.Seen.T {
		t.Fatalf("last open ignored corrected date: %q / %q", state.LastOpen, state.Seen.T)
	}
	if h.dirty {
		t.Fatal("state still dirty after successful save")
	}
}
