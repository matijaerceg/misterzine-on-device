//go:build linux

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func TestRestartExecHelper(t *testing.T) {
	if root := os.Getenv("MZ_RESTART_TEST_ROOT"); root != "" {
		h := &host{root: root, lg: log.Default()}
		os.Exit(h.restartApp())
	}
}

func TestRestartExecutesInstalledProgramWithOriginalFlags(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "misterzine"), []byte("#!/bin/sh\nprintf 'replacement\\n'\nprintf '%s\\n' \"$@\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	args := []string{"-test.run=^TestRestartExecHelper$", "--", "--resume"}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "MZ_RESTART_TEST_ROOT="+root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restart: %v: %s", err, out)
	}
	if string(out) != "replacement\n"+strings.Join(args, "\n")+"\n" {
		t.Fatalf("replacement arguments: %q", out)
	}
}

func TestFinishedUpdateDetectsReplacementAndOffersRestart(t *testing.T) {
	h := recoveryHost(t, t.TempDir())
	if err := os.WriteFile(filepath.Join(h.root, "misterzine"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	h.a.SetUpdate(updater.State{ID: "run", Status: "running"}, true)
	h.updateRunning, h.scanRunning = true, true
	state := updater.State{ID: "run", Status: "completed"}
	if err := store.Save(updater.StatePath(h.root), state); err != nil {
		t.Fatal(err)
	}
	h.applyUpdate(state, false)
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for !h.a.UpdateRestartAvailable() {
		select {
		case result := <-h.updates:
			h.receiveUpdate(result)
		case <-timer.C:
			t.Fatal("finished run did not offer the replaced program")
		}
	}
	h.requestUpdateRestart("old-run")
	if h.restartRequested {
		t.Fatal("stale action requested restart")
	}
	h.updateRunning = true
	h.requestUpdateRestart("run")
	if h.restartRequested {
		t.Fatal("active update requested restart")
	}
	h.updateRunning = false
	h.requestUpdateRestart("run")
	if !h.restartRequested {
		t.Fatal("finished update could not request restart")
	}
}

func TestDifferentProgram(t *testing.T) {
	dir := t.TempDir()
	running, installed := filepath.Join(dir, "running"), filepath.Join(dir, "installed")
	if err := os.WriteFile(running, []byte("old program"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, content string
		mode          os.FileMode
		want, wantErr bool
	}{
		{"identical reinstall", "old program", 0755, false, false},
		{"changed same size", "new program", 0755, true, false},
		{"not executable", "new program", 0644, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(installed, []byte(test.content), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(installed, test.mode); err != nil {
				t.Fatal(err)
			}
			got, err := differentProgram(running, installed)
			if got != test.want || (err != nil) != test.wantErr {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
	if got, err := differentProgram(running, filepath.Join(dir, "missing")); got || err == nil {
		t.Fatal("missing program offered restart")
	}
	if got, err := differentProgram(running, dir); got || err == nil {
		t.Fatal("directory offered restart")
	}
}
