package scan

import (
	"encoding/json"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGalagaFamily(t *testing.T) {
	sets := []string{"galagaef", "galagads", "galagamf", "galaga", "galagao", "galagost", "galagosb", "galgaxin", "galaped", "gatsbee", "vgalaga", "vgalagaf", "vgalagam"}
	var alts []Alt
	for _, set := range sets {
		a, ok := parseMRA(strings.NewReader(`<misterromdescription><setname>` + set + `</setname><parent>galaga</parent><rbf>galaga</rbf><rom index="0" zip="/hbmame/galaga.zip|namco51.zip|namco54.zip"><part name="x"/></rom></misterromdescription>`))
		if !ok || a.Parent != "galaga" {
			t.Fatalf("header: %+v", a)
		}
		a.Path = "_Arcade/_alternatives/_Galaga/" + set + ".mra"
		alts = append(alts, a)
	}
	row := data.Row{K: "galagamw", Base: "Arcade", Core: "galaga", SN: "galagamw", Family: "galaga"}
	if got := Alternatives(alts, &row); len(got) != 13 {
		t.Fatalf("Galaga alternatives: %v", got)
	}
	row.Family = ""
	if got := Alternatives(alts, &row); len(got) != 0 {
		t.Fatal("fixture must reproduce old failure", got)
	}
}

func TestFamilyMatchingBoundaries(t *testing.T) {
	row := data.Row{Base: "Arcade", Core: "shared", SN: "clone", Family: "parent"}
	alts := []Alt{
		{Path: "z", RBF: "shared", Parent: "parent"},
		{Path: "b", RBF: "shared", Zips: []string{`\HBMAME\PARENT.ZIP`}},
		{Path: "a", RBF: "shared", Setname: "clone"},
		{Path: "a", RBF: "shared", Setname: "clone"},
		{Path: "bios", RBF: "shared", Zips: []string{"bios.zip"}},
		{Path: "wrong-core", RBF: "unrelated", Parent: "parent"},
		{Path: "_parent/unrelated.mra", RBF: "shared", Setname: "other", Parent: "other"},
		{Path: "bad-parent", RBF: "shared", Parent: "/parent"},
	}
	if got := Alternatives(alts, &row); !reflect.DeepEqual(got, []string{"a", "b", "z"}) {
		t.Fatal(got)
	}
	row.Family = ""
	if got := Alternatives(alts, &row); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatal(got)
	}
}

func TestInstalledFamilyFallbackAndCache(t *testing.T) {
	card := t.TempDir()
	p := filepath.Join(card, "main.mra")
	write := func(parent string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(`<misterromdescription><rbf>game</rbf><setname>clone</setname><parent>`+parent+`</parent><rom index="0" zip="bios.zip"/></misterromdescription>`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	rows := []data.Row{{K: "game", Base: "Arcade", Core: "game", SN: "clone", MRA: "main.mra"}}
	alts := []Alt{{Path: "parent", RBF: "game", Parent: "parent"}, {Path: "other", RBF: "game", Parent: "other"}, {Path: "clone", RBF: "game", Setname: "clone"}, {Path: "bios", RBF: "game", Zips: []string{"bios.zip"}}}
	var c FamilyCache
	write("parent")
	want := []string{"clone", "parent"}
	if got := c.Resolve(card, alts, rows)["game"]; !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	if got := c.Resolve(card, alts, rows)["game"]; !reflect.DeepEqual(got, want) || len(c.headers) != 1 {
		t.Fatal(got)
	}
	write("other")
	stamp := time.Now().Add(time.Second)
	os.Chtimes(p, stamp, stamp)
	if got := c.Resolve(card, alts, rows)["game"]; !reflect.DeepEqual(got, []string{"clone", "other"}) {
		t.Fatal(got)
	}
	rows[0].Family = "parent" // catalog wins over conflicting installed metadata
	if got := c.Resolve(card, alts, rows)["game"]; !reflect.DeepEqual(got, want) || len(c.headers) != 0 {
		t.Fatal(got)
	}
	rows[0].Family = ""
	os.Remove(p)
	if got := c.Resolve(card, alts, rows)["game"]; !reflect.DeepEqual(got, []string{"clone"}) {
		t.Fatal(got)
	}
	os.WriteFile(p, []byte("broken"), 0600)
	c.Resolve(card, alts, rows)
	if len(c.headers) != 0 {
		t.Fatal("failed parse cached")
	}
	write("parent")
	if got := c.Resolve(card, alts, rows)["game"]; !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

func TestLegacyAlternativeCacheGainsParent(t *testing.T) {
	for _, version := range []int{2, 3} {
		card := fakeCard(t)
		dir := filepath.Join(card, "_Arcade", "_alternatives", "_1942")
		p := filepath.Join(dir, "1942 (set 2).mra")
		os.WriteFile(p, []byte(`<misterromdescription><rbf>jt1942</rbf><setname>1942a</setname><parent>1942</parent><rom index="0" zip="1942.zip"/></misterromdescription>`), 0600)
		st, _ := os.Stat(dir)
		cache := filepath.Join(card, "cache.json")
		raw, _ := json.Marshal(map[string]altDir{"_1942": {Version: version, Mtime: st.ModTime().UnixNano(), Alts: []Alt{{Path: "stale", RBF: "jt1942"}}}})
		os.WriteFile(cache, raw, 0600)
		alts, _, err := ScanAlternativesWithError(card, cache)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, a := range alts {
			if a.Parent == "1942" {
				found = true
			}
			if a.Path == "stale" {
				t.Fatal("legacy entry reused")
			}
		}
		if !found {
			t.Fatal("parent missing after upgrade")
		}
	}
}

func TestCatalogMembersRecoverParentlessMRA(t *testing.T) {
	row := data.Row{Base: "Arcade", Core: "st-v", SN: "decathlt", Family: "decathlt", FamilySets: []string{"decathlto"}}
	alts := []Alt{{Path: "older", RBF: "st-v", Setname: "decathlto"}, {Path: "unrelated", RBF: "st-v", Setname: "dnmtdeka"}}
	if got := Alternatives(alts, &row); !reflect.DeepEqual(got, []string{"older"}) {
		t.Fatal(got)
	}
	row.Family = ""
	if got := Alternatives(alts, &row); len(got) != 0 {
		t.Fatal("unanchored aliases used", got)
	}
}

func TestAlternativeCacheSeesInPlaceParentChange(t *testing.T) {
	card := fakeCard(t)
	dir := filepath.Join(card, "_Arcade", "_alternatives", "_1942")
	p := filepath.Join(dir, "1942 (set 2).mra")
	write := func(parent string) {
		os.WriteFile(p, []byte(`<misterromdescription><rbf>jt1942</rbf><setname>1942a</setname><parent>`+parent+`</parent><rom index="0" zip="1942.zip"/></misterromdescription>`), 0600)
	}
	write("1942")
	st, _ := os.Stat(dir)
	cache := filepath.Join(card, "cache.json")
	ScanAlternatives(card, cache)
	write("1943")
	stamp := time.Now().Add(time.Second)
	os.Chtimes(p, stamp, stamp)
	os.Chtimes(dir, st.ModTime(), st.ModTime())
	alts := ScanAlternatives(card, cache)
	for _, a := range alts {
		if strings.Contains(a.Path, "_1942/") && a.Parent != "1943" {
			t.Fatalf("stale header: %+v", a)
		}
	}
	os.Remove(p)
	if got := ScanAlternatives(card, cache); len(got) != 1 {
		t.Fatal("deleted MRA retained", got)
	}
}

// Some standalone hacks have no setname and embed a large ROM body. Their
// explicit parent is sufficient header identity; never read the bulky body.
func TestParentOnlyHeaderStopsBeforeEmbeddedROM(t *testing.T) {
	card := t.TempDir()
	p := filepath.Join(card, "main.mra")
	raw := `<misterromdescription><setname></setname><parent>puckman</parent><rbf>pacman</rbf><rom index="0"><part>` + strings.Repeat("00 ", 30000) + `</part></rom></misterromdescription>`
	os.WriteFile(p, []byte(raw), 0600)
	a, ok := ParseMRAHeader(p)
	if !ok || a.Parent != "puckman" {
		t.Fatalf("parent-only header rejected: %+v", a)
	}
	var c FamilyCache
	got := c.Resolve(card, []Alt{{Path: "alt", RBF: "pacman", Parent: "puckman"}}, []data.Row{{K: "hack", Base: "Arcade", Core: "pacman", MRA: "main.mra"}})
	if !reflect.DeepEqual(got["hack"], []string{"alt"}) {
		t.Fatal(got)
	}
}
