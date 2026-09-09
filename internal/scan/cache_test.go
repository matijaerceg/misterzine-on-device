package scan

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAlternativeCacheRetriesIncompleteMRA(t *testing.T) {
	for _, content := range []string{"", "<misterromdescription><rbf>game</rbf>"} {
		t.Run(content, func(t *testing.T) {
			card := t.TempDir()
			dir := filepath.Join(card, "_Arcade", "_alternatives", "_Game")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "game.mra")
			os.WriteFile(p, []byte(content), 0644)
			stamp, err := os.Stat(dir)
			if err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(card, "cache", "alts.json")
			if got, err := ScanAlternativesWithError(card, cache); err == nil || len(got) != 0 {
				t.Fatalf("incomplete MRA accepted: %v %v", got, err)
			}
			good := `<misterromdescription><rbf>game</rbf><setname>gamea</setname><rom index="0" zip="game.zip"><part name="x"/></rom></misterromdescription>`
			if err := os.WriteFile(p, []byte(good), 0644); err != nil {
				t.Fatal(err)
			}
			os.Chtimes(dir, stamp.ModTime(), stamp.ModTime())
			if got, err := ScanAlternativesWithError(card, cache); err != nil || len(got) != 1 {
				t.Fatalf("in-place repair stayed hidden: %v %v", got, err)
			}
		})
	}
}

func TestAlternativeCacheSkipsUnchangedWritesAndPreservesOnFailure(t *testing.T) {
	card := fakeCard(t)
	// Fix fixture directory mtimes after creating files (Windows can finish its
	// directory timestamp update just after the final file close).
	for _, name := range []string{"_1942", "_Colony 7"} {
		dir := filepath.Join(card, "_Arcade", "_alternatives", name)
		stamp := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(dir, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	cache := filepath.Join(card, "cache", "alts.json")
	initial, err := ScanAlternativesWithError(card, cache)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(cache, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanAlternativesWithError(card, cache); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cache)
	if err != nil || !info.ModTime().Equal(stamp) {
		nowBytes, _ := os.ReadFile(cache)
		t.Fatalf("unchanged cache rewritten: info=%v err=%v old=%s new=%s", info, err, original, nowBytes)
	}
	dir := filepath.Join(card, "_Arcade", "_alternatives", "_New")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "new.mra"), []byte(`<misterromdescription><rbf>new</rbf></misterromdescription>`), 0644)
	if err := os.Mkdir(cache+".tmp", 0755); err != nil {
		t.Fatal(err)
	}
	got, err := ScanAlternativesWithError(card, cache)
	if err == nil || len(got) != len(initial)+1 {
		t.Fatal("cache write failure lost usable scan results or its error")
	}
	after, err := os.ReadFile(cache)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatal("failed cache write damaged previous file")
	}
}
