package scan

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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

// Older entries must be rebuilt to acquire parent metadata.
func TestAlternativeCacheRebuildsVersion2(t *testing.T) {
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
	old := bytes.ReplaceAll(b, []byte(`"version":7`), []byte(`"version":2`))
	if bytes.Equal(old, b) {
		t.Fatalf("cache holds no version 7 entries: %s", b)
	}
	if err := os.WriteFile(cache, old, 0644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	os.Chtimes(cache, stamp, stamp)
	if got, _, err := ScanAlternativesWithError(card, cache); err != nil || len(got) != 2 {
		t.Fatalf("version 2 cache: %v %v", got, err)
	}
	if info, err := os.Stat(cache); err != nil || info.ModTime().Equal(stamp) {
		t.Fatal("version 2 cache entries were not upgraded")
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

// An MRA its folder lists that is gone when opened is skipped, not a scan
// failure, and never cached, so a later scan reads it once it is there. A
// dangling link stands in for what a card produces: a file deleted while the
// scan runs, or a damaged exFAT entry that lists but cannot be opened.
func TestAlternativeSkipsGoneMRA(t *testing.T) {
	card := t.TempDir()
	dir := filepath.Join(card, "_Arcade", "_alternatives", "_Game")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	good := `<misterromdescription><rbf>game</rbf><setname>gamea</setname><rom index="0" zip="game.zip"><part name="x"/></rom></misterromdescription>`
	if err := os.WriteFile(filepath.Join(dir, "good.mra"), []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(card, "elsewhere.mra")
	if err := os.Symlink(target, filepath.Join(dir, "gone.mra")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	cache := filepath.Join(card, "cache", "alts.json")
	for _, run := range []string{"cold", "warm"} {
		got, skipped, err := ScanAlternativesWithError(card, cache)
		if err != nil || len(got) != 1 || len(skipped) != 1 || !skipped[0].Gone || !skipped[0].Fresh ||
			skipped[0].Reason != GoneReason || skipped[0].Path != "_Arcade/_alternatives/_Game/gone.mra" {
			t.Fatalf("%s run: alts=%v skipped=%+v err=%v", run, got, skipped, err)
		}
	}
	if raw, err := os.ReadFile(cache); err != nil || strings.Contains(string(raw), "gone.mra") {
		t.Fatalf("gone entry cached: %s %v", raw, err)
	}
	// The file turns up (a repaired card) without the folder's mtime moving.
	if err := os.WriteFile(target, []byte(strings.Replace(good, "gamea", "gameb", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	if got, skipped, err := ScanAlternativesWithError(card, cache); err != nil || len(got) != 2 || len(skipped) != 0 {
		t.Fatalf("repaired: alts=%v skipped=%+v err=%v", got, skipped, err)
	}
}

// Only a missing file is gone: a link loop still fails the scan.
func TestAlternativeLinkLoopStaysAProblem(t *testing.T) {
	card := t.TempDir()
	dir := filepath.Join(card, "_Arcade", "_alternatives", "_Game")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(dir, "a.mra"), filepath.Join(dir, "b.mra")
	if err := os.Symlink(b, a); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	if _, skipped, err := ScanAlternativesWithError(card, ""); err == nil || len(skipped) != 0 {
		t.Fatalf("link loop: skipped=%+v err=%v", skipped, err)
	}
}

// A game folder listed but gone by the time it is read is named, not a scan
// failure, and the other folders still scan; any other failure reading it
// still is one.
func TestAlternativeSkipsGoneFolder(t *testing.T) {
	card := fakeCard(t)
	gone := filepath.Join(card, "_Arcade", "_alternatives", "_Colony 7")
	orig := readDir
	t.Cleanup(func() { readDir = orig })
	fail := func(err error) {
		readDir = func(name string) ([]os.DirEntry, error) {
			if name == gone {
				return nil, &fs.PathError{Op: "open", Path: name, Err: err}
			}
			return orig(name)
		}
	}
	fail(fs.ErrNotExist)
	alts, _, dirs, err := ScanAlternativesWithDirs(card, "")
	if err != nil || len(alts) != 1 || len(dirs) != 1 || !dirs[0].Gone || dirs[0].Reason != GoneReason ||
		dirs[0].Path != "_Arcade/_alternatives/_Colony 7" {
		t.Fatalf("gone folder: alts=%v dirs=%+v err=%v", alts, dirs, err)
	}
	fail(fs.ErrPermission)
	if alts, _, dirs, err := ScanAlternativesWithDirs(card, ""); err == nil || len(alts) != 1 || len(dirs) != 0 {
		t.Fatalf("unreadable folder: alts=%v dirs=%+v err=%v", alts, dirs, err)
	}
}

// An _alternatives folder the scan cannot look at is a scan problem naming
// it, not a folder quietly left out with every version in it. Here _Arcade
// lists its names but, without search permission, nothing inside it can be
// reached.
func TestAlternativeRootReadErrorIsAProblem(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs folder permissions that apply to this user")
	}
	card := t.TempDir()
	for rel, set := range map[string]string{
		"_Arcade/_alternatives/_Game/game.mra":       "gamea",
		"_Arcade/_DB/_alternatives/_Other/other.mra": "othera",
	} {
		p := filepath.Join(card, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(mra(set, set, "game", "")), 0644); err != nil {
			t.Fatal(err)
		}
	}
	arcade := filepath.Join(card, "_Arcade")
	if err := os.Chmod(arcade, 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(arcade, 0755) })
	alts, _, err := ScanAlternativesWithError(card, "")
	if !errors.Is(err, fs.ErrPermission) || !strings.Contains(err.Error(), "_alternatives") || len(alts) != 0 {
		t.Fatalf("roots out of reach: alts=%v err=%v", alts, err)
	}
	if err := os.Chmod(arcade, 0755); err != nil {
		t.Fatal(err)
	}
	if alts, _, err := ScanAlternativesWithError(card, ""); err != nil || len(alts) != 2 {
		t.Fatalf("roots in reach: alts=%v err=%v", alts, err)
	}
}
