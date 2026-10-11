//go:build linux

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// localRowsName is where the last complete scan's local rows are kept, in
// the cache folder (data.EncodeLocalRows).
const localRowsName = "local_rows.json"

// withSavedLocalRows adds the local rows the last complete scan saved to the
// catalogue rows, so the list shows the card's own games before this run's
// scan has found them again. MergeLocal drops any the catalogue now lists,
// as a catalogue refresh does: a star left on one of those is moved by the
// first complete scan (AdoptStrandedLocalKeys). note is for the startup log
// line; n is how many local rows were added.
func (h *host) withSavedLocalRows(catalogue []data.Row, hash string) (rows []data.Row, n int, note string) {
	b, err := os.ReadFile(filepath.Join(h.root, "cache", localRowsName))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			h.lg.Printf("cache: %v", err)
		}
		return catalogue, 0, ""
	}
	saved, local, err := data.DecodeLocalRows(b)
	if err != nil {
		// the scan rewrites it; until then the list waits for the scan
		h.lg.Printf("cache: %s ignored: %v", localRowsName, err)
		return catalogue, 0, ""
	}
	rows = data.MergeLocal(catalogue, local)
	n = len(rows) - len(catalogue)
	if n == 0 {
		return catalogue, 0, ""
	}
	note = fmt.Sprintf(" + %d local rows from the last scan", n)
	if saved != hash {
		note += fmt.Sprintf(" (of catalogue %.8s)", saved)
	}
	return rows, n, note
}

// saveLocalRows keeps a complete scan's local rows for the next start, when
// they differ from what is saved. Called off the UI goroutine by the scan;
// scans never overlap.
func (h *host) saveLocalRows(hash string, local []data.Row) {
	b, err := data.EncodeLocalRows(hash, local)
	if err != nil {
		h.lg.Printf("cache: %s: %v", localRowsName, err)
		return
	}
	path := filepath.Join(h.root, "cache", localRowsName)
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, b) {
		return
	}
	if err := store.WriteAtomic(path, b); err != nil {
		h.lg.Printf("cache: %s: %v", localRowsName, err)
	}
}
