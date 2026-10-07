package main

import (
	"os"
	"testing"
)

// TestMain keeps the running-updater check to this run's own folder, so a
// second checkout's tests running at the same time on this machine, with
// their fake Update All scripts, never read as a real updater here.
func TestMain(m *testing.M) {
	scope, err := os.MkdirTemp("", "mz-updater-scope-")
	if err != nil {
		panic(err)
	}
	os.Setenv("MISTERZINE_UPDATER_SCOPE", scope)
	code := m.Run()
	os.RemoveAll(scope)
	os.Exit(code)
}
