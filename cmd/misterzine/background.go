//go:build linux

package main

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
)

// Requests and completion flags belong to the UI loop. Snapshot the current
// dataset here; workers must never reach back into the mutable app.
func (h *host) requestCheck() {
	if h.updateRunning || h.updatePending {
		h.checkPending = true
		return
	}
	if h.checkRunning {
		return
	}
	h.checkPending = false
	h.checkRunning = true
	current, trusted := h.a.Data().Hash, h.clockTrusted()
	go h.check(current, trusted)
}

// Schedule from completion, so a failed first request retries promptly too.
func (h *host) finishCheck() {
	h.checkRunning = false
	delay := 30 * time.Minute
	if !h.clockTrusted() || h.checkFailed.Load() {
		delay = 15 * time.Second
	}
	h.nextCheck = time.Now().Add(delay)
}

func (h *host) requestScan() {
	if h.updateRunning || h.updatePending || h.scanRunning {
		h.scanPending = true
		return
	}
	h.scanPending = false
	h.scanRunning = true
	ds := h.a.Data()
	go h.scan(ds.Rows, ds.Hash, ds.Updated)
}

func (h *host) receiveScan(r scanResult) {
	first := h.index == nil
	if r.index != nil {
		h.index = r.index
	}
	if r.hash == h.a.Data().Hash && r.index != nil {
		h.status = r.status
	} else if r.hash != h.a.Data().Hash {
		// Reordered/new rows cannot use old positional statuses. Queue one scan
		// of the latest data rather than doing filesystem work during painting.
		h.scanPending = true
	}
	// A nil alternatives result means genuinely empty only on the final pass.
	if r.final && r.index != nil {
		h.alts = r.alts
	}
	if r.index != nil {
		h.a.SetHiddenSources(r.hidden, r.iniFound)
	}
	h.a.Refilter()
	if r.hash == h.a.Data().Hash && r.notice != "" {
		h.a.Notice(r.notice, 8*time.Second)
	} else if first && h.favLoadFailed {
		h.a.Notice(app.FavoritesUnavailableNotice, 12*time.Second)
	}
	if r.final {
		h.scanRunning = false
		if h.manualScan && !h.scanPending && r.hash == h.a.Data().Hash {
			h.manualScan = false
			// A failed core index sends no index: nothing to count. An
			// incomplete alternatives pass still delivers every status.
			h.a.FinishScan(r.notice, r.index != nil)
		}
		if h.scanPending {
			h.requestScan()
		}
	}
}

func (h *host) sendScan(r scanResult) bool {
	select {
	case h.scanCh <- r:
		return true
	case <-h.quit:
		return false
	}
}

func (h *host) sendNet(s string) {
	select {
	case h.netCh <- s:
	case <-h.quit:
	}
}
