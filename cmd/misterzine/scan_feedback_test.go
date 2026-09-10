//go:build linux

package main

import (
	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"path/filepath"
	"testing"
	"time"
)

func TestScanFailureKeepsPreviousInventory(t *testing.T) {
	h := backgroundHost(t)
	h.card = filepath.Join(h.card, "absent")
	h.index = &scan.Index{Cores: map[string]scan.Core{"known": {Path: "old.rbf"}}}
	h.status = []data.Status{data.StatusCurrent}
	h.alts = []scan.Alt{{Path: "old.mra"}}
	h.manualScan = true
	h.a.OpenScan()
	h.requestScan()
	select {
	case result := <-h.scanCh:
		if result.notice != "Card scan failed" || !result.final {
			t.Fatal(result)
		}
		h.receiveScan(result)
	case <-time.After(3 * time.Second):
		t.Fatal("failed scan stalled")
	}
	if h.scanRunning || h.manualScan || len(h.index.Cores) != 1 || len(h.alts) != 1 || h.status[0] != data.StatusCurrent {
		t.Fatal("failed scan discarded previous inventory or stayed active")
	}
	if h.a.Screen() != app.ScreenScan {
		t.Fatal("lost manual result screen")
	}
}
