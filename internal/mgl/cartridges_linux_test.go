package mgl

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// Keep this sparse: only the ZIP headers use disk space. SetOffset makes
// the central directory point past 4 GiB, as it does on a full-size pack.
func TestLargeNeoGeoCollection(t *testing.T) {
	const offset = 1<<32 + 100
	card := t.TempDir()
	folder := filepath.Join(card, "games", "NEOGEO")
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(folder, "Collection.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Seek(offset, 0); err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	z.SetOffset(offset)
	for _, p := range []string{"World/mslug.neo", "World/3 Count Bout (3countb).neo"} {
		if _, err := z.Create(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	var result Result
	if err := ScanCartridges(card, &result); err != nil || len(result.Problems) != 0 || len(result.Rows) != 2 {
		t.Fatalf("large pack: %d games, %v, %v", len(result.Rows), result.Problems, err)
	}
	for _, r := range result.Rows {
		if _, err := ValidateCartridge(card, r.MRA); err != nil {
			t.Fatal(err)
		}
	}
}
