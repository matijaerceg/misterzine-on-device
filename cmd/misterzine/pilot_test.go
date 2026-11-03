//go:build linux

package main

import (
	"encoding/json"
	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
	"os"
	"path/filepath"
	"testing"
)

func TestPilotLaunchRevalidatesShortcut(t *testing.T) {
	card := t.TempDir()
	os.MkdirAll(filepath.Join(card, "_Console"), 0755)
	os.WriteFile(filepath.Join(card, "_Console", "NeoGeo_20260101.rbf"), []byte("fixture"), 0644)
	target := filepath.Join(card, "game.mgl")
	for _, tc := range []struct {
		name, core, file string
		pass             bool
	}{
		{"cartridge", "_Console/NeoGeo", "mslug.neo", true},
		{"different system", "_Console/SNES", "mslug.neo", false},
		{"unreviewed title", "_Console/NeoGeo", "homebrew.neo", false},
		{"wrong core location", "_Other/NeoGeo", "mslug.neo", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			xml := `<mistergamedescription><rbf>` + tc.core + `</rbf><file type="f" index="1" path="` + tc.file + `"/></mistergamedescription>`
			if err := os.WriteFile(target, []byte(xml), 0644); err != nil {
				t.Fatal(err)
			}
			if err := validatePilotLaunch(card, "game.mgl"); (err == nil) != tc.pass {
				t.Fatalf("got %v, pass=%v", err, tc.pass)
			}
		})
	}
	os.WriteFile(target, []byte(`<mistergamedescription><rbf>_Console/NeoGeo</rbf><file type="s" index="1" path="mslug.chd"/></mistergamedescription>`), 0644)
	if err := validatePilotLaunch(card, "game.mgl"); err == nil {
		t.Fatal("accepted CD")
	}
	if err := validatePilotLaunch(card, "../game.mgl"); err == nil {
		t.Fatal("accepted escape")
	}
}

func TestPilotDeviceScanReport(t *testing.T) {
	h := &host{root: t.TempDir(), card: "../../testdata/mgl/card"}
	result, err := h.scanPilot()
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
