package scan

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The zips a problem names are the ones to add or replace: the absent ones
// when any is, else the unreadable ones, else the one holding a wrong
// version, else every zip that lacks the part.
func TestROMIssueZips(t *testing.T) {
	prog := crcOf("program")
	for _, tc := range []struct {
		name, mra string
		zips      map[string][]member
		junk      string // a file named like a zip that is not one
		want      string
	}{
		{"all absent", `<rom index="0" zip="clone.zip|parent.zip"><part name="p1.bin"/></rom>`, nil, "", "clone.zip|parent.zip"},
		{"clone absent, parent has another version", `<rom index="0" zip="clone.zip|parent.zip" md5="none"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"parent.zip": {{name: "p1.bin", data: "parent"}}}, "", "clone.zip"},
		{"wrong version", `<rom index="0" zip="g.zip"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "older"}}}, "", "g.zip"},
		{"incomplete", `<rom index="0" zip="19xx.zip|qsound.zip"><part name="dl-1425.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"19xx.zip": {{name: "a", data: "a"}}, "qsound.zip": {{name: "b", data: "b"}}}, "", "19xx.zip|qsound.zip"},
		{"unreadable", `<rom index="0" zip="junk.zip|g.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "other", data: "o"}}}, "junk.zip", "junk.zip"},
		{"trailing space trimmed", `<rom index="0" zip="gorfpgm1.zip "><part name="873a.x1"/></rom>`, nil, "", "gorfpgm1.zip"},
		{"games-relative", `<rom index="0" zip="/hbmame/hb.zip"><part name="p"/></rom>`, nil, "", "/hbmame/hb.zip"},
		{"jtbeta through the key rule", `<rom index="0" zip="game.zip"><part name="p"/></rom><rom index="17" zip="jtbeta.zip" md5="None"><part name="beta.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"game.zip": {{name: "p", data: "p"}}}, "", "jtbeta.zip"},
		{"the MRA itself", `<rom index="0" zip="g.zip"><part name="p1.bin"/></rom><rom index="1" zip=g.zip"></rom>`, nil, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := filepath.Join(t.TempDir(), "fat")
			putFile(t, filepath.Join(card, "_Arcade/game.mra"), "<misterromdescription>"+tc.mra+"</misterromdescription>")
			mame := filepath.Join(card, "games/mame")
			if err := os.MkdirAll(mame, 0755); err != nil {
				t.Fatal(err)
			}
			for name, members := range tc.zips {
				putZip(t, filepath.Join(mame, name), members...)
			}
			if tc.junk != "" {
				putFile(t, filepath.Join(mame, tc.junk), "not a zip")
			}
			got := NewROMCheck(card).Check("_Arcade/game.mra", true)
			if got.Text == "" || got.Zips != tc.want {
				t.Fatalf("zips %q for %q; want %q", got.Zips, got.Text, tc.want)
			}
		})
	}
}

func TestROMListText(t *testing.T) {
	found := map[string]ROMResult{
		"_Arcade/b.mra":                       {Text: "Missing game ROM: pacman.zip or puckman.zip", Block: true, Zips: "pacman.zip|puckman.zip"},
		"_Arcade/A.mra":                       {Text: "Wrong ROM version: Pacman.ZIP (pm1.bin) is 1, the MRA expects 2", Zips: "Pacman.ZIP"},
		"_Arcade/_Alternatives/_Galaga/c.mra": {Text: "Cannot read game menu file", Block: true},
		"_Arcade/d.mra":                       {Text: "Incomplete ROM: galaga.zip (no gg1.bin)", Block: true, Zips: "galaga.zip"},
	}
	want := `MisterZine ROM check: 4 of 120 MRA files have a ROM problem.
Written after each card scan: the zip files first, then each MRA and its problem.

Zip files to add or replace (3):
galaga.zip
Pacman.ZIP
puckman.zip

MRA files (4):
_Arcade/_Alternatives/_Galaga/c.mra
  Cannot read game menu file
_Arcade/A.mra
  Wrong ROM version: Pacman.ZIP (pm1.bin) is 1, the MRA expects 2
_Arcade/b.mra
  Missing game ROM: pacman.zip or puckman.zip
_Arcade/d.mra
  Incomplete ROM: galaga.zip (no gg1.bin)
`
	if got := string(romListText(found, 120)); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	want = "MisterZine ROM check: no problems in 7 MRA files.\nWritten after each card scan.\n"
	if got := string(romListText(nil, 7)); got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestROMWriteList(t *testing.T) {
	c := NewROMCheck(t.TempDir())
	answers := map[string]ROMResult{
		"_Arcade/bad.mra": {Text: "Missing game ROM: x.zip", Block: true, Zips: "x.zip"},
		"_Arcade/ok.mra":  {},
	}
	c.run = func(rel string, _ *CoreAccess) ROMResult { return answers[rel] }
	for rel := range answers {
		c.Check(rel, true)
	}
	path := filepath.Join(t.TempDir(), "misterzine", ROMListName)
	// a core and an MRA the check never answered are no part of it
	paths := []string{"_Arcade/bad.mra", "_Arcade/ok.mra", "_Arcade/unchecked.mra", "_Arcade/cores/x.rbf"}
	if err := c.WriteList(path, paths); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := romListText(map[string]ROMResult{"_Arcade/bad.mra": answers["_Arcade/bad.mra"]}, 2); string(got) != string(want) {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}

	// the same findings leave the file alone
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteList(path, paths); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(path); err != nil || !st.ModTime().Equal(old) {
		t.Fatalf("unchanged findings rewrote the list: %v %v", st.ModTime(), err)
	}

	// new ones replace it
	answers["_Arcade/bad.mra"] = ROMResult{}
	c.Check("_Arcade/bad.mra", true)
	if err := c.WriteList(path, paths); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(romListText(nil, 2)) {
		t.Fatalf("after the fix got\n%s", got)
	}

	// with no MRA checked there is no list, and none to remove is fine
	for range 2 {
		if err := c.WriteList(path, []string{"_Arcade/cores/x.rbf"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("list kept with no MRA checked: %v", err)
	}
}
