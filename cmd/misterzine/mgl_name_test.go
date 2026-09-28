//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
)

func TestMGLNameFollowsTheBuild(t *testing.T) {
	defer beta.Set(false)()
	if mglPath() != "/media/fat/MisterZine.mgl" || otherMGL() != "/media/fat/MisterZine Arcade.mgl" {
		t.Fatalf("free build: %q, other %q", mglPath(), otherMGL())
	}
	beta.Set(true)
	if mglPath() != "/media/fat/MisterZine Arcade.mgl" || otherMGL() != "/media/fat/MisterZine.mgl" {
		t.Fatalf("beta build: %q, other %q", mglPath(), otherMGL())
	}
}

// Switching between the free version and the beta leaves one entry: each
// writes its own and removes the other's, a lowercase leftover included.
func TestEnsureMGLRemovesTheOtherBuildsEntry(t *testing.T) {
	dir := t.TempDir()
	free, arcade := filepath.Join(dir, "MisterZine.mgl"), filepath.Join(dir, "MisterZine Arcade.mgl")
	keep := filepath.Join(dir, "MisterZine Tools.mgl")
	for _, p := range []string{filepath.Join(dir, "misterzine.mgl"), keep} {
		if err := os.WriteFile(p, []byte(mglBody), 0644); err != nil {
			t.Fatal(err)
		}
	}
	entries := func() []string {
		ents, _ := os.ReadDir(dir)
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		return names
	}
	// the beta arrives
	if err := ensureMGLAt(arcade, free); err != nil {
		t.Fatal(err)
	}
	if got := entries(); len(got) != 2 || got[0] != "MisterZine Arcade.mgl" || got[1] != "MisterZine Tools.mgl" {
		t.Fatalf("after the beta wrote its entry: %q", got)
	}
	if b, err := os.ReadFile(arcade); err != nil || string(b) != mglBody {
		t.Fatalf("beta entry %q, %v", b, err)
	}
	// and again: nothing changes
	if err := ensureMGLAt(arcade, free); err != nil {
		t.Fatal(err)
	}
	// back to the free version
	if err := ensureMGLAt(free, arcade); err != nil {
		t.Fatal(err)
	}
	if got := entries(); len(got) != 2 || got[0] != "MisterZine Tools.mgl" || got[1] != "MisterZine.mgl" {
		t.Fatalf("after the free version wrote its entry: %q", got)
	}
}

func TestDisableLauncherRemovesBothEntries(t *testing.T) {
	dir := t.TempDir()
	startup := filepath.Join(dir, "user-startup.sh")
	free, arcade := filepath.Join(dir, "MisterZine.mgl"), filepath.Join(dir, "MisterZine Arcade.mgl")
	for _, p := range []string{free, arcade} {
		if err := os.WriteFile(p, []byte(mglBody), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := disableLauncherFiles(startup, free, arcade); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{free, arcade} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s survived disabling", filepath.Base(p))
		}
	}
}
