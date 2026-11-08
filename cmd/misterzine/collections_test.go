//go:build linux

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Exercise startup, repeated rescans, the persisted version choice and Start
// on the same host. No files need to move into the distribution's folders.
func TestIsolatedArcadeStartAndRescan(t *testing.T) {
	h := backgroundHost(t)
	row := catalogueRows()[0]
	row.Src = "ongo"
	isolated := "_Ongo/" + row.MRA
	alt := "_Ongo/_Arcade/_alternatives/_Colony 7/Set 2.mra"
	core := "_Ongo/cores/Defender_20260801.rbf"
	stockCore := "_Arcade/cores/Defender_20240101.rbf"
	writeMRA(t, h.card, isolated, row.Title, row.SN, row.Core)
	writeMRA(t, h.card, alt, row.Title, "colony7a", row.Core)
	row.Family, row.FamilySets = row.SN, []string{"colony7a"}
	putCartFile(t, h.card, core, "isolated build")
	putCartFile(t, h.card, "downloader.ini", "[MiSTer]\n")
	putCartFile(t, h.card, "downloader_ongo_isolated.ini", "[Videodr0me/MiSTer_Ongo_Isolated]\n")
	versions := map[string]string{}
	launched := ""
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, InstalledOnly: true,
		Alternatives: h.alternatives, Versions: versions, Favorites: map[string]bool{row.K: true},
		Exists: func(p string) bool { _, err := os.Stat(filepath.Join(h.card, p)); return err == nil },
		Status: func(i int) data.Status {
			if i < len(h.status) {
				return h.status[i]
			}
			return data.StatusUnknown
		},
		VersionCore: func(p string) (string, bool) { return h.index.VersionCore(p) },
		Launch: func(p string) {
			var err error
			launched, err = h.prepareLaunch(p)
			if err != nil {
				t.Fatal(err)
			}
		},
	}, data.Ingest([]data.Row{row}, "catalogue", time.Now()), nil)
	rescan := func(want string, status data.Status) {
		t.Helper()
		h.requestScan()
		finishBackground(t, h)
		ds := h.a.Data()
		if len(ds.Rows) != 1 || ds.Rows[0].K != row.K || ds.Rows[0].MRA != want || h.status[0] != status ||
			h.a.CursorKey() != row.K || !h.a.FavoriteSet()[row.K] {
			t.Fatalf("rescan: rows %v, paths %s, statuses %v, cursor %q", len(ds.Rows), ds.Rows[0].MRA, h.status, h.a.CursorKey())
		}
	}
	now := time.Now()
	start := func(want string) {
		t.Helper()
		launched = ""
		now = now.Add(time.Second)
		h.a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: now})
		h.a.Handle(platform.Event{Key: platform.KeyStart, At: now.Add(time.Millisecond)})
		if want != "" {
			want = filepath.Join(h.card, want)
		}
		if launched != want {
			t.Fatalf("Start sent %q, want %q", launched, want)
		}
	}
	remove := func(p string) {
		t.Helper()
		if err := os.Remove(filepath.Join(h.card, p)); err != nil {
			t.Fatal(err)
		}
	}
	rescan(isolated, data.StatusCurrent)
	start(isolated)
	if !slices.Contains(h.alternatives(&h.a.Data().Rows[0]), alt) {
		t.Fatal("isolated alternative missing")
	}
	writeMRA(t, h.card, row.MRA, row.Title, row.SN, row.Core)
	putCartFile(t, h.card, stockCore, "stock build")
	rescan(row.MRA, data.StatusOutdated)
	start(row.MRA)
	versions[row.K] = isolated
	start(isolated)
	remove(core)
	if _, err := h.prepareLaunch(isolated); err == nil || !strings.Contains(err.Error(), "_Ongo/cores") {
		t.Fatalf("removed core before rescan: %v", err)
	}
	rescan(row.MRA, data.StatusOutdated)
	if name, ok := h.index.VersionCore(isolated); ok || name != row.Core || !slices.Contains(h.alternatives(&h.a.Data().Rows[0]), isolated) {
		t.Fatal("broken isolated version lost its diagnosis")
	}
	start(row.MRA)
	putCartFile(t, h.card, core, "restored")
	remove(stockCore)
	rescan(isolated, data.StatusCurrent)
	start(isolated)
	remove(core)
	rescan(row.MRA, data.StatusNotFound)
	start("")
	remove(row.MRA)
	rescan(isolated, data.StatusNotFound)
	if name, ok := h.index.VersionCore(isolated); ok || name != row.Core {
		t.Fatal("launcher exists but isolated core is missing")
	}
	if err := os.RemoveAll(filepath.Join(h.card, "_Ongo")); err != nil {
		t.Fatal(err)
	}
	rescan(row.MRA, data.StatusNotFound)
}
