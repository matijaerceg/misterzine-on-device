//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

const cardCacheName = "card_status.json"

func (h *host) cardStatus(i int) data.Status {
	if i >= 0 && i < len(h.status) {
		return h.status[i]
	}
	return data.StatusUnknown
}

func (h *host) cardCheckMessage() string {
	if !h.scanRunning && h.cardScanProblem != "" {
		return h.cardScanProblem
	}
	if h.scanRunning || h.index == nil {
		return "Checking card..."
	}
	return ""
}

// A complete scan's answers for the list. Keep the game keys as well as the
// generation: a positional answer from another row must never look installed.
type cardInventory struct {
	statusOnly bool                   // an interrupted or older update left the catalogue and inventory out of step
	Version    int                    `json:"version"`
	Generation string                 `json:"generation"`
	ScanFolder bool                   `json:"scan_neogeo_folder,omitempty"`
	Status     map[string]data.Status `json:"status"`
	Paths      map[string]string      `json:"collection_paths,omitempty"`
	Owners     map[string]string      `json:"shared_file_owners,omitempty"`
	Alts       map[string][]string    `json:"versions,omitempty"`
	AltCores   map[string]string      `json:"version_cores,omitempty"`
	AltSets    map[string]string      `json:"version_sets,omitempty"`
	Missing    map[string]string      `json:"missing_version_cores,omitempty"`
	Hidden     map[string]bool        `json:"hidden_sources,omitempty"`
	INIFound   bool                   `json:"ini_found,omitempty"`
}

// Restore before the saved filter and return cursor are applied. Keep the
// real core index nil until the live scan reports.
func (h *host) restoreCard(ds *data.Dataset) {
	start := time.Now()
	b, err := os.ReadFile(filepath.Join(h.root, "cache", cardCacheName))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			h.lg.Printf("card cache: %v", err)
		}
		return
	}
	var saved cardInventory
	if err := json.Unmarshal(b, &saved); err != nil {
		h.lg.Printf("card cache ignored: %v", err)
		return
	}
	if saved.Version != 1 || ds.Hash == "" || saved.Generation == "" || saved.ScanFolder != h.settings.ScanNeoGeoFolder {
		return
	}
	saved.statusOnly = saved.Generation != ds.Gen
	if !saved.statusOnly && len(saved.Status) != len(ds.Rows) {
		return
	}
	status := make([]data.Status, len(ds.Rows))
	matched := 0
	for i, r := range ds.Rows {
		st, ok := saved.Status[r.K]
		if !ok && saved.statusOnly {
			continue // no previous observation of this game
		}
		if !ok || st > data.StatusLikelyOutdated {
			return
		}
		status[i] = st
		matched++
	}
	if saved.statusOnly {
		// Saved statuses have stable game keys. Installed paths, shared-file
		// ownership and version lists belong to the old catalogue.
		if matched > 0 {
			h.status = status
			h.savedCard = &cardInventory{statusOnly: true, Paths: saved.Paths, Owners: saved.Owners,
				Hidden: saved.Hidden, INIFound: saved.INIFound}
			h.lg.Printf("card cache: %d statuses from the previous catalogue (%v)", matched, time.Since(start).Round(time.Millisecond))
		}
		return
	}
	for key, p := range saved.Paths {
		i := ds.Index(key)
		if i < 0 || i >= ds.NCat || ds.Rows[i].MRA == "" || !validCachedLauncher(p) || !strings.EqualFold(path.Ext(p), ".mra") {
			return
		}
	}
	for _, paths := range saved.Alts {
		for _, p := range paths {
			if !validCachedLauncher(p) {
				return
			}
		}
	}
	for key, p := range saved.Paths {
		r := &ds.Rows[ds.Index(key)]
		r.CatalogueMRA, r.MRA = r.MRA, p
	}
	h.status, h.owners = status, saved.Owners
	h.alts, h.altCores, h.altSets, h.altGen = saved.Alts, saved.AltCores, saved.AltSets, ds.Gen
	saved.Status, saved.Paths = nil, nil
	h.savedCard = &saved
	h.lg.Printf("card cache: restored %d statuses (%v)", len(status), time.Since(start).Round(time.Millisecond))
}

// When we only have last-known presence, the catalogue's own launcher can
// still be checked at Start. Let the scan resolve a collection override or
// remembered version before offering a different file in its place.
func (h *host) checkProvisionalLaunch(row *data.Row, target string) string {
	s := h.savedCard
	if s == nil || !s.statusOnly {
		return ""
	}
	for _, wanted := range []string{s.Paths[row.K], h.state.Versions[row.K], h.state.Versions[row.PreferenceKey()]} {
		if wanted != "" && wanted != target {
			return h.cardCheckMessage()
		}
	}
	if strings.EqualFold(path.Ext(target), ".mra") {
		if row.Core == "" || s.Owners[row.K] != "" {
			return h.cardCheckMessage()
		}
		if err := scan.CheckMRACore(h.card, target, row.Core); err != nil {
			return err.Error()
		}
	}
	return ""
}

// Versions resolved by the scan are MRA files or duplicate MGL shortcuts.
func validCachedLauncher(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	return (ext == ".mra" || ext == ".mgl") && fs.ValidPath(p) && !strings.HasPrefix(p, "core:") && !strings.ContainsAny(p, "\x00\r\n")
}

// saveCard runs on the scan worker, after statuses have been matched to the
// final rows. Avoid writing the card again after an unchanged return.
func (h *host) saveCard(rows []data.Row, gen string, scanFolder bool, res scanResult) {
	saved := cardInventory{
		Version:    1,
		Generation: gen,
		ScanFolder: scanFolder,
		Status:     make(map[string]data.Status, len(rows)),
		Paths:      map[string]string{},
		Owners:     res.owners,
		Alts:       map[string][]string{},
		AltCores:   res.altCores,
		AltSets:    res.altSets,
		Missing:    map[string]string{},
		Hidden:     res.hidden,
		INIFound:   res.iniFound,
	}
	for i, row := range rows {
		saved.Status[row.K] = res.status[i]
		if row.CatalogueMRA != "" {
			saved.Paths[row.K] = row.MRA
		}
		paths := availableAlternatives(res.alts[row.K], res.altCores, res.index)
		if len(paths) > 0 {
			saved.Alts[row.K] = paths
		}
		if name, ok := res.index.VersionCore(row.MRA); name != "" && !ok {
			saved.Missing[row.MRA] = name
		}
		for _, p := range paths {
			if name, ok := res.index.VersionCore(p); name != "" && !ok {
				saved.Missing[p] = name
			}
		}
	}
	b, err := json.Marshal(saved)
	if err == nil {
		p := filepath.Join(h.root, "cache", cardCacheName)
		if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, b) {
			return
		}
		err = store.WriteAtomic(p, b)
	}
	if err != nil {
		h.lg.Printf("card cache: %v", err)
	}
}
