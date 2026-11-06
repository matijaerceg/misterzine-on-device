//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

func neoGeoCatalogue() *mgl.Catalogue {
	return mgl.NeoGeoCatalogue()
}

func (h *host) scanNeoGeo(scanFolder bool) (mgl.Result, error, error) {
	result, err := mgl.Scan(h.card, neoGeoCatalogue())
	if len(result.Problems) > 0 {
		err = errors.Join(err, errors.New(strings.Join(result.Problems, "; ")))
	}
	var folderErr error
	if scanFolder && err == nil {
		if scanErr := mgl.ScanCartridges(h.card, &result); scanErr != nil {
			result.Problems = append(result.Problems, scanErr.Error())
		}
		if len(result.Problems) > 0 {
			folderErr = errors.New(strings.Join(result.Problems, "; "))
		}
	}
	b, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr == nil {
		marshalErr = store.WriteAtomic(filepath.Join(h.root, "mgl-report.json"), b)
	}
	return result, errors.Join(err, marshalErr), folderErr
}

// This runs on the UI loop. Drop folder rows immediately on opt-out, even
// if a scan of a slow share is still coming back. Their saved stars survive.
func (h *host) setScanNeoGeoFolder(on bool) {
	h.settings.ScanNeoGeoFolder = on
	h.setDirty = true
	if !on {
		ds := h.a.Data()
		rows := make([]data.Row, 0, len(ds.Rows))
		status := make([]data.Status, 0, len(h.status))
		for i, row := range ds.Rows {
			if !row.IsNeoGeoFile() {
				rows = append(rows, row)
				if i < len(h.status) {
					status = append(status, h.status[i])
				}
			}
		}
		if len(rows) != len(ds.Rows) {
			// Only folder cartridges went away. Retained games keep their
			// card status, shared-file ownership and available versions.
			next := data.Ingest(rows, ds.Hash, ds.Updated)
			if h.altGen == ds.Gen {
				h.altGen = next.Gen
			}
			h.status = status
			h.a.SetData(next, nil)
		}
	}
	h.requestScan()
}

func (h *host) prepareLaunch(target string) (string, error) {
	ext := strings.ToLower(filepath.Ext(target))
	if _, ok := mgl.GameBackingPath(target); ok && ext != ".mra" && ext != ".rbf" && ext != ".mgl" {
		if !h.settings.ScanNeoGeoFolder {
			return "", errors.New("Neo Geo folder scanning is off")
		}
		gamePath, err := mgl.ValidateCartridge(h.card, target)
		if err != nil {
			return "", err
		}
		idx := h.index
		if idx == nil {
			// Cached games are selectable from the first frame. Look up
			// their core now without waiting for the catalogue status pass.
			idx = scan.ScanCores(h.card)
			idx.FindNeoGeoRBF(h.card)
		}
		rbf := idx.NeoGeoRBF
		if rbf == "" {
			return "", errors.New("standard Neo Geo core missing")
		}
		if !mgl.CorePresent(h.card, mgl.Document{RBF: rbf}) {
			return "", errors.New("Neo Geo core missing")
		}
		// Bind the launch to the scanned library even if Main prefers another drive.
		// Only this app-owned cache file is written, including for read-only NAS sets.
		doc := struct {
			XMLName xml.Name `xml:"mistergamedescription"`
			Core    string   `xml:"rbf"`
			File    mgl.File `xml:"file"`
		}{Core: rbf, File: mgl.File{Path: gamePath, Type: "f", Index: "1", Delay: "1"}}
		buf, err := xml.Marshal(doc)
		if err != nil {
			return "", err
		}
		// Main's SXML parser decodes the five named XML entities only.
		// Go uses numeric entities for quotes, including the apostrophe in KOF '99.
		buf = bytes.ReplaceAll(buf, []byte("&#39;"), []byte("&apos;"))
		buf = bytes.ReplaceAll(buf, []byte("&#34;"), []byte("&quot;"))
		p := filepath.Join(h.root, "cache", "neogeo-launch.mgl")
		if err := store.WriteAtomic(p, buf); err != nil {
			return "", fmt.Errorf("prepare Neo Geo launch: %w", err)
		}
		return mister.LaunchPath(h.card, p)
	}
	abs, err := mister.LaunchPath(h.card, target)
	if err == nil && strings.EqualFold(filepath.Ext(target), ".mgl") {
		err = validateNeoGeoLaunch(h.card, target)
	}
	return abs, err
}

// Re-read the shortcut before handing the unchanged file to Main. A shortcut
// edited since discovery must still satisfy the supported core and title rules.
func validateNeoGeoLaunch(card, target string) error {
	root, err := os.OpenRoot(card)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(target)
	if err != nil {
		return err
	}
	defer f.Close()
	d, err := mgl.Parse(f)
	if err != nil {
		return err
	}
	if ok, reason := d.Cartridge(); !ok {
		return errors.New(reason)
	}
	if _, ok := neoGeoCatalogue().Match(d); !ok {
		return errors.New("game target is not recognised in the Neo Geo arcade catalogue")
	}
	if !mgl.CorePresent(card, d) {
		return errors.New("Neo Geo core missing at shortcut's core path")
	}
	return nil
}
