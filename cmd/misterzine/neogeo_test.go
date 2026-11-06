//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
)

func TestNeoGeoLaunchRevalidatesShortcut(t *testing.T) {
	card := t.TempDir()
	os.MkdirAll(filepath.Join(card, "_Console"), 0755)
	os.WriteFile(filepath.Join(card, "_Console", "NeoGeo_20260101.rbf"), []byte("fixture"), 0644)
	target := filepath.Join(card, "game.mgl")
	for _, tc := range []struct {
		name, core, file string
		pass             bool
	}{
		{"cartridge", "_Console/NeoGeo", "mslug.neo", true},
		{"expanded catalogue", "_Console/NeoGeo", "garou.neo", true},
		{"parenthesized set", "_Console/NeoGeo", "3 Count Bout (3countb).neo", true},
		{"region only", "_Console/NeoGeo", "3 Count Bout (USA).neo", false},
		{"regional set", "_Console/NeoGeo", "kof97h.zip", true},
		{"MiSTer alias", "_Console/NeoGeo", "tws96.rom", true},
		{"different system", "_Console/SNES", "mslug.neo", false},
		{"unreviewed title", "_Console/NeoGeo", "homebrew.neo", false},
		{"wrong core location", "_Other/NeoGeo", "mslug.neo", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			xml := `<mistergamedescription><rbf>` + tc.core + `</rbf><file type="f" index="1" path="` + tc.file + `"/></mistergamedescription>`
			if err := os.WriteFile(target, []byte(xml), 0644); err != nil {
				t.Fatal(err)
			}
			if err := validateNeoGeoLaunch(card, "game.mgl"); (err == nil) != tc.pass {
				t.Fatalf("got %v, pass=%v", err, tc.pass)
			}
		})
	}
	os.WriteFile(target, []byte(`<mistergamedescription><rbf>_Console/NeoGeo</rbf><file type="s" index="1" path="mslug.chd"/></mistergamedescription>`), 0644)
	if err := validateNeoGeoLaunch(card, "game.mgl"); err == nil {
		t.Fatal("accepted CD")
	}
	if err := validateNeoGeoLaunch(card, "../game.mgl"); err == nil {
		t.Fatal("accepted escape")
	}
}

func TestNeoGeoDeviceScanReport(t *testing.T) {
	h := &host{root: t.TempDir(), card: "../../testdata/mgl/card"}
	result, err := h.scanNeoGeo()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 5 {
		t.Fatalf("want five rows, got %d", len(result.Rows))
	}
	b, err := os.ReadFile(filepath.Join(h.root, "mgl-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved mgl.Result
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Rows) != 5 || len(saved.Entries) != 9 {
		t.Fatalf("unexpected report: %d rows, %d entries", len(saved.Rows), len(saved.Entries))
	}
}

func writeNeoGeoShortcut(t *testing.T, card, rel, core string) {
	t.Helper()
	p := filepath.Join(card, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	xml := `<mistergamedescription><rbf>` + core + `</rbf><file type="f" index="1" path="mslug.neo"/></mistergamedescription>`
	if err := os.WriteFile(p, []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
}

// Exercise the ordinary host's full scan, cache and rescan path without the
// test-build tag. MRA games and Neo Geo shortcuts must coexist.
func TestNeoGeoInNormalCardScanAndCache(t *testing.T) {
	h := backgroundHost(t)
	orphanCard(t, h)
	os.MkdirAll(filepath.Join(h.card, "_Console"), 0755)
	core := filepath.Join(h.card, "_Console", "NeoGeo_20260101.rbf")
	if err := os.WriteFile(core, []byte("fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	target := "_@Favorites/Metal Slug.mgl"
	duplicate := "_@Favorites/_Copies/Metal Slug.mgl"
	writeNeoGeoShortcut(t, h.card, target, "_Console/NeoGeo")
	writeNeoGeoShortcut(t, h.card, duplicate, "_Console/NeoGeo")
	writeNeoGeoShortcut(t, h.card, "_@Favorites/Other system.mgl", "_Console/SNES")
	scanOnce(t, h, catalogueRows(), "cat")
	ds := h.a.Data()
	key := ""
	for i, r := range ds.Rows {
		if r.IsNeoGeoShortcut() {
			if key != "" || r.Title != "Metal Slug" || h.status[i] != data.StatusFoundUndated {
				t.Fatalf("unexpected imported row: %+v, status %v", r, h.status[i])
			}
			key = r.K
			if len(h.alts[key]) != 1 || ds.Der[i].CoreGroup != "Neo Geo" {
				t.Fatalf("shortcut alternatives/group missing: %v %+v", h.alts[key], ds.Der[i])
			}
		}
	}
	if key == "" || len(ds.Rows) != 4 {
		t.Fatalf("normal scan did not keep both kinds of local game: %+v", ds.Rows)
	}
	reopened, count, note := restart(t, h, catalogueRows(), "cat", map[string]bool{key: true})
	if count != 2 || !strings.Contains(note, "2 local rows") || !reopened.a.FavoriteSet()[key] {
		t.Fatalf("startup lost local rows or favorite: %d %q %v", count, note, reopened.a.FavoriteSet())
	}
	if err := os.Remove(core); err != nil {
		t.Fatal(err)
	}
	reopened.requestScan()
	finishBackground(t, reopened)
	for i, r := range reopened.a.Data().Rows {
		if r.K == key && reopened.status[i] != data.StatusNotFound {
			t.Fatal("missing Neo Geo core still shown as installed")
		}
	}
	os.Remove(filepath.Join(h.card, target))
	os.Remove(filepath.Join(h.card, duplicate))
	reopened.requestScan()
	finishBackground(t, reopened)
	for _, r := range reopened.a.Data().Rows {
		if r.IsNeoGeoShortcut() {
			t.Fatal("removed shortcut survived a complete rescan")
		}
	}
}

func TestNeoGeoNormalLaunchRejectsEditedShortcut(t *testing.T) {
	h := favoritesHost(t.TempDir())
	h.card = t.TempDir()
	h.quit = make(chan struct{})
	var output bytes.Buffer
	h.lg = log.New(&output, "", 0)
	writeNeoGeoShortcut(t, h.card, "game.mgl", "_Console/SNES")
	h.requestLaunch("game.mgl")
	if h.launch != "" || strings.Contains(output.String(), "command interface unavailable") || !strings.Contains(output.String(), "Neo Geo") {
		t.Fatalf("normal launch skipped shortcut validation: %q", output.String())
	}
	select {
	case <-h.quit:
		t.Fatal("invalid shortcut closed the app")
	default:
	}
}
