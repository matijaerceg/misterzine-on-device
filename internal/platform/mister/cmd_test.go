//go:build linux

package mister

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchCoreMatchesScanNames(t *testing.T) {
	card := t.TempDir()
	dir := filepath.Join(card, "_Console")
	os.MkdirAll(dir, 0755)
	for _, name := range []string{"SNES_20260101.rbf", "SNES_20260908_22baeda_DB9.rbf", "jt1942.rbf"} {
		os.WriteFile(filepath.Join(dir, name), []byte("test"), 0644)
	}
	os.Mkdir(filepath.Join(dir, "SNES_20990101.rbf"), 0755)
	for _, target := range []string{"core:SNES", "core:snes_20260908_22baeda_db9"} {
		got, err := LaunchPath(card, target)
		if err != nil || got != filepath.ToSlash(filepath.Join(dir, "SNES_20260908_22baeda_DB9.rbf")) {
			t.Fatalf("%s: %s %v", target, got, err)
		}
	}
	if _, err := LaunchPath(card, "core:jt1942"); err != nil {
		t.Fatal(err)
	}
	if _, err := LaunchPath(card, "_Console/SNES_20990101.rbf"); err == nil {
		t.Fatal("directory accepted as core")
	}
	if err := (*Cmd)(nil).Available(); err == nil {
		t.Fatal("nil interface accepted")
	}
}
