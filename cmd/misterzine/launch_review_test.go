//go:build linux

package main

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentQuitHasOneWinner(t *testing.T) {
	h := &host{quit: make(chan struct{})}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if h.closeQuit() {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatal("shutdown started more than once")
	}
	select {
	case <-h.quit:
	default:
		t.Fatal("quit channel remains open")
	}
}

func TestInvalidLaunchKeepsAppOpen(t *testing.T) {
	h := favoritesHost(t.TempDir())
	h.card = t.TempDir()
	h.quit = make(chan struct{})
	for _, target := range []string{"core:missing", "_Arcade/missing.mra", "folder.mra"} {
		if target == "folder.mra" {
			os.Mkdir(filepath.Join(h.card, target), 0755)
		}
		h.requestLaunch(target)
		if h.launch != "" {
			t.Fatal("invalid target committed launch")
		}
		select {
		case <-h.quit:
			t.Fatal("invalid launch closed app")
		default:
		}
	}
	os.WriteFile(filepath.Join(h.card, "present.mra"), []byte("test"), 0644)
	h.requestLaunch("present.mra") // no command interface in this test
	if h.launch != "" {
		t.Fatal("missing command interface committed launch")
	}
	select {
	case <-h.quit:
		t.Fatal("missing interface closed app")
	default:
	}
}

func TestStartupHookRecognitionMatchesRemoval(t *testing.T) {
	for _, line := range []string{startupLine, "  " + startupLine + "  ", startupLine + " # enabled by MisterZine"} {
		if !hasStartupHook(line) {
			t.Fatalf("did not recognize active hook %q", line)
		}
		root := t.TempDir()
		startup := filepath.Join(root, "startup.sh")
		mgl := filepath.Join(root, "MisterZine.mgl")
		os.WriteFile(startup, []byte("#!/bin/sh\n"+line+"\necho preserved\n"), 0755)
		if err := disableLauncherFiles(startup, mgl); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(startup)
		if hasStartupHook(string(got)) || string(got) != "#!/bin/sh\necho preserved\n" {
			t.Fatalf("hook removal mismatch: %q", got)
		}
	}
	for _, line := range []string{"# " + startupLine, "echo '" + startupLine + "'", "false && " + startupLine} {
		if hasStartupHook(line) {
			t.Fatalf("unrelated/commented command treated as hook: %q", line)
		}
	}
}
