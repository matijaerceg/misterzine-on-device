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

// A core filed in a folder MiSTer's menu opens (_Computer/_Other) launches
// from the list as it does from the menu.
func TestLaunchCoreInSubfolder(t *testing.T) {
	card := t.TempDir()
	dir := filepath.Join(card, "_Computer", "_Other")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "Aquarius_20260603.rbf"), []byte("test"), 0644)
	got, err := LaunchPath(card, "core:Aquarius")
	if err != nil || got != filepath.ToSlash(filepath.Join(dir, "Aquarius_20260603.rbf")) {
		t.Fatalf("core:Aquarius: %s %v", got, err)
	}
}

// A name ending in a dot puts ".." before the extension (issue #12).
func TestLaunchPathDottedNames(t *testing.T) {
	card := t.TempDir()
	os.MkdirAll(filepath.Join(card, "_Arcade", "_MeatCores"), 0755)
	for _, target := range []string{"_Arcade/S.O.S..mra", "_Arcade/_MeatCores/Dragon Ball Z V.R. V.S..mra"} {
		os.WriteFile(filepath.Join(card, filepath.FromSlash(target)), []byte("<misterromdescription/>"), 0644)
		got, err := LaunchPath(card, target)
		if err != nil || got != filepath.ToSlash(filepath.Join(card, filepath.FromSlash(target))) {
			t.Fatalf("%s: %s %v", target, got, err)
		}
		if _, err := LaunchPath(card, filepath.ToSlash(filepath.Join(card, filepath.FromSlash(target)))); err != nil {
			t.Fatalf("absolute %s: %v", target, err)
		}
	}
	os.WriteFile(filepath.Join(card, "x.mra"), []byte("x"), 0644)
	for _, target := range []string{"_Arcade/../x.mra", "../x.mra", "_Arcade/..", card + "/_Arcade/../x.mra"} {
		if _, err := LaunchPath(card, target); err == nil {
			t.Fatalf("%s accepted", target)
		}
	}
}
