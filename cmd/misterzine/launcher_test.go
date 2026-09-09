//go:build linux

package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDisableLauncherPreservesOtherBootHooks(t *testing.T) {
	dir := t.TempDir()
	startup, mgl := filepath.Join(dir, "user-startup.sh"), filepath.Join(dir, "MisterZine.mgl")
	others := "#!/bin/sh\n# fan control\n/usr/bin/fan start\n"
	if err := os.WriteFile(startup, []byte(others+startupMark+"\n"+startupLine+"\n"), 0751); err != nil {
		t.Fatal(err)
	}
	// MiSTer's login shell uses a restrictive umask. Set the fixture's
	// mode explicitly so this checks preservation, not file creation.
	if err := os.Chmod(startup, 0751); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mgl, []byte(mglBody), 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := disableLauncherFiles(startup, mgl); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(startup)
		if err != nil || string(got) != others {
			t.Fatalf("other boot hooks changed: %q, %v", got, err)
		}
		st, err := os.Stat(startup)
		if err != nil || st.Mode().Perm() != 0751 {
			t.Fatalf("boot script permissions changed: %v, %v", st, err)
		}
		if _, err := os.Stat(mgl); !os.IsNotExist(err) {
			t.Fatal("main-menu entry survived disabling")
		}
	}
}

func TestDisableLauncherWithoutBootScript(t *testing.T) {
	dir := t.TempDir()
	mgl := filepath.Join(dir, "MisterZine.mgl")
	if err := os.WriteFile(mgl, []byte(mglBody), 0644); err != nil {
		t.Fatal(err)
	}
	if err := disableLauncherFiles(filepath.Join(dir, "missing.sh"), mgl); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mgl); !os.IsNotExist(err) {
		t.Fatal("menu entry was not removed when the boot hook was already absent")
	}
}

func TestDisableLauncherReportsUnreadableBootScript(t *testing.T) {
	dir := t.TempDir()
	mgl := filepath.Join(dir, "MisterZine.mgl")
	if err := os.WriteFile(mgl, []byte(mglBody), 0644); err != nil {
		t.Fatal(err)
	}
	if err := disableLauncherFiles(dir, mgl); err == nil {
		t.Fatal("unreadable boot script was silently accepted")
	}
	if _, err := os.Stat(mgl); err != nil {
		t.Fatal("menu entry was removed despite the boot-script failure")
	}
}

func TestLauncherStopKeepsOwningWatcherAlive(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := launcherHelper(ctx, "watcher", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("watcher was killed before finishing its app session: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "menu-restored")); err != nil {
		t.Fatal("watcher did not reach menu restoration after the app exited")
	}
	if _, err := os.Stat(filepath.Join(dir, "watch.pid")); !os.IsNotExist(err) {
		t.Fatal("finished watcher left a PID file behind")
	}
}

func TestLauncherStopStillStopsIdleWatcher(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := launcherHelper(ctx, "idle", dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	path := filepath.Join(dir, "watch.pid")
	deadline := time.Now().Add(2 * time.Second)
	for watcherPIDFrom(path) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("idle watcher did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopWatcher(path)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("idle watcher did not stop")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("stopped watcher left its PID file behind")
	}
}

func TestOldWatcherCannotRemoveNewWatcherPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watch.pid")
	if err := os.WriteFile(path, []byte("222"), 0644); err != nil {
		t.Fatal(err)
	}
	removeWatcherPID(path, 111)
	if b, err := os.ReadFile(path); err != nil || string(b) != "222" {
		t.Fatal("old watcher removed the replacement watcher's PID file")
	}
}

func launcherHelper(ctx context.Context, role, dir string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWatcherStopHelper$", "watch-test")
	cmd.Env = append(os.Environ(), "MZ_WATCH_TEST_ROLE="+role, "MZ_WATCH_TEST_DIR="+dir)
	return cmd
}

// A real parent/child process pair models the watcher blocked in cmd.Run
// while the app calls stopWatcher. No MiSTer device files are touched.
func TestWatcherStopHelper(t *testing.T) {
	role, dir := os.Getenv("MZ_WATCH_TEST_ROLE"), os.Getenv("MZ_WATCH_TEST_DIR")
	if role == "" {
		return
	}
	path := filepath.Join(dir, "watch.pid")
	if role == "app" {
		stopWatcher(path)
		if watcherPIDFrom(path) == 0 {
			t.Fatal("app lost its owning watcher")
		}
		return
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		t.Fatal(err)
	}
	if role == "idle" {
		time.Sleep(10 * time.Second)
		return
	}
	defer removeWatcherPID(path, os.Getpid())
	cmd := exec.Command(os.Args[0], "-test.run=^TestWatcherStopHelper$")
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "MZ_WATCH_TEST_ROLE=") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	cmd.Env = append(cmd.Env, "MZ_WATCH_TEST_ROLE=app")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("app helper: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "menu-restored"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherDetectsAtomicBinaryReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(path, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	running, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if binaryReplaced(path, running) {
		t.Fatal("unchanged binary triggered restart")
	}
	if err := os.WriteFile(path+".new", []byte("new"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".new", path); err != nil {
		t.Fatal(err)
	}
	if !binaryReplaced(path, running) {
		t.Fatal("atomic replacement was missed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if binaryReplaced(path, running) {
		t.Fatal("missing binary triggered restart")
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if binaryReplaced(path, running) {
		t.Fatal("directory triggered restart")
	}
}

func TestMenuRestoreWaitsForUpdateAndRespectsCoreChanges(t *testing.T) {
	for _, nextCore := range []string{"misterzine", "SNES", "MENU", ""} {
		t.Run(nextCore, func(t *testing.T) {
			var logs bytes.Buffer
			polls, waits := 0, 0
			restore := waitForMenuRestore(log.New(&logs, "", 0), func() (string, bool) {
				polls++
				if polls <= 3 {
					return "misterzine", true
				}
				return nextCore, false
			}, func(time.Duration) {
				waits++
				if waits > 3 {
					t.Fatal("never stopped waiting")
				}
			})
			if waits != 3 || restore != (nextCore == "misterzine") {
				t.Fatalf("restore=%v waits=%d", restore, waits)
			}
			if strings.Count(logs.String(), "waiting for updater") != 1 {
				t.Fatal("wait logging missing or repeated")
			}
		})
	}
}
