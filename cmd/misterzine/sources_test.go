//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// A card scan reads downloader.ini and, with Sources on installed, the
// app drops the sources without a database there.
func TestScanHidesSourcesTheDownloaderLacks(t *testing.T) {
	h := backgroundHost(t)
	rows := []data.Row{
		{K: "m", Title: "MiSTer game", Base: "Arcade", Src: "distribution_mister"},
		{K: "x", Title: "Meathax game", Base: "Arcade", Src: "meathax"},
		{K: "r", Title: "rmCores game", Base: "Arcade", Src: "rmcores"},
	}
	h.a.SetData(data.Ingest(rows, "h1", time.Now()), nil)
	h.a.SetInstalledOnly(true)
	ini := "[mister]\nfilter = arcade\n\n[distribution_mister]\ndb_url = https://raw.githubusercontent.com/MiSTer-devel/Distribution_MiSTer/main/db.json.zip\n"
	if err := os.WriteFile(filepath.Join(h.card, "downloader.ini"), []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	h.requestScan()
	finishBackground(t, h)
	if h.a.Visible() != 1 {
		t.Fatalf("visible rows after the scan: %d", h.a.Visible())
	}
	// enabling a database and rescanning brings its source back
	if err := os.WriteFile(filepath.Join(h.card, "downloader.ini"), []byte(ini+"\n[meathax/meatcores]\ndb_url = https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h.requestScan()
	finishBackground(t, h)
	if h.a.Visible() != 2 {
		t.Fatalf("visible rows after enabling the database: %d", h.a.Visible())
	}
	if err := os.WriteFile(filepath.Join(h.card, "downloader.ini"), []byte(ini+"\n[meathax/meatcores]\ndb_url = https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip\n\n[rmonic79/rmcores]\ndb_url = https://raw.githubusercontent.com/rmonic79/rmcores/db/db.json.zip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h.requestScan()
	finishBackground(t, h)
	if h.a.Visible() != 3 {
		t.Fatalf("visible rows after enabling rmCores too: %d", h.a.Visible())
	}
	h.a.SetInstalledOnly(false)
	h.a.Refilter()
	os.Remove(filepath.Join(h.card, "downloader.ini"))
	h.a.SetInstalledOnly(true)
	h.requestScan()
	finishBackground(t, h)
	if h.a.Visible() != 3 {
		t.Fatalf("no downloader.ini must hide nothing: %d", h.a.Visible())
	}
}
