//go:build linux

package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
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
	if pending := h.pendingCatalogue.Load(); pending != nil {
		current = pending.Meta.Hash
	}
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
	h.nextScanRetry = time.Time{}
	ds := h.a.Data()
	h.scanningNeoGeoFolder = h.settings.ScanNeoGeoFolder
	go h.scan(scanRequest{displayed: ds, catalogue: h.pendingCatalogue.Load(), savedLocals: h.localSaved,
		scanFolder: h.scanningNeoGeoFolder, hasInventory: h.index != nil || h.savedCard != nil})
}

// Card recovery is independent of the network check. Space out repeated
// failures, and let an explicit rescan or a new catalogue start immediately.
func (h *host) finishScanRetry(complete bool, now time.Time) {
	if complete {
		h.scanRetryDelay = 0
		h.nextScanRetry = time.Time{}
		return
	}
	h.scanRetryDelay = min(max(30*time.Second, h.scanRetryDelay*2), 10*time.Minute)
	h.nextScanRetry = now.Add(h.scanRetryDelay)
}

func (h *host) retryScan(now time.Time) {
	if h.nextScanRetry.IsZero() || now.Before(h.nextScanRetry) || h.scanRunning || h.updateRunning || h.updatePending {
		return
	}
	h.requestScan()
}

// receiveScan installs a scan result. Results are checked against the
// dataset on screen when the scan started. A replacement catalogue must
// also still be the one we're waiting for.
func (h *host) receiveScan(r scanResult) {
	first := h.index == nil
	current := r.gen == h.a.Data().Gen && h.scanningNeoGeoFolder == h.settings.ScanNeoGeoFolder &&
		r.catalogue == h.pendingCatalogue.Load()
	if current && (h.savedCard != nil || r.catalogue != nil && h.index != nil) {
		// The saved list includes local games, collection paths and versions.
		// Publishing only the first pass would split that inventory in two.
		if !r.final {
			return
		}
		if !r.complete {
			// Keep the last complete list and report the failed/incomplete scan.
			r.index, r.rows = nil, nil
		}
	}
	selection := ""
	if current && r.final && r.rows != nil {
		// The card's local rows changed: install the new row set and its
		// statuses in one step, before anything paints.
		old := h.a.Data()
		hash, updated := old.Hash, old.Updated
		if r.catalogue != nil {
			hash = r.catalogue.Meta.Hash
			updated, _ = data.ParseMetaTime(r.catalogue.Meta.Updated)
		}
		ds := data.Ingest(r.rows, hash, updated)
		if ds.Gen != r.nextGen {
			h.lg.Printf("scan: local rows generation mismatch (%s vs %s); rescanning", ds.Gen, r.nextGen)
			h.scanPending = true
			current = false
		} else {
			selection = h.a.CursorKey()
			if moved := r.moves[selection]; moved != "" {
				selection = moved
			}
			h.status, h.owners = r.status, r.owners
			h.index = r.index
			h.a.SetData(ds, nil)
			if r.catalogue != nil {
				if news := h.a.CatalogueNews(old, r.catalogue.Rows); news != "" {
					h.a.Notice(news, 12*time.Second)
				}
				h.pendingCatalogue.Store(nil)
				h.dirty = true
			}
			h.localSaved = false
			// a standin row the card now runs from the catalogue hands its star over
			h.a.RenameKeys(r.moves)
			if h.img != nil {
				h.prefetchSet = true
				h.img.SetPrefetch(picsFor(ds), h.settings.Prefetch)
			}
		}
	} else if current && r.index != nil {
		h.index = r.index
		h.status, h.owners = r.status, r.owners
	} else if !current {
		// Reordered/new rows cannot use old positional statuses. Queue one scan
		// of the latest data rather than doing filesystem work during painting.
		h.scanPending = true
	}
	// A nil alternatives result means genuinely empty only on the final pass.
	if r.final && r.index != nil && current {
		if r.complete {
			h.savedCard = nil
			// the last run's local rows are this card's, as far as a scan can tell
			h.localSaved = false
			// a star on a local row that a catalogue refresh covered before
			// this run's first scan could list it, so no takeover saw it
			if moves := h.a.AdoptStrandedLocalKeys(); len(moves) > 0 {
				h.lg.Printf("scan: %d local keys moved to the catalogue rows that now list them", len(moves))
			}
		}
		h.alts, h.altCores, h.altSets = r.alts, r.altCores, r.altSets
		h.altGen = h.a.Data().Gen
		h.a.ROMChanged()
		// before the sweep, so its answers go by this catalogue and card
		h.roms.SetAccess(r.access)
		// the sweep starts on a later tick: set up in the same frame as the
		// new row set's first repaint, it stretched that frame to 150-200 ms
		// on a Pi
		h.sweepDue = time.Now().Add(500 * time.Millisecond)
	}
	if current && r.index != nil {
		h.a.SetHiddenSources(r.hidden, r.iniFound)
		h.a.SetSourceFiles(r.sourceFiles)
	}
	h.a.Refilter()
	if selection != "" && h.a.CursorKey() != selection && h.a.Data().Index(selection) >= 0 {
		h.a.MoveToKey(selection)
	}
	if current && r.notice != "" {
		// Repeated automatic failures stay in the log and empty-list message;
		// they should not keep covering the status bar while someone browses.
		if r.notice != h.cardScanProblem || h.manualScan {
			notice := r.notice
			if r.final && r.index == nil && (h.index != nil || h.savedCard != nil) {
				notice += ". Using saved list."
			}
			h.a.Notice(notice, 8*time.Second)
		}
	} else if first && h.favLoadFailed {
		h.a.Notice(app.FavoritesUnavailableNotice, 12*time.Second)
	}
	if r.final {
		if current {
			h.cardScanProblem = r.notice
			h.finishScanRetry(r.complete, time.Now())
		}
		if current && r.diag != nil {
			h.scanDiag = r.diag
		}
		if !h.prefetchSet && h.img != nil {
			// the first scan is over: the picture service may count and
			// fetch the prefetch list now
			h.prefetchSet = true
			h.img.SetPrefetch(picsFor(h.a.Data()), h.settings.Prefetch)
		}
		h.scanRunning = false
		if current && !h.scanPending {
			h.a.CardScanned(r.index != nil)
		}
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
	h.roms.Sweep(paths, func(n, reused int, d time.Duration) {
		h.lg.Printf("rom sweep: %d MRAs checked, %d of them from the last run's verdicts (%v)", n, reused, d.Round(time.Millisecond))
		h.romReportMu.RLock()
		defer h.romReportMu.RUnlock()
		if list := h.romList; list != "" {
			if err := h.roms.WriteList(list, paths); err != nil {
				h.lg.Printf("rom sweep: list: %v", err)
			}
		}
	})
}

// romListPath is where a finished sweep writes its list of ROM problems:
// whenever the ROM report is allowed (it is free for everyone now).
func romListPath(root string, month access.Month, showBeta bool) string {
	if !access.ROMReport.Allowed(month, showBeta) {
		return ""
	}
	return filepath.Join(root, scan.ROMListName)
}

// cardRelative is path as a player finds it on the card, such as
// misterzine/rom_problems.txt; "" stays "".
func cardRelative(card, path string) string {
	if path == "" {
		return ""
	}
	if rel, err := filepath.Rel(card, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
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
