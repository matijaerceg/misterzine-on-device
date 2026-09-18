package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestRenameKeys(t *testing.T) {
	favs, versions, recents := 0, 0, 0
	a := New(Config{PhysW: 320, PhysH: 240,
		Favorites:      map[string]bool{"local:orphan": true, "colony7": true},
		Versions:       map[string]string{"local:orphan": "_Arcade/_Extra/Orphan (set 2).mra", "local:kept": "x.mra", "taken": "old.mra"},
		RecentLaunches: []data.Recent{{K: "taken", At: "2026-09-18T10:00:00Z"}, {K: "local:orphan", At: "2026-09-17T10:00:00Z"}, {K: "local:kept", At: "2026-09-16T10:00:00Z"}},
		FavChanged:     func() { favs++ }, VersionChanged: func() { versions++ }, RecentsChanged: func() { recents++ },
	}, data.Ingest(localTestRows(), "h", time.Now()), nil)
	a.RenameKeys(map[string]string{"local:orphan": "orphan", "local:taken": "taken", "local:none": "none"})
	if !reflect.DeepEqual(a.cfg.Favorites, map[string]bool{"orphan": true, "colony7": true}) {
		t.Fatalf("favorites %v", a.cfg.Favorites)
	}
	if !reflect.DeepEqual(a.cfg.Versions, map[string]string{"orphan": "_Arcade/_Extra/Orphan (set 2).mra", "local:kept": "x.mra", "taken": "old.mra"}) {
		t.Fatalf("versions %v", a.cfg.Versions)
	}
	if got := a.cfg.RecentLaunches; len(got) != 3 || got[1].K != "orphan" || got[0].K != "taken" {
		t.Fatalf("recents %v", got)
	}
	if favs != 1 || versions != 1 || recents != 1 {
		t.Fatalf("change hooks favs=%d versions=%d recents=%d", favs, versions, recents)
	}
	// An existing entry under the new key is kept; the old key still goes.
	a.cfg.Versions["local:kept"] = "x.mra"
	a.cfg.Versions["kept"] = "y.mra"
	a.cfg.RecentLaunches = []data.Recent{{K: "kept", At: "2026-09-18T10:00:00Z"}, {K: "local:kept", At: "2026-09-16T10:00:00Z"}}
	a.RenameKeys(map[string]string{"local:kept": "kept"})
	if a.cfg.Versions["kept"] != "y.mra" || a.cfg.Versions["local:kept"] != "" || len(a.cfg.RecentLaunches) != 1 || a.cfg.RecentLaunches[0].At != "2026-09-18T10:00:00Z" {
		t.Fatalf("existing entries lost: %v %v", a.cfg.Versions, a.cfg.RecentLaunches)
	}
	a.RenameKeys(nil) // no-op
}
