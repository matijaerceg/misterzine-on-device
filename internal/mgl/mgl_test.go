package mgl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample(core, file string) string {
	return `<mistergamedescription><rbf>` + core + `</rbf>` + file + `</mistergamedescription>`
}

func TestGeneratorConventions(t *testing.T) {
	for _, target := range []string{
		"mslug.neo", "Darksoft/mslug.zip", "mslug", "Arcade/mslug.rom",
		"../../../../../media/fat/games/NEOGEO/mslug.neo",
		"../../../../../media/usb0/games/NEOGEO/mslug",
		"../../../../../media/fat/games/NEOGEO/Collection.zip/mslug.neo",
	} {
		t.Run(target, func(t *testing.T) {
			x := sample("_Console/NeoGeo", `<setname same_dir="1">custom-settings</setname><file delay="1" type="f" index="1" path="`+target+`"/>`)
			d, err := Parse(strings.NewReader(x))
			if err != nil {
				t.Fatal(err)
			}
			if ok, why := d.Cartridge(); !ok {
				t.Fatal(why)
			}
			if d.Files[0].Path != target || d.Setname.Name != "custom-settings" || d.Setname.SameDir != "1" {
				t.Fatalf("lost launch data: %+v", d)
			}
		})
	}
	for _, core := range []string{"_Console/NeoGeo", "_Console/_Handheld/NeoGeo_20261001.rbf", "_LLAPI/NeoGeo_LLAPI", "_YC/NeoGeoYC"} {
		d, err := Parse(strings.NewReader(sample(core, `<file delay="1" type="f" index="1" path="mslug.neo"/>`)))
		if err != nil {
			t.Fatal(err)
		}
		if ok, why := d.Cartridge(); !ok {
			t.Fatalf("%s: %s", core, why)
		}
	}
}

func TestExcludesOtherSystemsAndAmbiguousLaunches(t *testing.T) {
	for name, x := range map[string]string{
		"other core":        sample("_Console/SNES", `<file type="f" index="1" path="mslug.neo"/>`),
		"lookalike core":    sample("_Console/NeoGeoPocket", `<file type="f" index="1" path="mslug.neo"/>`),
		"CD":                sample("_Console/NeoGeo", `<file type="s" index="1" path="mslug.chd"/>`),
		"CD disguised slot": sample("_Console/NeoGeo", `<file type="f" index="1" path="mslug.chd"/>`),
		"core only":         sample("_Console/NeoGeo", ""),
		"multiple files":    sample("_Console/NeoGeo", `<file type="f" index="1" path="mslug.neo"/><file type="f" index="1" path="aof.neo"/>`),
		"wrong slot":        sample("_Console/NeoGeo", `<file type="f" index="0" path="mslug.neo"/>`),
		"missing method":    sample("_Console/NeoGeo", `<file index="1" path="mslug.neo"/>`),
		"nested impostor":   sample("_Console/NeoGeo", `<extension><file type="f" index="1" path="mslug.neo"/></extension>`),
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Parse(strings.NewReader(x))
			if err != nil {
				t.Fatal(err)
			}
			if ok, _ := d.Cartridge(); ok {
				t.Fatal("wrongly included", x)
			}
		})
	}
}

func TestMalformedXMLAndSizeLimits(t *testing.T) {
	for _, x := range []string{
		"", "<mistergamedescription>", `<mra><rbf>NeoGeo</rbf></mra>`,
		sample("NeoGeo", "") + sample("NeoGeo", ""),
		sample("NeoGeo", `<rbf>SNES</rbf>`),
		sample("NeoGeo", `<setname>a</setname><setname>b</setname>`),
		sample("NeoGeo", `<file path="mslug.neo"/>`) + "trailing",
		`<!DOCTYPE example>` + sample("NeoGeo", ""),
		sample("NeoGeo", `<file path="bad&#10;path.neo"/>`),
		sample("NeoGeo", `<file path="broken"quote.neo"/>`),
		sample("NeoGeo", `<file path="mslug.neo" path="unknown.neo"/>`),
		strings.Repeat(" ", MaxBytes+1),
	} {
		if _, err := Parse(strings.NewReader(x)); err == nil {
			t.Fatalf("accepted malformed input %.120q", x)
		}
	}
}

func TestLegacyAmpersandsAndEscaping(t *testing.T) {
	for _, escaped := range []bool{false, true} {
		p := "Run & Gun/mslug.neo"
		if escaped {
			p = "Run &amp; Gun/mslug.neo"
		}
		d, err := Parse(strings.NewReader(sample("NeoGeo", `<file type="f" index="1" path="`+p+`"/>`)))
		if err != nil {
			t.Fatal(err)
		}
		if d.Files[0].Path != "Run & Gun/mslug.neo" || d.LegacyAmpersand == escaped {
			t.Fatalf("wrong decoding: %+v", d)
		}
	}
}

func pilot(t *testing.T) *Catalogue {
	t.Helper()
	f, err := os.Open("../../testdata/mgl/arcade-pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := ReadCatalogue(f)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFixturesAndNoCatalogue(t *testing.T) {
	r, err := Scan("../../testdata/mgl/card", pilot(t))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, e := range r.Entries {
		counts[e.Decision]++
	}
	if len(r.Rows) != 5 || counts["included"] != 5 || counts["excluded"] != 3 || counts["candidate"] != 1 || len(r.Problems) != 0 {
		t.Fatalf("unexpected report: %+v", r)
	}
	for _, row := range r.Rows {
		if row.Date != "" || row.Updated != "" || !row.IsArcade() || !row.IsLocal() || !strings.HasSuffix(row.MRA, ".mgl") {
			t.Fatalf("invented metadata: %+v", row)
		}
	}
	r, err = Scan("../../testdata/mgl/card", nil)
	if err != nil || len(r.Rows) != 0 {
		t.Fatalf("no catalogue must not claim arcade identity: %+v %v", r, err)
	}
}

func TestIdentityDoesNotTrustSetnameOrShortcutTitle(t *testing.T) {
	d, err := Parse(strings.NewReader(sample("NeoGeo", `<setname same_dir="1">mslug</setname><file type="f" index="1" path="homebrew/unknown.neo"/>`)))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pilot(t).Match(d); ok {
		t.Fatal("setname is a configuration name, not proof of game identity")
	}
	for _, x := range []string{`[{"id":"x","title":"X","aliases":["same"]},{"id":"y","title":"Y","aliases":["same"]}]`, `[] trailing`, `[{"id":"x","title":"X","typo":true}]`} {
		if _, err := ReadCatalogue(strings.NewReader(x)); err == nil {
			t.Fatal("accepted ambiguous/invalid catalogue", x)
		}
	}
}

func TestScanCopiesRenameAndSettings(t *testing.T) {
	root := t.TempDir()
	put := func(p, x string) {
		t.Helper()
		p = filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(x), 0644); err != nil {
			t.Fatal(err)
		}
	}
	x := sample("_Console/NeoGeo", `<file type="f" index="1" path="mslug.neo"/>`)
	put("_@Favorites/One.mgl", x)
	put("_Games/_NEOGEO/Copy.mgl", x)
	put("_Arcade/Settings.mgl", strings.Replace(x, "</mistergamedescription>", `<reset delay="2" hold="1"/></mistergamedescription>`, 1))
	put("games/NEOGEO/Not a menu.mgl", x)
	put("_Games/plain-subfolder/Hidden.mgl", x)
	r, err := Scan(root, pilot(t))
	if err != nil || len(r.Rows) != 2 || len(r.Entries) != 3 {
		t.Fatalf("bad menu discovery: %+v %v", r, err)
	}
	key := r.Rows[0].K
	if len(r.Alternatives[key]) != 1 {
		t.Fatalf("copy not associated: %+v", r)
	}
	if err := os.Rename(filepath.Join(root, "_@Favorites/One.mgl"), filepath.Join(root, "_@Favorites/Renamed.mgl")); err != nil {
		t.Fatal(err)
	}
	r, err = Scan(root, pilot(t))
	if err != nil || r.Rows[0].K != key {
		t.Fatal("renaming shortcut lost identity")
	}
	got, err := os.ReadFile(filepath.Join(root, "_@Favorites/Renamed.mgl"))
	if err != nil || string(got) != x {
		t.Fatal("scan changed source MGL")
	}
	if err := os.Remove(filepath.Join(root, "_@Favorites/Renamed.mgl")); err != nil {
		t.Fatal(err)
	}
	r, err = Scan(root, pilot(t))
	found := false
	for _, row := range r.Rows {
		if row.K == key && row.MRA == "_Games/_NEOGEO/Copy.mgl" {
			found = true
		}
	}
	if err != nil || !found {
		t.Fatal("remaining copy did not take over")
	}
}

func TestScanDoesNotFollowLinks(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	x := sample("NeoGeo", `<file type="f" index="1" path="mslug.neo"/>`)
	if err := os.WriteFile(filepath.Join(outside, "game.mgl"), []byte(x), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "_Linked")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := os.Symlink(filepath.Join(outside, "game.mgl"), filepath.Join(root, "linked.mgl")); err != nil {
		t.Fatal(err)
	}
	r, err := Scan(root, pilot(t))
	if err != nil || len(r.Entries) != 0 {
		t.Fatalf("followed links: %+v %v", r, err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add(sample("_Console/NeoGeo", `<file type="f" index="1" path="mslug.neo"/>`))
	f.Add("<broken>")
	f.Fuzz(func(t *testing.T, x string) {
		d, err := Parse(strings.NewReader(x))
		if err == nil {
			d.Cartridge()
		}
	})
}
