package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCardLayout(t *testing.T) {
	card := fakeCard(t)
	mk := func(rel string) {
		t.Helper()
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk("_Arcade/cores/Gigandes_baz.rbf")
	mk("_Arcade/Gigandes (World bazset).mra")
	mk("_Arcade/_Organized/_A/x.mra")
	mk("_Arcade/_Extra/a.mra")
	mk("_Arcade/_Extra/deeper/b.mra")
	mk("_Arcade/_Extra/deeper/deepest/c.mra")
	mk("games/mame/gigandes.zip")
	mk("games/mame/volfied.zip")
	mk("games/hbmame/x.zip")
	mk("MiSTer.ini")
	app := filepath.Join(card, "misterzine")
	mk("misterzine/log.txt")
	mk("misterzine/cache/local.json")

	got := strings.Join(CardLayout(card, app), "\n")
	for _, want := range []string{
		"Card top level: ",
		"_Arcade/ (",
		"MiSTer.ini",
		"_Arcade/: 3 MRAs, 4 folders", // Colony 7, 1942, Gigandes; _Extra, _Organized, _alternatives, cores
		"  _Organized/: skipped, an organiser's folder",
		"  cores/: ",
		"  _alternatives/: ",
		"  _Extra/: 1 MRAs, 1 folders",
		"    deeper/: 1 MRAs, 1 folders",
		"      deepest/: 1 MRAs, not listed deeper",
		"Gigandes_baz.rbf",
		"Other core folders (.rbf files): _Console 1, _Computer 1, _Other 0, _Utility 0",
		"(mame: 2 zips, hbmame: 1 zips)",
		"log.txt (1 B)",
		"cache/",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "x.mra") {
		t.Error("a file inside a skipped folder was listed")
	}
}

// A mame folder at the top of the SD card is the one MiSTer picks, and then
// cannot read from; the layout says so.
func TestCardLayoutRootMame(t *testing.T) {
	card := t.TempDir()
	os.MkdirAll(filepath.Join(card, "mame"), 0755)
	os.MkdirAll(filepath.Join(card, "games", "mame"), 0755)
	got := strings.Join(CardLayout(card, ""), "\n")
	if !strings.Contains(got, "a mame folder at the top of the SD card") {
		t.Fatalf("no root mame note in\n%s", got)
	}
}

func TestCardLayoutBounded(t *testing.T) {
	card := t.TempDir()
	for i := 0; i < layoutFolders+100; i++ {
		os.MkdirAll(filepath.Join(card, "_Arcade", "_Many", fmt.Sprintf("d%04d", i)), 0755)
	}
	lines := CardLayout(card, "")
	if len(lines) > layoutFolders+20 {
		t.Fatalf("%d lines for a card with %d folders", len(lines), layoutFolders+100)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "... more folders not listed") {
		t.Fatal("the cut is not marked")
	}
}
