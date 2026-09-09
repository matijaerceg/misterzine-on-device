//go:build linux

package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/images"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

func backgroundHost(t *testing.T) *host {
	t.Helper()
	h := favoritesHost(t.TempDir())
	h.card = t.TempDir()
	h.quit = make(chan struct{})
	h.uiRun = make(chan func(), 8)
	h.netCh = make(chan string, 4)
	h.scanCh = make(chan scanResult, 1)
	h.clock.Trusted = true
	h.client = fetch.NewClient("test")
	h.img = images.New(filepath.Join(h.root, "shots"), nil, h.lg, 1<<20)
	t.Cleanup(func() { close(h.quit); h.img.Close() })
	return h
}

func finishBackground(t *testing.T, h *host) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for h.checkRunning || h.scanRunning {
		select {
		case f := <-h.uiRun:
			f()
		case r := <-h.scanCh:
			h.receiveScan(r)
		case <-h.netCh:
		case <-deadline:
			t.Fatal("background work did not finish")
		}
	}
}

func TestRefreshUsesInstalledHashAndCoalescesRequests(t *testing.T) {
	h := backgroundHost(t)
	raw := []byte(`[{"k":"new-game","title":"New game","base":"Console"}]`)
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	var metas, downloads atomic.Int32
	var fail atomic.Bool
	entered, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/meta.json" {
			if fail.Load() {
				http.Error(w, "temporary failure", 500)
				return
			}
			if metas.Add(1) == 1 {
				close(entered)
				<-release
			}
			fmt.Fprintf(w, `{"hash":%q,"updated":"2026-09-08T00:00Z"}`, hash)
		} else {
			downloads.Add(1)
			w.Write(raw)
		}
	}))
	defer srv.Close()
	old := fetch.Site
	fetch.Site = srv.URL
	defer func() { fetch.Site = old }()
	h.requestCheck()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("no request")
	}
	for i := 0; i < 20; i++ {
		h.requestCheck()
	}
	close(release)
	finishBackground(t, h)
	if metas.Load() != 1 || downloads.Load() != 1 || h.a.Data().Hash != hash {
		t.Fatal("overlapping check or missing dataset installation")
	}
	for i := 0; i < 3; i++ {
		h.requestCheck()
		finishBackground(t, h)
	}
	if metas.Load() != 4 || downloads.Load() != 1 {
		t.Fatalf("unchanged data downloaded again: meta=%d data=%d", metas.Load(), downloads.Load())
	}
	fail.Store(true)
	h.requestCheck()
	finishBackground(t, h)
	if d := time.Until(h.nextCheck); d < 10*time.Second || d > 16*time.Second {
		t.Fatalf("failure retry delay: %v", d)
	}
	fail.Store(false)
	h.requestCheck()
	finishBackground(t, h)
	if d := time.Until(h.nextCheck); d < 29*time.Minute || d > 31*time.Minute {
		t.Fatalf("success check delay: %v", d)
	}
	cached, err := os.ReadFile(filepath.Join(h.root, "cache", "data.json"))
	if err != nil || string(cached) != string(raw) {
		t.Fatalf("cache mismatch: %v", err)
	}
}

func TestScanCoalescesAndReplacesStaleRowStatuses(t *testing.T) {
	h := backgroundHost(t)
	h.a.SetData(data.Ingest([]data.Row{{K: "old", Core: "Missing"}}, "old", time.Now()), nil)
	h.alts = []scan.Alt{{Path: "previous.mra"}}
	h.requestScan()
	for i := 0; i < 20; i++ {
		h.requestScan()
	}
	// Data changes before either queued scan result is applied.
	h.a.SetData(data.Ingest([]data.Row{{K: "a"}, {K: "b"}}, "new", time.Now()), nil)
	var first scanResult
	select {
	case first = <-h.scanCh:
	case <-time.After(3 * time.Second):
		t.Fatal("scan not started")
	}
	h.receiveScan(first)
	if len(h.status) != 0 || len(h.alts) != 1 {
		t.Fatal("stale statuses applied or fast pass erased alternatives")
	}
	final := <-h.scanCh
	if !final.final {
		t.Fatal("missing final scan result")
	}
	h.receiveScan(final)
	if len(h.alts) != 0 {
		t.Fatal("empty alternatives scan failed to clear deleted alternatives")
	}
	finishBackground(t, h)
	if len(h.status) != 2 || h.scanPending {
		t.Fatal("coalesced rescan did not use latest rows")
	}
	select {
	case <-h.scanCh:
		t.Fatal("extra overlapping scan result")
	default:
	}
}

func TestUpdateDefersNewBackgroundWorkUntilFinished(t *testing.T) {
	h := backgroundHost(t)
	h.updateRunning = true
	h.a.SetUpdate(updater.State{ID: "run", Status: "running"}, true)
	h.requestScan()
	// Prevent HTTP work while exercising pending-refresh resume.
	h.checkRunning = true
	h.requestCheck()
	if h.scanRunning || !h.scanPending || !h.checkPending {
		t.Fatal("work started during Update All")
	}
	h.receiveUpdate(updateResult{state: updater.State{ID: "run", Status: "completed"}})
	if !h.scanRunning || h.updateRunning {
		t.Fatal("post-update scan did not start")
	}
	h.checkRunning = false
	// Check remains pending until it can actually start.
	if !h.checkPending {
		t.Fatal("pending refresh lost")
	}
	finishBackground(t, h)
}

func TestBackgroundDeliveryStopsWhenUIQuits(t *testing.T) {
	h := &host{quit: make(chan struct{}), uiRun: make(chan func()), scanCh: make(chan scanResult), netCh: make(chan string)}
	close(h.quit)
	done := make(chan struct{})
	go func() {
		h.runOnUI(func() { t.Error("ran after quit") })
		h.sendNet("test")
		h.sendScan(scanResult{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker blocked after UI quit")
	}
}
