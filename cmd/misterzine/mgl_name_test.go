//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
)

// The main-menu entry is MisterZine Arcade in Stable and MisterZine Arcade
// BETA in the members' Beta, and its setname stays misterzine in both, which
// CORENAME, the INI's [MisterZine] section and the watcher go by.
func TestMenuEntryNamesTheBuild(t *testing.T) {
	for _, tc := range []struct {
		beta          bool
		entry, others string
	}{
		{false, "/media/fat/MisterZine Arcade.mgl", "/media/fat/MisterZine Arcade BETA.mgl /media/fat/MisterZine.mgl"},
		{true, "/media/fat/MisterZine Arcade BETA.mgl", "/media/fat/MisterZine Arcade.mgl /media/fat/MisterZine.mgl"},
	} {
		restore := beta.Set(tc.beta)
		sent := ""
		openMenuEntry(func(line string) error { sent = line; return nil })
		entry, others := menuMGL(), strings.Join(otherMGLs(), " ")
		restore()
		if entry != tc.entry || others != tc.others {
			t.Fatalf("beta %v: entry %q, others %q", tc.beta, entry, others)
		}
		if sent != "load_core "+tc.entry {
			t.Fatalf("beta %v: Open at boot and Return after game send %q", tc.beta, sent)
		}
	}
	if mglBody != "<mistergamedescription>\n\t<rbf>menu</rbf>\n\t<setname>misterzine</setname>\n</mistergamedescription>\n" {
		t.Fatalf("the entry's setname changed: %q", mglBody)
	}
}

// A card that moves between Stable and Beta shows one entry: each build's
// writes its own name and removes the other's, whichever way round.
func TestEachBuildReplacesTheOthersEntry(t *testing.T) {
	dir := t.TempDir()
	stable, betaEntry, legacy := filepath.Join(dir, "MisterZine Arcade.mgl"), filepath.Join(dir, "MisterZine Arcade BETA.mgl"), filepath.Join(dir, "MisterZine.mgl")
	if err := os.WriteFile(stable, []byte(mglBody), 0644); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		entry  string
		others []string
		want   string
	}{
		{betaEntry, []string{stable, legacy}, "MisterZine Arcade BETA.mgl"}, // to Beta
		{stable, []string{betaEntry, legacy}, "MisterZine Arcade.mgl"},      // back to Stable
		{betaEntry, []string{stable, legacy}, "MisterZine Arcade BETA.mgl"}, // and again
	} {
		if err := ensureMGLAt(step.entry, step.others...); err != nil {
			t.Fatal(err)
		}
		if got := mglEntries(t, dir); len(got) != 1 || got[0] != step.want {
			t.Fatalf("after writing %s: %q", filepath.Base(step.entry), got)
		}
	}
}

// Open at boot and Return after game write the entry's path to Main's
// command FIFO whole and unquoted, space included, as one line.
func TestMenuEntryReachesMainWhole(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "MiSTer_cmd")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	// Main's end: open for reading first, so the command's non-blocking
	// open finds a reader
	r, err := os.OpenFile(fifo, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	was := mainCmdFIFO
	mainCmdFIFO = fifo
	defer func() { mainCmdFIFO = was }()
	if err := openMenuEntry(sendMainCmd); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(buf[:n]); got != "load_core /media/fat/MisterZine Arcade.mgl\n" {
		t.Fatalf("Main reads %q", got)
	}
}

func mglEntries(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names
}

// A card updated from a release that wrote MisterZine.mgl ends with the one
// entry under the new name: the old one goes, in any case of its name,
// whether or not the new one was there already, and other entries stay.
func TestEnsureMGLReplacesTheOldEntry(t *testing.T) {
	for _, already := range []bool{false, true} {
		dir := t.TempDir()
		entry, legacy := filepath.Join(dir, "MisterZine Arcade.mgl"), filepath.Join(dir, "MisterZine.mgl")
		keep := filepath.Join(dir, "MisterZine Tools.mgl")
		for _, p := range []string{filepath.Join(dir, "misterzine.mgl"), keep} {
			if err := os.WriteFile(p, []byte(mglBody), 0644); err != nil {
				t.Fatal(err)
			}
		}
		if already {
			if err := os.WriteFile(entry, []byte(mglBody), 0644); err != nil {
				t.Fatal(err)
			}
		}
		for run := 0; run < 2; run++ { // and again: nothing changes
			if err := ensureMGLAt(entry, legacy); err != nil {
				t.Fatal(err)
			}
			if got := mglEntries(t, dir); len(got) != 2 || got[0] != "MisterZine Arcade.mgl" || got[1] != "MisterZine Tools.mgl" {
				t.Fatalf("new entry there before %v, run %d: %q", already, run, got)
			}
			if b, err := os.ReadFile(entry); err != nil || string(b) != mglBody {
				t.Fatalf("entry %q, %v", b, err)
			}
		}
	}
}

// Turning the shortcut off (and Uninstall's launcher disable) removes the
// entry under any of its names.
func TestDisableLauncherRemovesEveryName(t *testing.T) {
	dir := t.TempDir()
	startup := filepath.Join(dir, "user-startup.sh")
	entry, betaEntry, legacy := filepath.Join(dir, "MisterZine Arcade.mgl"), filepath.Join(dir, "MisterZine Arcade BETA.mgl"), filepath.Join(dir, "MisterZine.mgl")
	for _, p := range []string{entry, betaEntry, legacy} {
		if err := os.WriteFile(p, []byte(mglBody), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := disableLauncherFiles(startup, entry, betaEntry, legacy); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{entry, betaEntry, legacy} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s survived disabling", filepath.Base(p))
		}
	}
}
