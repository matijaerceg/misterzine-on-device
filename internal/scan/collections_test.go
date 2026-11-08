package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestCollectionDiscoveryAndOwnership(t *testing.T) {
	card := t.TempDir()
	const mra = "<misterromdescription><rbf>unit</rbf><setname>testset</setname><rom index=\"0\"/></misterromdescription>"
	for _, p := range []string{"_Ongo/_Arcade/Game.mra", "_Ignored/Game.mra", "backup/_Arcade/Game.mra", "_Organized/_Arcade/Game.mra", "_Ongo/_Organized/Game.mra"} {
		putFile(t, filepath.Join(card, p), mra)
	}
	putFile(t, filepath.Join(card, "_Homebrew", "cores", "custom.rbf"), "x")
	putFile(t, filepath.Join(card, "_Homebrew", "Renamed.mra"), "<misterromdescription><rbf>custom</rbf><setname>mygame</setname></misterromdescription>")
	rows := []data.Row{
		{K: "mirrored", Base: "Arcade", MRA: "_Arcade/Game.mra", Core: "unit", SN: "testset", Src: "ppriest"},
		{K: "ongo", Base: "Arcade", MRA: "_Arcade/Game.mra", Core: "unit", SN: "testset", Src: "ongo"},
		{K: "custom", Base: "Arcade", MRA: "_Arcade/My Game.mra", Core: "custom", SN: "mygame"},
	}
	idx := ScanCores(card)
	got, changed := idx.ResolveCollections(card, "", rows)
	if idx.Err != nil || idx.collections.err != nil || !changed || len(idx.collections.games) != 2 {
		t.Fatalf("scan: %v %v, games %v", idx.Err, idx.collections.err, idx.collections.games)
	}
	if got[0].MRA != rows[0].MRA || got[1].MRA != "_Ongo/_Arcade/Game.mra" || got[2].MRA != "_Homebrew/Renamed.mra" {
		t.Fatalf("ownership: %v", got)
	}
	if Status(card, idx, &got[1]).Found() || !Status(card, idx, &got[2]).Found() {
		t.Fatal("the cores or the partial install were mixed up")
	}
	// The original rows and globally selectable system cores are unaffected.
	if rows[1].MRA != "_Arcade/Game.mra" || rows[1].CatalogueMRA != "" {
		t.Fatal("input catalogue changed")
	}
	if _, ok := idx.Lookup("custom"); ok {
		t.Fatal("isolated RBF leaked into direct core lookup")
	}
}

func TestCollectionROMFallback(t *testing.T) {
	card := t.TempDir()
	mra := "_Pack/_Arcade/Test.mra"
	putFile(t, filepath.Join(card, mra), `<misterromdescription><rbf>unit</rbf><rom index="0" zip="pack.zip"><part name="one.bin"/></rom></misterromdescription>`)
	putZip(t, filepath.Join(card, "_Pack", "mame", "pack.zip"), member{name: "one.bin", data: "game"})
	cache := filepath.Join(t.TempDir(), "roms.json")
	c := NewROMCheck(card)
	c.CachePath = cache
	runSweep(t, c, []string{mra})
	c = NewROMCheck(card)
	c.CachePath = cache
	if _, reused := runSweep(t, c, []string{mra}); reused != 1 {
		t.Fatal("warm fallback did not reuse its verdict")
	}
	if got := c.Check(mra, true); got.Text != "" {
		t.Fatalf("collection fallback: %+v", got)
	}
	if err := os.MkdirAll(filepath.Join(card, "games", "mame"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, reused := runSweep(t, c, []string{mra}); reused != 0 {
		t.Fatal("reused an isolated ROM verdict after the shared folder appeared")
	}
	if got := c.Check(mra, true); !got.Block {
		t.Fatalf("empty shared directory must win: %+v", got)
	}
}
