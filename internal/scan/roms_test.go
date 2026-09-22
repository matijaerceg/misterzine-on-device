package scan

import (
	"archive/zip"
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// member is one file in a test archive. A zero crc is computed from data;
// raw entries keep crc, method and flags exactly as given.
type member struct {
	name   string
	data   string
	crc    uint32
	raw    bool
	method uint16
	flags  uint16
}

func putFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func putZip(t *testing.T, p string, members ...member) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, m := range members {
		if m.raw {
			hw, err := w.CreateRaw(&zip.FileHeader{Name: m.name, Method: m.method, Flags: m.flags, CRC32: m.crc,
				CompressedSize64: uint64(len(m.data)), UncompressedSize64: uint64(len(m.data))})
			if err != nil {
				t.Fatal(err)
			}
			hw.Write([]byte(m.data))
			continue
		}
		mw, err := w.Create(m.name)
		if err != nil {
			t.Fatal(err)
		}
		mw.Write([]byte(m.data))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func crcOf(s string) string { return fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(s))) }

func md5Of(s string) string { return fmt.Sprintf("%x", md5.Sum([]byte(s))) }

func TestROMRequirements(t *testing.T) {
	prog, sound := crcOf("program"), crcOf("sound")
	for _, tc := range []struct {
		name, mra string
		zips      map[string][]member
		want      string // "" for no issue
		block     bool
	}{
		{"jurassic", `<rom index="4" zip="jpark.zip"><part name="program"/></rom><rom index="9" zip="jpark.zip"><part name="sprites"/></rom><rom index="0"><part>00</part></rom>`,
			nil, "Missing game ROM: jpark.zip", true},
		{"present", `<rom index="4" zip="jpark.zip"><part name="program"/></rom>`,
			map[string][]member{"jpark.zip": {{name: "program", data: "program"}}}, "", false},
		{"parent fallback", `<rom index="0" zip="clone.zip|parent.zip"><part name="program" crc="` + prog + `"/></rom>`,
			map[string][]member{"parent.zip": {{name: "program", data: "program"}}}, "", false},
		{"part override", `<rom index="0" zip="parent.zip"><part name="program" zip="extra.zip"/></rom>`,
			map[string][]member{"parent.zip": {{name: "program", data: "program"}}}, "Missing game ROM: extra.zip", true},
		{"inline", `<rom index="0"><part>1234</part></rom>`, nil, "", false},
		{"alternate layouts", `<rom index="0" zip="absent.zip"><part name="a"/></rom><rom index="0" zip="present.zip"><part name="b"/></rom>`,
			map[string][]member{"present.zip": {{name: "b", data: "b"}}}, "", false},
		{"later requirement", `<rom index="0"><part>00</part></rom><rom index="5" zip="sound.zip"><part name="sound"/></rom>`,
			nil, "Missing game ROM: sound.zip", true},
		{"comments", `<!-- invalid XML -- comment --><rom index="4" zip="jpark.zip"><part name="program"/></rom>`,
			nil, "Missing game ROM: jpark.zip", true},

		// looking inside
		{"found by crc under another name", `<rom index="0" zip="g.zip"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "renamed.bin", data: "program"}}}, "", false},
		{"name without crc", `<rom index="0" zip="g.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "anything"}}}, "", false},
		{"name in other case", `<rom index="0" zip="g.zip"><part name="P1.BIN"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "x"}}}, "", false},
		{"member in a folder is not the bare name", `<rom index="0" zip="g.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "sub/p1.bin", data: "x"}}}, "Incomplete ROM: g.zip (no p1.bin)", true},
		{"part name with a folder", `<rom index="0" zip="g.zip"><part name="sub/p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "sub/p1.bin", data: "x"}}}, "", false},
		{"part absent from present zip", `<rom index="0" zip="g.zip"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "other.bin", data: "other"}}}, "Incomplete ROM: g.zip (no p1.bin)", true},
		{"incomplete names every zip", `<rom index="0" zip="19xx.zip|qsound.zip"><part name="dl-1425.bin" crc="` + sound + `"/></rom>`,
			map[string][]member{"19xx.zip": {{name: "a", data: "a"}}, "qsound.zip": {{name: "b", data: "b"}}},
			"Incomplete ROM: 19xx.zip or qsound.zip (no dl-1425.bin)", true},

		// wrong version: blocks only when a real md5 makes Main discard the ROM
		{"wrong version, md5 None", `<rom index="0" zip="g.zip" md5="None"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "older"}}}, "Wrong ROM version: g.zip (p1.bin)", false},
		{"wrong version, no md5", `<rom index="0" zip="g.zip"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "older"}}}, "Wrong ROM version: g.zip (p1.bin)", false},
		{"wrong version, real md5", `<rom index="0" zip="g.zip" md5="0123456789abcdef0123456789abcdef"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "older"}}}, "Wrong ROM version: g.zip (p1.bin)", true},
		// Galaxian (New Invasion) from HBMame: its part CRCs went stale but its
		// md5 fits the files, so Main sends the ROM and the game plays
		{"stale crc, md5 fits", `<rom index="0" zip="g.zip" md5="` + strings.ToUpper(md5Of("newer")) + `"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "newer"}}}, "", false},
		// the md5 covers inline bytes and each part as read: from offset, cut
		// to length, repeat times, in document order
		{"md5 over inline, offset, length, repeat", `<rom index="0" zip="g.zip" md5="` + md5Of("\x01\x02bcbc"+"program") + `"><part>01 02</part>` +
			`<part name="p1.bin" crc="deadbeef" offset="0x1" length="2" repeat="2"/><interleave output="16"><part name="p2.bin" crc="` + prog + `" map="01"/></interleave></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "abcd"}, {name: "p2", data: "program"}}}, "", false},
		{"md5 counts a wrong file Main loads", `<rom index="0" zip="g.zip" md5="` + md5Of("program") + `"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "older"}}}, "Wrong ROM version: g.zip (p1.bin)", true},
		{"real md5 but no rom zip", `<rom index="0" md5="0123456789abcdef0123456789abcdef"><part name="p1.bin" zip="g.zip" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "older"}}}, "Wrong ROM version: g.zip (p1.bin)", false},
		// Bagman (set 2): the parent's same-named file comes first and Main
		// loads it; the game plays, so this only warns
		{"parent first with same name", `<rom index="0" zip="bagman.zip|bagmans4.zip" md5="none"><part name="p3.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"bagman.zip": {{name: "p3.bin", data: "parent colours"}}, "bagmans4.zip": {{name: "bagman_color_3pa2.3p", data: "program"}}},
			"Wrong ROM version: bagman.zip (p3.bin)", false},
		{"clone absent, parent has another version", `<rom index="0" zip="clone.zip|parent.zip" md5="none"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"parent.zip": {{name: "p1.bin", data: "parent"}}}, "Missing game ROM: clone.zip", false},
		{"right crc in a later zip loses to an earlier name", `<rom index="0" zip="a.zip|b.zip" md5="none"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"a.zip": {{name: "p1.bin", data: "other"}}, "b.zip": {{name: "p1.bin", data: "program"}}},
			"Wrong ROM version: a.zip (p1.bin)", false},
		{"clone split set, clone absent", `<rom index="0" zip="clone.zip|parent.zip"><part name="c1.bin" crc="` + prog + `"/><part name="shared.bin"/></rom>`,
			map[string][]member{"parent.zip": {{name: "shared.bin", data: "s"}}}, "Missing game ROM: clone.zip", true},
		{"jtbeta older key", `<rom index="0" zip="game.zip"><part name="p"/></rom><rom index="17" zip="jtbeta.zip" md5="None"><part name="beta.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"game.zip": {{name: "p", data: "p"}}, "jtbeta.zip": {{name: "beta.bin", data: "older key"}}},
			"Wrong ROM version: jtbeta.zip (beta.bin)", false},
		{"jtbeta absent", `<rom index="0" zip="game.zip"><part name="p"/></rom><rom index="17" zip="jtbeta.zip" md5="None"><part name="beta.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"game.zip": {{name: "p", data: "p"}}}, "Missing game ROM: jtbeta.zip", true},
		{"a block outranks an earlier warning", `<rom index="1" zip="w.zip"><part name="p" crc="` + prog + `"/></rom><rom index="2" zip="gone.zip"><part name="q"/></rom>`,
			map[string][]member{"w.zip": {{name: "p", data: "older"}}}, "Missing game ROM: gone.zip", true},
		{"index-0 warning layout beats a blocked one", `<rom index="0" zip="absent.zip"><part name="a"/></rom><rom index="0" zip="w.zip"><part name="p" crc="` + prog + `"/></rom>`,
			map[string][]member{"w.zip": {{name: "p", data: "older"}}}, "Wrong ROM version: w.zip (p)", false},

		// md5 over a range: streamed, and past the end reads nothing
		{"md5 reads only the part's range", `<rom index="0" zip="g.zip" md5="` + md5Of("c") + `"><part name="big" crc="deadbeef" offset="2" length="1"/></rom>`,
			map[string][]member{"g.zip": {{name: "big", data: "abcdef"}}}, "", false},
		{"md5 offset past the end", `<rom index="0" zip="g.zip" md5="` + md5Of("") + `"><part name="big" crc="deadbeef" offset="100"/></rom>`,
			map[string][]member{"g.zip": {{name: "big", data: "abcdef"}}}, "", false},

		// index-0 sections in Main's order
		{"index 00 is index 0", `<rom index="00" zip="absent.zip"><part name="a"/></rom><rom index="0" zip="present.zip"><part name="b"/></rom>`,
			map[string][]member{"present.zip": {{name: "b", data: "b"}}}, "", false},
		{"missing index is index 0", `<rom zip="absent.zip"><part name="a"/></rom><rom index="0" zip="present.zip"><part name="b"/></rom>`,
			map[string][]member{"present.zip": {{name: "b", data: "b"}}}, "", false},
		{"the core keeps the last section sent", `<rom index="0"><part>00</part></rom><rom index="0" zip="game.zip"><part name="p"/></rom>`,
			nil, "Missing game ROM: game.zip", true},
		{"a section passing its md5 ends the list", `<rom index="0" zip="good.zip" md5="` + md5Of("a") + `"><part name="a" crc="` + crcOf("a") + `"/></rom>` +
			`<rom index="0" zip="absent.zip"><part name="b"/></rom>`,
			map[string][]member{"good.zip": {{name: "a", data: "a"}}}, "", false},
		{"a section failing its md5 is discarded", `<rom index="0" zip="absent.zip" md5="0123456789abcdef0123456789abcdef"><part name="a"/></rom>` +
			`<rom index="0" zip="good.zip"><part name="b"/></rom>`,
			map[string][]member{"good.zip": {{name: "b", data: "b"}}}, "", false},
		{"only discarded sections", `<rom index="0" zip="absent.zip" md5="0123456789abcdef0123456789abcdef"><part name="a"/></rom>`,
			nil, "Missing game ROM: absent.zip", true},

		// what Main cannot read
		{"folder named like a zip", `<rom index="0" zip="g.zip"><part name="p1.bin"/></rom>`, nil, "Unreadable ROM: g.zip", true},
		{"junk named like a zip", `<rom index="0" zip="junk.zip"><part name="p1.bin"/></rom>`, nil, "Unreadable ROM: junk.zip", true},
		{"no .zip in the name", `<rom index="0" zip="roms"><part name="p1.bin"/></rom>`, nil, "Unreadable ROM: roms", true},
		{"unreadable outranks lacking", `<rom index="0" zip="junk.zip|g.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "other", data: "o"}}}, "Unreadable ROM: junk.zip", true},
		{"several unreadable", `<rom index="0" zip="junk.zip|g.zip"><part name="p1.bin"/></rom>`, nil, "Unreadable ROM: junk.zip or g.zip", true},
		{"encrypted entry", `<rom index="0" zip="g.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "x", raw: true, method: zip.Store, flags: 0x1}}}, "Unreadable ROM: g.zip", true},
		{"deflate64 entry", `<rom index="0" zip="g.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"g.zip": {{name: "p1.bin", data: "x", raw: true, method: 9}}}, "Unreadable ROM: g.zip", true},
		{"crc hit on an unextractable entry does not fall back to the name", `<rom index="0" zip="g.zip"><part name="p1.bin" crc="` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "odd.bin", data: "program", raw: true, method: 9, crc: crc32.ChecksumIEEE([]byte("program"))}, {name: "p1.bin", data: "program"}}},
			"Unreadable ROM: g.zip", true},
		{"unextractable entry skipped for the next zip", `<rom index="0" zip="a.zip|b.zip"><part name="p1.bin"/></rom>`,
			map[string][]member{"a.zip": {{name: "p1.bin", data: "x", raw: true, method: 9}}, "b.zip": {{name: "p1.bin", data: "x"}}}, "", false},

		// zip lists as written
		{"trailing space: crc finds it", `<rom index="0" zip="gorf.zip|gorfpgm1.zip "><part name="873a.x1" crc="` + prog + `"/></rom>`,
			map[string][]member{"gorfpgm1.zip": {{name: "873a.x1", data: "program"}}}, "", false},
		{"trailing space: name alone misses", `<rom index="0" zip="gorfpgm1.zip "><part name="873a.x1"/></rom>`,
			map[string][]member{"gorfpgm1.zip": {{name: "873a.x1", data: "program"}}}, "Incomplete ROM: gorfpgm1.zip (no 873a.x1)", true},
		{"empty entry skipped", `<rom index="0" zip="a.zip||b.zip"><part name="p"/></rom>`,
			map[string][]member{"b.zip": {{name: "p", data: "p"}}}, "", false},
		{"crc uppercase", `<rom index="0" zip="g.zip"><part name="x" crc="` + strings.ToUpper(prog) + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "renamed", data: "program"}}}, "", false},
		{"crc with 0x and spaces", `<rom index="0" zip="g.zip"><part name="x" crc="  0x` + prog + `"/></rom>`,
			map[string][]member{"g.zip": {{name: "renamed", data: "program"}}}, "", false},
		{"games-relative zip", `<rom index="0" zip="/hbmame/hb.zip"><part name="p"/></rom>`, nil, "Missing game ROM: /hbmame/hb.zip", true},
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
			switch tc.name {
			case "folder named like a zip":
				os.MkdirAll(filepath.Join(mame, "g.zip", "p1.bin"), 0755)
			case "junk named like a zip", "unreadable outranks lacking":
				putFile(t, filepath.Join(mame, "junk.zip"), "not a zip")
			case "several unreadable":
				putFile(t, filepath.Join(mame, "junk.zip"), "not a zip")
				putFile(t, filepath.Join(mame, "g.zip"), "not a zip")
			case "no .zip in the name":
				putFile(t, filepath.Join(mame, "roms"), "a file")
			}
			got := NewROMCheck(card).Check("_Arcade/game.mra", true)
			if got.Text != tc.want || got.Block != tc.block {
				t.Fatalf("got %q block=%v; want %q block=%v", got.Text, got.Block, tc.want, tc.block)
			}
		})
	}
}

func TestMainHexAndNumbers(t *testing.T) {
	for in, want := range map[string]string{
		"": "", "0a,FF": "\x0a\xff", "12\n34\t56": "\x12\x34\x56", "abc": "\xab\x0c", "0 1": "\x09\x01",
	} {
		if got := string(mainHex(in)); got != want {
			t.Errorf("mainHex(%q) = %q; want %q", in, got, want)
		}
	}
	for in, want := range map[string]int{"": 0, "16": 16, "0x10": 16, "010": 8, " 7z": 7, "09": 0, "-1": -1} {
		if got := parseCUL(in); got != want {
			t.Errorf("parseCUL(%q) = %d; want %d", in, got, want)
		}
	}
}

func TestParseCRC(t *testing.T) {
	for in, want := range map[string]uint32{
		"": 0, "zz": 0, "1a2B3c4D": 0x1a2b3c4d, "0x1a2b3c4d": 0x1a2b3c4d, " \t0Xff": 0xff,
		"12 34": 0x12, "123456789": 0xffffffff, "-1": 0xffffffff, "0x": 0,
	} {
		if got := parseCRC(in); got != want {
			t.Errorf("parseCRC(%q) = %#x; want %#x", in, got, want)
		}
	}
}

func TestROMStorageAndFreshCheck(t *testing.T) {
	media := t.TempDir()
	card := filepath.Join(media, "fat")
	putFile(t, filepath.Join(card, "game.mra"), `<misterromdescription><rom index="4" zip="jpark.zip"><part name="a"/></rom></misterromdescription>`)
	putZip(t, filepath.Join(card, "games/mame/jpark.zip"), member{name: "a", data: "a"})
	if err := os.MkdirAll(filepath.Join(media, "usb0/games/mame"), 0755); err != nil {
		t.Fatal(err)
	}
	check := NewROMCheck(card)
	if got := check.Check("game.mra", true); got.Text != "Missing game ROM: jpark.zip" {
		t.Fatalf("must use USB directory, got %q", got.Text)
	}
	putZip(t, filepath.Join(media, "usb0/games/mame/jpark.zip"), member{name: "a", data: "a"})
	if got := check.Check("game.mra", true); got.Text != "" {
		t.Fatalf("fresh Start check must see new archive: %s", got.Text)
	}
}

// A mame folder at the card's root stops Main finding any zip, even one
// inside it (tried on a DE10): every MRA that needs a zip says so, and
// removing the folder clears it.
func TestROMRootMameFolder(t *testing.T) {
	card := filepath.Join(t.TempDir(), "fat")
	putFile(t, filepath.Join(card, "_Arcade/game.mra"), `<misterromdescription><rom index="0" zip="g.zip"><part name="p"/></rom></misterromdescription>`)
	putFile(t, filepath.Join(card, "_Arcade/inline.mra"), `<misterromdescription><rom index="0"><part>00</part></rom></misterromdescription>`)
	putZip(t, filepath.Join(card, "games/mame/g.zip"), member{name: "p", data: "p"})
	putZip(t, filepath.Join(card, "mame/g.zip"), member{name: "p", data: "p"})
	c := NewROMCheck(card)
	want := "MiSTer can't load ROMs while " + filepath.ToSlash(filepath.Join(card, "mame")) + " exists"
	if got := c.Check("_Arcade/game.mra", true); got.Text != want || !got.Block {
		t.Fatalf("root mame: got %+v; want %q blocking", got, want)
	}
	if got := c.Check("_Arcade/inline.mra", true); got.Text != "" {
		t.Fatalf("an MRA without zips is unaffected: %q", got.Text)
	}
	if err := os.RemoveAll(filepath.Join(card, "mame")); err != nil {
		t.Fatal(err)
	}
	if got := c.Check("_Arcade/game.mra", true); got.Text != "" {
		t.Fatalf("games/mame used once the root folder is gone: %q", got.Text)
	}
}

// Start reads a zip again even when its replacement has the same size and
// time, which Details' cache would not notice.
func TestROMFreshRereadsZip(t *testing.T) {
	card := filepath.Join(t.TempDir(), "fat")
	zp := filepath.Join(card, "games/mame/g.zip")
	putFile(t, filepath.Join(card, "_Arcade/game.mra"), `<misterromdescription><rom index="0" zip="g.zip"><part name="aaaa"/></rom></misterromdescription>`)
	putZip(t, zp, member{name: "aaaa", data: "x"})
	c := NewROMCheck(card)
	if got := c.Check("_Arcade/game.mra", true); got.Text != "" {
		t.Fatalf("present part reported: %q", got.Text)
	}
	before, err := os.Stat(zp)
	if err != nil {
		t.Fatal(err)
	}
	putZip(t, zp, member{name: "bbbb", data: "x"})
	os.Chtimes(zp, before.ModTime(), before.ModTime())
	if after, err := os.Stat(zp); err != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("replacement not the same size and time: %v", err)
	}
	if got := c.Check("_Arcade/game.mra", true); got.Text != "Incomplete ROM: g.zip (no aaaa)" {
		t.Fatalf("Start kept the old zip: %q", got.Text)
	}
}

// A zip rewritten in place is read again, and so is an edited MRA; a
// shifted time is enough, whatever the sizes.
func TestROMCachesFollowChanges(t *testing.T) {
	card := filepath.Join(t.TempDir(), "fat")
	mra := filepath.Join(card, "_Arcade/game.mra")
	zp := filepath.Join(card, "games/mame/g.zip")
	putFile(t, mra, `<misterromdescription><rom index="0" zip="g.zip"><part name="aaaa"/></rom></misterromdescription>`)
	putZip(t, zp, member{name: "aaaa", data: "x"})
	check := NewROMCheck(card)
	if got := check.Check("_Arcade/game.mra", true); got.Text != "" {
		t.Fatalf("present part reported: %q", got.Text)
	}
	putZip(t, zp, member{name: "bbbb", data: "x"}) // same size
	later := time.Now().Add(time.Minute)
	os.Chtimes(zp, later, later)
	if got := check.Check("_Arcade/game.mra", true); got.Text != "Incomplete ROM: g.zip (no aaaa)" {
		t.Fatalf("rewritten zip not read again: %q", got.Text)
	}
	putFile(t, mra, `<misterromdescription><rom index="0" zip="g.zip"><part name="bbbb"/></rom></misterromdescription>`)
	os.Chtimes(mra, later, later)
	if got := check.Check("_Arcade/game.mra", true); got.Text != "" {
		t.Fatalf("edited MRA not read again: %q", got.Text)
	}
}

// Details never waits long: a slow check draws no line at first, lands
// with a Ready signal, and a stale answer is shown while it is renewed.
func TestROMCheckDetailsInBackground(t *testing.T) {
	c := NewROMCheck(t.TempDir())
	c.wait = time.Millisecond
	release := make(chan struct{})
	answer := ROMResult{"Missing game ROM: x.zip", true}
	c.run = func(string) ROMResult { <-release; return answer }
	if got := c.Check("g.mra", false); got != (ROMResult{}) {
		t.Fatalf("slow first check returned %+v", got)
	}
	close(release)
	select {
	case <-c.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("no Ready signal after the late result")
	}
	if got := c.Check("g.mra", false); got != answer {
		t.Fatalf("cached result %+v", got)
	}
	// stale: returned at once while a renewed check runs
	c.mu.Lock()
	e := c.results["g.mra"]
	e.at = e.at.Add(-time.Minute)
	c.results["g.mra"] = e
	c.mu.Unlock()
	fixed := make(chan struct{})
	c.run = func(string) ROMResult { <-fixed; return ROMResult{} }
	if got := c.Check("g.mra", false); got != answer {
		t.Fatalf("stale result not reused: %+v", got)
	}
	close(fixed)
	select {
	case <-c.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("no Ready signal after the answer changed")
	}
	if got := c.Check("g.mra", false); got != (ROMResult{}) {
		t.Fatalf("renewed result %+v", got)
	}
	if got := c.Check("readme.txt", false); got != (ROMResult{}) {
		t.Fatal("non-MRA checked")
	}
}

// A quick first check answers in time and signals nothing.
func TestROMCheckQuickNoSignal(t *testing.T) {
	c := NewROMCheck(t.TempDir())
	c.wait = 5 * time.Second
	c.run = func(string) ROMResult { return ROMResult{"Missing game ROM: x.zip", true} }
	if got := c.Check("g.mra", false); got.Text == "" {
		t.Fatal("quick check not waited for")
	}
	time.Sleep(10 * time.Millisecond)
	select {
	case <-c.Ready():
		t.Fatal("Ready fired for an answer Details already drew")
	default:
	}
}
