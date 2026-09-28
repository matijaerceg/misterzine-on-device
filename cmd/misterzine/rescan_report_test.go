//go:build linux

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// An Update All run that brings a newer core and a missing MRA ends with a
// rescan; the run's screen changes when that scan lands (from "Checking
// card..." to what it changed).
func TestUpdateRescanReportFollowsTheRun(t *testing.T) {
	h := backgroundHost(t)
	mk := func(rel string) {
		p := filepath.Join(h.card, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk("_Arcade/cores/Defender_20240101.rbf")
	mk("_Arcade/Older.mra")
	rows := []data.Row{
		{K: "older", Title: "Older", Base: "Arcade", MRA: "_Arcade/Older.mra", Core: "defender", Updated: "2026-07-14"},
		{K: "fresh", Title: "Fresh", Base: "Arcade", MRA: "_Arcade/Fresh.mra", Core: "defender", Updated: "2026-07-14"},
	}
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, Status: func(i int) data.Status {
		if i < len(h.status) {
			return h.status[i]
		}
		return data.StatusUnknown
	}}, data.Ingest(rows, "test", time.Now()), nil)
	h.requestScan()
	finishBackground(t, h)
	if len(h.status) != 2 || h.status[0] != data.StatusOutdated || h.status[1] != data.StatusNotFound {
		t.Fatalf("before the run: %v", h.status)
	}
	h.updateRunning = true
	h.a.SetUpdate(updater.State{ID: "run", Status: "running"}, true)
	mk("_Arcade/cores/Defender_20260714.rbf")
	mk("_Arcade/Fresh.mra")
	h.receiveUpdate(updateResult{state: updater.State{ID: "run", Status: "completed"}})
	if !h.scanRunning {
		t.Fatal("the run did not end with a rescan")
	}
	checking := updateFrame(h.a)
	finishBackground(t, h)
	if h.status[0] != data.StatusCurrent || h.status[1] != data.StatusCurrent {
		t.Fatalf("after the run: %v", h.status)
	}
	if bytes.Equal(checking, updateFrame(h.a)) {
		t.Fatal("the update screen did not change when the rescan landed")
	}
}

func updateFrame(a *app.App) []byte {
	if a.Screen() != app.ScreenUpdate {
		panic("not on the update screen")
	}
	a.Paint()
	return bytes.Clone(a.Logical().Pix)
}
