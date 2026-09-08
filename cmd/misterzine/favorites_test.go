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
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

func favoritesHost(root string) *host {
	h := &host{root: root, lg: log.New(io.Discard, "", 0), settings: store.DefaultSettings()}
	h.loadFavorites()
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Favorites: h.favs.Set(), FavoritesUnavailable: h.favLoadFailed}, data.Ingest(nil, "test", time.Now()), nil)
	return h
}

func TestUnreadableFavoritesSurviveRepeatedLaunches(t *testing.T) {
	for _, original := range []string{
		`{"v":1,"favs":[{"k":"saved-game"`,
		`{"v":1,"favs":[{"k":"saved-game"}],"removed":"wrong type"}`,
	} {
		t.Run(original, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "favorites.json")
			if err := os.WriteFile(path, []byte(original), 0644); err != nil {
				t.Fatal(err)
			}
			for launch := 0; launch < 2; launch++ {
				h := favoritesHost(root)
				if !h.favLoadFailed || len(h.favs.Set()) != 0 {
					t.Fatal("unreadable favorites must not become an editable partial set")
				}
				// Neither an autosave nor a final quit/launch save may overwrite it,
				// even if a caller incorrectly marks favorites dirty.
				h.favDirty = true
				h.saveAll(false)
				h.saveAll(true)
				got, err := os.ReadFile(path)
				if err != nil || string(got) != original {
					t.Fatalf("launch %d changed the unreadable file: %q, %v", launch, got, err)
				}
				if _, err := os.Stat(path + ".bad"); !os.IsNotExist(err) {
					t.Fatal("favorites were renamed instead of being preserved in place")
				}
			}
		})
	}
}

func TestFavoritesReadFailureDoesNotOverwriteRecoveredFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "favorites.json")
	// A directory reliably produces a read error even when the tests run
	// as root on a MiSTer. Restore a readable file before the final save to
	// model a temporary read error followed by a successful possible write.
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	h := favoritesHost(root)
	if !h.favLoadFailed {
		t.Fatal("read error was treated as a fresh install")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"v":1,"favs":[{"k":"saved-game","at":"2026-09-08T00:00:00Z"}]}`)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	h.saveAll(true)
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("overwrote favorites after a read error: %q, %v", got, err)
	}
	if next := favoritesHost(root); next.favLoadFailed || !next.a.FavoriteSet()["saved-game"] {
		t.Fatal("a later launch did not recover the now-readable favorites")
	}
}

func TestFavoritesPersistOnNewAndExistingInstalls(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "new"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "favorites.json")
			if existing {
				if err := os.WriteFile(path, []byte(`{"v":1,"favs":[{"k":"old","at":"2026-09-01T00:00:00Z"}]}`), 0644); err != nil {
					t.Fatal(err)
				}
			}
			h := favoritesHost(root)
			if h.favLoadFailed {
				t.Fatal("favorites incorrectly disabled")
			}
			h.a.FavoriteSet()["new"] = true
			h.favDirty = true
			h.saveAll(false)
			next := favoritesHost(root)
			if !next.a.FavoriteSet()["new"] || next.a.FavoriteSet()["old"] != existing {
				t.Fatal("favorite addition or an existing favorite was lost")
			}
			delete(next.a.FavoriteSet(), "new")
			next.favDirty = true
			next.saveAll(true)
			last := favoritesHost(root)
			if last.a.FavoriteSet()["new"] || last.a.FavoriteSet()["old"] != existing {
				t.Fatal("favorite removal was not persisted")
			}
			if len(last.favs.Removed) != 1 || last.favs.Removed[0].K != "new" {
				t.Fatal("favorite removal lost its sync tombstone")
			}
		})
	}
}
