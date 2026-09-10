//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func TestSupportCapturePersistsWithoutEnablingRemoteDebug(t *testing.T) {
	h := favoritesHost(t.TempDir())
	now := time.Now()
	h.troubleshooting.loaded = true
	h.troubleshooting.capture = support.NewCapture(support.Report{Version: "test"}, now.Add(-time.Second), now.Add(time.Second))
	h.troubleshooting.capture.AddDevice(support.Device{Node: "event2", Name: "Arcade"})
	h.troubleshooting.capture.Record("event2", 1, 299, 1, now)
	h.recordSupportInput(platform.Event{Key: platform.KeyStart, Pressed: true, Source: "debug"})
	r := h.finishSupport(false)
	if !r.ButtonReceived() || r.StartDown != 0 || h.troubleshooting.capture != nil || h.debugEnabled {
		t.Fatalf("capture leaked: %+v", r)
	}
	fresh := favoritesHost(h.root)
	if got := fresh.loadSupport(); !got.Tested || got.Signals[0].Code != 299 || got.Devices[0].Name != "Arcade" {
		t.Fatalf("lost saved evidence: %+v", got)
	}
	for _, file := range []string{"debug.flag", "debug-token"} {
		if _, err := os.Stat(filepath.Join(h.root, file)); !os.IsNotExist(err) {
			t.Fatalf("created %s", file)
		}
	}
}

func TestSupportLaunchFailureIsSavedAndNormalLaunchDoesNotRecord(t *testing.T) {
	h := favoritesHost(t.TempDir())
	h.card = t.TempDir()
	h.requestLaunch("_Arcade/Absent.mra")
	if _, err := os.Stat(filepath.Join(h.root, "troubleshooting.json")); !os.IsNotExist(err) {
		t.Fatal("normal launch created support report")
	}
	r := h.testSupportLaunch("Absent", "_Arcade/Absent.mra")
	if r.Launch.Result != "Cannot launch: target missing or invalid" || r.Launch.Detail == "" || h.launch != "" {
		t.Fatalf("wrong launch evidence: %+v", r)
	}
	fresh := favoritesHost(h.root)
	if got := fresh.loadSupport(); got.Launch.Result != r.Launch.Result {
		t.Fatal("launch result not retained")
	}
}
