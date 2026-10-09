package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// cv1kMRA is the shape of Insert-Coin's Cave pack: the trainer keeps the
// game's name and core, loads the game's archive for graphics and sound,
// and its own for the patched program.
func cv1kMRA(name, setname, program string) string {
	return `<misterromdescription>
    <name>` + name + `</name>
    <setname>` + setname + `</setname>
    <rbf>ikacore_CV1k</rbf>
    <rom index="1" address="0x34400000" zip="akatana.zip" md5="none"><part name="u2" crc="89a2e1a5"/></rom>
    <rom index="2" address="0x3C800000" zip="akatana.zip" md5="none"><part name="u23" crc="34a67e24"/></rom>
    <rom index="0" zip="` + program + `" md5="none"><part name="u4" crc="9f9317ac"/></rom>
</misterromdescription>`
}

func writeCard(t *testing.T, files map[string]string) string {
	t.Helper()
	card := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(card, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return card
}

// A pack that files patched sets in a folder inside the game's is read, two
// folders down at most, warm runs included; the archives every <rom> loads
// are kept apart from the program's.
func TestAlternativesReadGameSubfolders(t *testing.T) {
	const name = "Akai Katana (Japan, 2010/ 8/13 MASTER VER.)"
	card := writeCard(t, map[string]string{
		"_Arcade/_alternatives/_Akai Katana/_trainer/Akai Katana.mra":          cv1kMRA(name, "akatanat", "akatanat.zip"),
		"_Arcade/_alternatives/_Akai Katana/_trainer/_old/Akai Katana.mra":     cv1kMRA(name, "akatanat0", "akatanat0.zip"),
		"_Arcade/_alternatives/_Akai Katana/_trainer/_old/_v1/Akai Katana.mra": cv1kMRA(name, "akatanat1", "akatanat1.zip"),
		"_Arcade/_alternatives/_Akai Katana/_Organized/Akai Katana.mra":        cv1kMRA(name, "akatanao", "akatanao.zip"),
	})
	cache := filepath.Join(card, "cache", "alts.json")
	for _, run := range []string{"cold", "warm"} {
		got, skipped, err := ScanAlternativesWithError(card, cache)
		if err != nil || len(skipped) != 0 || len(got) != 2 {
			t.Fatalf("%s: alts=%v skipped=%v err=%v", run, got, skipped, err)
		}
		if got[0].Path != "_Arcade/_alternatives/_Akai Katana/_trainer/Akai Katana.mra" || got[1].Path != "_Arcade/_alternatives/_Akai Katana/_trainer/_old/Akai Katana.mra" {
			t.Fatalf("%s: paths %q %q", run, got[0].Path, got[1].Path)
		}
		a := got[0]
		if a.Setname != "akatanat" || len(a.Zips) != 1 || a.Zips[0] != "akatanat.zip" || len(a.OtherZips) != 1 || a.OtherZips[0] != "akatana.zip" {
			t.Fatalf("%s: header %+v", run, a)
		}
	}
}

// A trainer under the game's folder becomes a version of the catalogue game
// (and of a local game when the catalogue lacks it), while an archive shared
// with a differently named game on the same core ties nothing together.
func TestInferParents(t *testing.T) {
	const name = "Akai Katana (Japan, 2010/ 8/13 MASTER VER.)"
	trainer := Alt{Path: "_Arcade/_alternatives/_Akai Katana/_trainer/Akai Katana.mra", RBF: "ikacore_cv1k", Setname: "akatanat", Name: name,
		Zips: []string{"akatanat.zip"}, OtherZips: []string{"akatana.zip"}}
	row := data.Row{K: "akatana", SN: "akatana", Core: "ikacore_CV1k", Title: name, Base: "Arcade", MRA: "_Arcade/Akai Katana.mra"}

	alts := []Alt{trainer}
	InferParents(alts, nil, []data.Row{row})
	if alts[0].Parent != "akatana" {
		t.Fatalf("catalogue game: parent %q", alts[0].Parent)
	}
	if vs := Alternatives(alts, &row); len(vs) != 1 || vs[0] != trainer.Path {
		t.Fatalf("trainer is not a version: %v", vs)
	}

	// the game known only from the card, as the local walk read it
	alts = []Alt{trainer}
	base := Alt{Path: "_Arcade/Akai Katana.mra", RBF: "ikacore_cv1k", Setname: "akatana", Name: name, Zips: []string{"akatana.zip"}}
	InferParents(alts, []Alt{base}, nil)
	if alts[0].Parent != "akatana" {
		t.Fatalf("local game: parent %q", alts[0].Parent)
	}

	// copied up a level, as a player did before subfolders were read
	moved := trainer
	moved.Path = "_Arcade/_alternatives/_Akai Katana/Akai Katana.mra"
	alts = []Alt{moved}
	InferParents(alts, nil, []data.Row{row})
	if alts[0].Parent != "akatana" {
		t.Fatalf("moved trainer: parent %q", alts[0].Parent)
	}

	// the game's archive listed as a fallback for the program counts too,
	// while a differently named game's does not (futaribl is Another Ver)
	fallback := trainer
	fallback.Setname, fallback.Zips, fallback.OtherZips = "futaribljt", []string{"futaribljt.zip", "futariblj.zip", "futaribl.zip"}, []string{"futariblj.zip", "futaribl.zip"}
	alts = []Alt{fallback}
	rowBL := data.Row{K: "futaribl", SN: "futaribl", Core: "ikacore_CV1k", Title: name + " Another Ver", Base: "Arcade"}
	InferParents(alts, nil, []data.Row{row, rowBL, {K: "futariblj", SN: "futariblj", Core: "ikacore_CV1k", Title: name, Base: "Arcade"}})
	if alts[0].Parent != "futariblj" {
		t.Fatalf("fallback archive: parent %q", alts[0].Parent)
	}

	// another game loading this game's archive for its samples is no version
	other := trainer
	other.Name, other.Setname = "Deathsmiles", "dsmiles"
	alts = []Alt{other}
	InferParents(alts, nil, []data.Row{row})
	if alts[0].Parent != "" {
		t.Fatalf("shared archive linked: parent %q", alts[0].Parent)
	}

	// outside _alternatives, or with a parent of its own, nothing changes
	loose, named := trainer, trainer
	loose.Path, named.Parent = "_Arcade/Akai Katana trainer.mra", "akatanab"
	alts = []Alt{loose, named}
	InferParents(alts, nil, []data.Row{row})
	if alts[0].Parent != "" || alts[1].Parent != "akatanab" {
		t.Fatalf("parents %q %q", alts[0].Parent, alts[1].Parent)
	}

	// archives of two same-named games: no telling which, so neither
	two := trainer
	two.OtherZips = []string{"akatana.zip", "akatanaj.zip"}
	alts = []Alt{two}
	rowJ := row
	rowJ.K, rowJ.SN = "akatanaj", "akatanaj"
	InferParents(alts, nil, []data.Row{row, rowJ})
	if alts[0].Parent != "" {
		t.Fatalf("ambiguous archives linked: parent %q", alts[0].Parent)
	}
}
