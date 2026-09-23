//go:build linux

package main

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
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
	go h.scan(ds.Rows, ds.NCat, ds.Gen, ds.Hash, ds.Updated)
}

// receiveScan installs a scan result. Results are checked against the
// dataset generation they were computed for: the catalogue hash plus the
// local rows, since positional statuses are only right for that row set.
func (h *host) receiveScan(r scanResult) {
	first := h.index == nil
	if r.index != nil {
		h.index = r.index
	}
	current := r.gen == h.a.Data().Gen
	if current && r.final && r.rows != nil {
		// The card's local rows changed: install the new row set and its
		// statuses in one step, before anything paints.
		old := h.a.Data()
		ds := data.Ingest(r.rows, old.Hash, old.Updated)
		if ds.Gen != r.nextGen {
			h.lg.Printf("scan: local rows generation mismatch (%s vs %s); rescanning", ds.Gen, r.nextGen)
			h.scanPending = true
			current = false
		} else {
			h.status = r.status
			h.a.SetData(ds, nil)
			// a standin row the card now runs from the catalogue hands its star over
			h.a.RenameKeys(r.moves)
			if h.img != nil {
				h.img.SetPrefetch(picsFor(ds), h.settings.Prefetch)
			}
		}
	} else if current && r.index != nil {
		h.status = r.status
	} else if !current {
		// Reordered/new rows cannot use old positional statuses. Queue one scan
		// of the latest data rather than doing filesystem work during painting.
		h.scanPending = true
	}
	// A nil alternatives result means genuinely empty only on the final pass.
	if r.final && r.index != nil && current {
		h.alts, h.altCores = r.alts, r.altCores
		h.altGen = h.a.Data().Gen
		// the sweep starts on a later tick: set up in the same frame as the
		// new row set's first repaint, it stretched that frame to 150-200 ms
		// on a Pi
		h.sweepDue = time.Now().Add(500 * time.Millisecond)
	}
	if r.index != nil {
		h.a.SetHiddenSources(r.hidden, r.iniFound)
	}
	h.a.Refilter()
	if current && r.notice != "" {
		h.a.Notice(r.notice, 8*time.Second)
	} else if first && h.favLoadFailed {
		h.a.Notice(app.FavoritesUnavailableNotice, 12*time.Second)
	}
	if r.final {
		if r.diag != nil {
			h.scanDiag = r.diag
		}
		h.scanRunning = false
		if h.manualScan && !h.scanPending && current {
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

// sweepROMs hands the ROM check every file the list may mark, after a
// complete scan: each row's main MRA when the card has it, then the
// versions the card offers. The answers arrive over roms.Ready.
func (h *host) sweepROMs() {
	if h.roms == nil {
		return
	}
	ds := h.a.Data()
	paths := make([]string, 0, len(ds.Rows))
	seen := map[string]bool{}
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	for i := range ds.Rows {
		r := &ds.Rows[i]
		if r.MRA == "" {
			continue
		}
		if i < len(h.status) && h.status[i].Found() {
			add(r.MRA)
		}
		for _, alt := range h.alternatives(r) {
			add(alt)
		}
	}
	h.lg.Printf("rom sweep: %d files to check", len(paths))
	h.roms.Sweep(paths, func(n int, d time.Duration) {
		h.lg.Printf("rom sweep: %d MRAs checked (%v)", n, d.Round(time.Millisecond))
	})
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
