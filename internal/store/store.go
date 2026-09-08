// Package store persists the app's small JSON files atomically: settings,
// state, favorites. Every write goes to a temp file, is synced, then renamed
// over the target, so a power cut mid-write leaves the old file intact.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Settings are the user's device settings.
type Settings struct {
	Schema   int    `json:"schema"`
	Rotation string `json:"rotation"` // auto, left, right, off
	Inset    int    `json:"inset"`
	Prefetch bool   `json:"prefetch"`
	Scroll   string `json:"scroll"` // normal, fast, turbo
}

// DefaultSettings for a fresh install.
func DefaultSettings() Settings {
	return Settings{Schema: 1, Rotation: "auto", Inset: 8, Scroll: "fast"}
}

// State is what the app restores between runs.
type State struct {
	Schema   int             `json:"schema"`
	CursorK  string          `json:"cursor_k"`
	Sort     string          `json:"sort"` // updated, debut
	Filters  data.Filters    `json:"filters"`
	LastOpen string          `json:"last_open"`
	DataHash string          `json:"data_hash"`
	Seen     data.SeenRecord `json:"seen"`
}

// FavEntry is one favorite with its change time.
type FavEntry struct {
	K  string `json:"k"`
	At string `json:"at"`
}

// Favorites is a set of row keys with tombstones, so a later account merge
// is a union where the newer change wins.
type Favorites struct {
	V        int        `json:"v"`
	Favs     []FavEntry `json:"favs"`
	Removed  []FavEntry `json:"removed,omitempty"`
	SyncedAt *string    `json:"synced_at"`
}

// Set returns the live favorites as a set.
func (f *Favorites) Set() map[string]bool {
	m := map[string]bool{}
	for _, e := range f.Favs {
		m[e.K] = true
	}
	return m
}

// Apply records the set's current content, tombstoning removals.
func (f *Favorites) Apply(set map[string]bool, now time.Time) {
	f.V = 1
	stamp := now.UTC().Format(time.RFC3339)
	old := f.Set()
	var favs []FavEntry
	for _, e := range f.Favs {
		if set[e.K] {
			favs = append(favs, e)
		} else {
			f.Removed = append(f.Removed, FavEntry{K: e.K, At: stamp})
		}
	}
	for k := range set {
		if !old[k] {
			favs = append(favs, FavEntry{K: k, At: stamp})
		}
	}
	f.Favs = favs
	// drop tombstones that were re-added
	var removed []FavEntry
	for _, e := range f.Removed {
		if !set[e.K] {
			removed = append(removed, e)
		}
	}
	f.Removed = removed
}

// Load reads a JSON file into v. A missing file is reported as ErrNotExist.
func Load(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		bad := path + ".bad"
		os.Rename(path, bad)
		return fmt.Errorf("%s: %w (moved to %s)", filepath.Base(path), err, filepath.Base(bad))
	}
	return nil
}

// Save writes v as JSON atomically.
func Save(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return WriteAtomic(path, b)
}

// WriteAtomic writes bytes through a temp file and a rename.
func WriteAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil && !errors.Is(err, os.ErrInvalid) {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
