package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// A fan patch built with arcadepatches.com's kit lands under
// _Arcade/_Arcade Patches/..., carries a new setname and no <parent>. The
// site lists that setname in the base game's family_sets, so the file must
// file under the game as a version, not stand as a local game of its own.
func TestDiscoverLocalArcadePatchJoinsFamily(t *testing.T) {
	card := t.TempDir()
	mk := func(rel, content string) {
		t.Helper()
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk("_Arcade/cores/jtcps2_20260901.rbf", "x")
	mk("_Arcade/Vampire Hunter 2 (Japan).mra", mra("Vampire Hunter 2 (Japan)", "vhunt2", "jtcps2", ""))
	const patched = "_Arcade/_Arcade Patches/_Restorations/Night Warriors 2 (USA) [English Translation].mra"
	mk(patched, `<misterromdescription><name>Night Warriors 2 (USA)</name><setname>nwarr2en</setname><rbf>jtcps2</rbf>`+
		`<rom index="0" zip="nwarr2en.zip|vhunt2.zip"><part name="x"/></rom></misterromdescription>`)
	catalogue := []data.Row{{K: "vhunt2", Title: "Vampire Hunter 2", Base: "Arcade", Core: "jtcps2", SN: "vhunt2", Family: "vhunt2",
		FamilySets: []string{"vhunt2", "nwarr2en"}, MRA: "_Arcade/Vampire Hunter 2 (Japan).mra",
		PT: []data.Patch{{T: "translation", N: "Night Warriors 2", SN: "nwarr2en"}}}}
	status := []data.Status{data.StatusCurrent}
	alts, _, _ := ScanAlternativesWithError(card, "")
	var fc FamilyCache
	resolved := fc.Resolve(card, alts, catalogue)
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), status, alts, AttachedPaths(resolved), false)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(res.Rows) != 0 {
		t.Fatalf("patched file became a local row: %+v", res.Rows)
	}
	if v := res.Versions["vhunt2"]; len(v) != 1 || v[0] != patched {
		t.Fatalf("versions = %v, want the patched file", v)
	}
	if res.VersionCores[patched] != "jtcps2" {
		t.Fatalf("version core = %q", res.VersionCores[patched])
	}
	merged := map[string][]string{}
	for k, v := range resolved {
		merged[k] = v
	}
	MergeVersions(merged, res.Versions)
	found := false
	for _, p := range merged["vhunt2"] {
		found = found || p == patched
	}
	if !found {
		t.Fatalf("launch list for vhunt2 lacks the patch: %v", merged["vhunt2"])
	}
}
