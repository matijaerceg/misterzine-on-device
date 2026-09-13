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
	Schema         int           `json:"schema"`
	Rotation       string        `json:"rotation"` // auto, left, right, off
	FollowRotation bool          `json:"follow_ini_rotation"`
	FilterRotation bool          `json:"filter_ini_rotation"`
	InstalledOnly  bool          `json:"installed_sources_only"` // Options -> Sources: installed only
	Inset          int           `json:"inset"`
	Prefetch       bool          `json:"prefetch"`
	Scroll         string        `json:"scroll"` // rows per second: 20, 30, 60
	HoldDelay      int           `json:"hold_delay_ms"`
	Screensaver    string        `json:"screensaver_minutes"`
	RememberSort   bool          `json:"remember_sort"`
	LastSort       data.SortMode `json:"last_sort"`
	ViewsOff       []string      `json:"views_off"` // Options -> Views: the list orders left out of the Y cycle, by name (updated, debut, year, alphabetical, maker, favorites, recents)
	// OpenAtBoot and ReturnAfterGame are carried out by the resident menu
	// launcher (misterzine launcher watch), which reads this file itself.
	OpenAtBoot      bool   `json:"open_at_boot"`      // Options -> Open at boot: pick the menu entry once Main's menu is up
	ReturnAfterGame bool   `json:"return_after_game"` // Options -> Return after game: pick it again when a launched game exits to Menu
	TitleFont       string `json:"title_font"`        // list titles: tall (default), narrow or normal
	ListShot        string `json:"list_shot"`         // list thumbnail: gameplay (default) or title
	DateFormat      string `json:"date_format"`       // list dates: mm-dd (default), dd-mm, mon-d, d-mon, yymmdd
	ListLayout      string `json:"list_layout"`       // main view: list (default), split or picture
	ButtonLabels    string `json:"button_labels"`     // legend button names: mister (default), xbox, playstation or numbers
	Canvas          string `json:"canvas"`            // Options -> Canvas: fit (default: sized to the display's integer scale) or 320x240
	MenuButton      string `json:"menu_button"`       // Options -> Menu button: options (default) or leave
	InsetX          int    `json:"inset_x"`
	InsetY          int    `json:"inset_y"`
}

// DefaultSettings for a fresh install.
func DefaultSettings() Settings {
	return Settings{Schema: 1, Rotation: "auto", FollowRotation: true, Inset: 15, InsetX: 15, InsetY: 15, Scroll: "30", HoldDelay: 300, Screensaver: "1", RememberSort: true, ViewsOff: []string{"recents"}}
}

// LoadSettings reads path over the defaults and migrates older files.
func LoadSettings(path string) (Settings, error) {
	s := DefaultSettings()
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		bad := path + ".bad"
		os.Rename(path, bad)
		return DefaultSettings(), fmt.Errorf("%s: %w (moved to %s)", filepath.Base(path), err, filepath.Base(bad))
	}
	// a file from before the two-axis inset carries only "inset"
	var probe struct {
		X       *int `json:"inset_x"`
		Y       *int `json:"inset_y"`
		Recents bool `json:"recents_view"` // before views_off: Options -> Recents view
	}
	json.Unmarshal(b, &probe)
	// Only a file with no views_off key at all predates Options -> Views;
	// a null there (an older build saved an empty set that way) means every
	// view is on, and must not switch Recents back off.
	var keys map[string]json.RawMessage
	json.Unmarshal(b, &keys)
	if _, has := keys["views_off"]; !has {
		// a file from before Options -> Views: Recents was the one optional view
		s.ViewsOff = []string{}
		if !probe.Recents {
			s.ViewsOff = []string{"recents"}
		}
	}
	s.Migrate(probe.X == nil && probe.Y == nil)
	return s, nil
}

// Migrate brings an older settings file up to date: the single inset
// becomes two (when legacy says the file predates the split), and the
// speed adjectives become rows per second.
func (s *Settings) Migrate(legacy bool) {
	if !s.LastSort.Valid() {
		s.LastSort = data.SortUpdated
	}
	views := []string{} // never nil: an empty set saves as [] and reloads as every view on
	for _, n := range s.ViewsOff {
		if _, ok := data.ParseSort(n); ok {
			views = append(views, n)
		}
	}
	if len(views) >= len(data.ViewOrder) {
		views = []string{} // nothing left on: back to every view
	}
	s.ViewsOff = views
	switch s.Screensaver {
	case "off", "1", "2", "5", "10":
	default:
		s.Screensaver = "1"
	}
	switch s.HoldDelay {
	case 200, 300, 500:
	default:
		s.HoldDelay = 300
	}
	if legacy {
		s.InsetX, s.InsetY = s.Inset, s.Inset
	}
	switch s.Scroll {
	case "normal":
		s.Scroll = "20"
	case "fast", "":
		s.Scroll = "30"
	case "turbo":
		s.Scroll = "60"
	}
}

// State is what the app restores between runs.
type State struct {
	Schema   int             `json:"schema"`
	LastOpen string          `json:"last_open"`
	DataHash string          `json:"data_hash"`
	Seen     data.SeenRecord `json:"seen"`
	Filters  data.Filters    `json:"filters"`
	// Versions is the last chosen version per row key, a card-relative path.
	Versions map[string]string `json:"versions,omitempty"`
	// Recents is the launch history, newest first, kept whether or not the
	// Recents view is on.
	Recents []data.Recent `json:"recents,omitempty"`
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

// LoadFavorites preserves the original file on every error. Unlike cached
// data, favorites cannot be downloaded again, so do not rename a bad file
// and let the next launch mistake it for a fresh installation.
func LoadFavorites(path string) (Favorites, error) {
	var f Favorites
	b, err := os.ReadFile(path)
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return Favorites{}, fmt.Errorf("%s: %w (file kept unchanged)", filepath.Base(path), err)
	}
	return f, nil
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
