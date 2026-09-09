//go:build linux

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

func TestAutosaveWithoutNavigationOrQuit(t *testing.T) {
	for _, kind := range []string{"settings", "favorites", "state"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			h := favoritesHost(root)
			switch kind {
			case "settings":
				h.settings.Prefetch = true
				h.setDirty = true
			case "favorites":
				h.a.FavoriteSet()["saved-game"] = true
				h.favDirty = true
			case "state":
				h.dirty = true
			}
			now := time.Now()
			h.autosave(now, false)
			h.autosave(now.Add(500*time.Millisecond), false)
			switch kind {
			case "settings":
				got, err := store.LoadSettings(filepath.Join(root, "settings.json"))
				if err != nil || !got.Prefetch || got.Rotation != "auto" {
					t.Fatalf("settings not preserved: %+v, %v", got, err)
				}
			case "favorites":
				if !favoritesHost(root).a.FavoriteSet()["saved-game"] {
					t.Fatal("favorite not saved while app stayed open")
				}
			case "state":
				var got store.State
				if err := store.Load(filepath.Join(root, "state.json"), &got); err != nil || got.DataHash != "test" {
					t.Fatalf("state not saved: %+v, %v", got, err)
				}
			}
			if h.pendingSave() {
				t.Fatal("successful save still pending")
			}
		})
	}
}

func TestAutosaveRetriesFailedFilesWithoutNewInput(t *testing.T) {
	for _, file := range []string{"state.json", "settings.json", "favorites.json"} {
		t.Run(file, func(t *testing.T) {
			root := t.TempDir()
			h := favoritesHost(root)
			path := filepath.Join(root, file)
			// A directory prevents atomic replacement, including when running as root.
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatal(err)
			}
			h.dirty, h.setDirty, h.favDirty = true, true, true
			h.a.FavoriteSet()["saved-game"] = true
			now := time.Now()
			h.autosave(now, false)
			h.autosave(now.Add(500*time.Millisecond), false)
			if !h.pendingSave() {
				t.Fatal("failed save forgotten")
			}
			if (h.dirty != (file == "state.json")) || (h.setDirty != (file == "settings.json")) || (h.favDirty != (file == "favorites.json")) {
				t.Fatal("successful files must not remain dirty")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			h.autosave(now.Add(time.Second), false)
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("retry happened too soon")
			}
			h.autosave(now.Add(5500*time.Millisecond), false)
			if h.pendingSave() {
				t.Fatal("retry did not save")
			}
			var got any
			if err := store.Load(path, &got); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRotationIntentSurvivesSaves(t *testing.T) {
	for _, rotation := range []gfx.Rotation{gfx.RotNone, gfx.RotRight, gfx.RotLeft} {
		t.Run(string(rune('0'+rotation)), func(t *testing.T) {
			root := t.TempDir()
			h := favoritesHost(root)
			h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Rotation: rotation,
				SettingsChanged: func() { h.setDirty = true },
				Action: func(kind, arg string) {
					if kind == "rotation" {
						h.settings.Rotation = arg
					}
				},
			}, data.Ingest(nil, "test", time.Now()), nil)
			h.saveAll(true) // first save, with no existing settings file
			check := func(want string) {
				t.Helper()
				got, err := store.LoadSettings(filepath.Join(root, "settings.json"))
				if err != nil || got.Rotation != want {
					t.Fatalf("rotation=%q want %q: %v", got.Rotation, want, err)
				}
			}
			check("auto")
			now := time.Now()
			tap := func(k platform.Key) {
				now = now.Add(time.Second)
				h.a.Handle(platform.Event{Key: k, Pressed: true, At: now})
				h.a.Handle(platform.Event{Key: k, At: now.Add(time.Millisecond)})
			}
			tap(platform.KeyBack) // Options
			tap(platform.KeyDown)
			tap(platform.KeyDown) // Scroll speed
			tap(platform.KeyRight)
			if !h.setDirty {
				t.Fatal("settings input did not reach callback")
			}
			h.saveAll(false)
			check("auto")
			tap(platform.KeyHome) // Rotation
			if rotation == gfx.RotRight {
				tap(platform.KeyLeft)
			} else {
				tap(platform.KeyRight)
			}
			h.saveAll(false)
			if rotation == gfx.RotNone {
				check("right")
			} else {
				check("off")
			}
		})
	}
}

func TestFiltersAutosaveAndRestore(t *testing.T) {
	root := t.TempDir()
	h := favoritesHost(root)
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, FiltersChanged: func() { h.dirty = true }}, data.Ingest([]data.Row{{K: "game"}}, "test", time.Now()), nil)
	choices := data.Filters{Install: data.InstallFound, FavOnly: true, Since: true, BaseOff: map[string]bool{"Console": true}, SrcOff: map[string]bool{"other": true}, RotOff: map[string]bool{"v": true}, PlrOff: map[string]bool{"4": true}, GenreOff: map[string]bool{"Racing": true}, DirectionsOff: map[string]bool{"4": true}, ButtonsOff: map[string]bool{"6": true}}
	h.a.SetFilters(choices)
	now := time.Now()
	h.autosave(now, false)
	h.autosave(now.Add(time.Second), false)
	var state store.State
	if err := store.Load(filepath.Join(root, "state.json"), &state); err != nil {
		t.Fatal(err)
	}
	reopened := app.New(app.Config{PhysW: 320, PhysH: 240}, h.a.Data(), &state.Seen)
	reopened.SetFilters(state.Filters)
	if !reflect.DeepEqual(reopened.Filters(), choices) {
		t.Fatalf("lost filter choices: %+v", reopened.Filters())
	}
	h.a.SetFilters(data.Filters{})
	h.saveAll(false)
	state = store.State{}
	if err := store.Load(filepath.Join(root, "state.json"), &state); err != nil || state.Filters.Active() {
		t.Fatalf("cleared filters returned: %+v %v", state.Filters, err)
	}
}
