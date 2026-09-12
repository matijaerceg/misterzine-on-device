package scan

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// An MRA without a usable header is skipped and cached with its folder, not
// a scan failure; a rewrite of that file in place (the folder's mtime does
// not move) is still noticed.
func TestAlternativeSkipsUnreadableMRAAndSeesRepair(t *testing.T) {
	cases := []struct{ content, reason string }{
		{"", "no XML content"},
		{"\n\n", "no XML content"},
		{"not an mra\n", "no XML content"},
		{"<misterromdescription><rbf>game</rbf>", "unexpected EOF"},
		{"<misterromdescription><name>x</name></misterromdescription>", "no <rbf> element"},
	}
	for _, c := range cases {
		t.Run(c.content, func(t *testing.T) {
			card := t.TempDir()
			dir := filepath.Join(card, "_Arcade", "_alternatives", "_Game")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "game.mra")
			os.WriteFile(p, []byte(c.content), 0644)
			stamp, err := os.Stat(dir)
			if err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(card, "cache", "alts.json")
			got, skipped, err := ScanAlternativesWithError(card, cache)
			if err != nil || len(got) != 0 || len(skipped) != 1 || !skipped[0].Fresh ||
				skipped[0].Path != "_Arcade/_alternatives/_Game/game.mra" || !strings.Contains(skipped[0].Reason, c.reason) {
				t.Fatalf("unreadable MRA: alts=%v skipped=%+v err=%v", got, skipped, err)
			}
			// Warm run: the skip comes from the cache, so it is not fresh.
			if got, skipped, err = ScanAlternativesWithError(card, cache); err != nil || len(got) != 0 || len(skipped) != 1 || skipped[0].Fresh {
				t.Fatalf("cached skip: alts=%v skipped=%+v err=%v", got, skipped, err)
			}
			good := `<misterromdescription><rbf>game</rbf><setname>gamea</setname><rom index="0" zip="game.zip"><part name="x"/></rom></misterromdescription>`
			if err := os.WriteFile(p, []byte(good), 0644); err != nil {
				t.Fatal(err)
			}
			os.Chtimes(dir, stamp.ModTime(), stamp.ModTime())
			if got, skipped, err = ScanAlternativesWithError(card, cache); err != nil || len(got) != 1 || len(skipped) != 0 {
				t.Fatalf("in-place repair stayed hidden: alts=%v skipped=%+v err=%v", got, skipped, err)
			}
		})
	}
}

// A version 2 cache entry (written before skips were recorded) stays valid.
func TestAlternativeCacheAcceptsVersion2(t *testing.T) {
	card := fakeCard(t)
	for _, name := range []string{"_1942", "_Colony 7"} {
		dir := filepath.Join(card, "_Arcade", "_alternatives", name)
		stamp := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(dir, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	cache := filepath.Join(card, "cache", "alts.json")
	if _, _, err := ScanAlternativesWithError(card, cache); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	old := bytes.ReplaceAll(b, []byte(`"version":3`), []byte(`"version":2`))
	if bytes.Equal(old, b) {
		t.Fatalf("cache holds no version 3 entries: %s", b)
	}
	if err := os.WriteFile(cache, old, 0644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	os.Chtimes(cache, stamp, stamp)
	if got, _, err := ScanAlternativesWithError(card, cache); err != nil || len(got) != 2 {
		t.Fatalf("version 2 cache: %v %v", got, err)
	}
	if info, err := os.Stat(cache); err != nil || !info.ModTime().Equal(stamp) {
		t.Fatal("version 2 cache entries were rewritten without a change on the card")
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
	initial, _, err := ScanAlternativesWithError(card, cache)
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
	if _, _, err := ScanAlternativesWithError(card, cache); err != nil {
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
	got, _, err := ScanAlternativesWithError(card, cache)
	if err == nil || len(got) != len(initial)+1 {
		t.Fatal("cache write failure lost usable scan results or its error")
	}
	after, err := os.ReadFile(cache)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatal("failed cache write damaged previous file")
	}
}
