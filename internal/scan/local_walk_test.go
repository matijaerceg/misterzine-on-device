package scan

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func mra(name, setname, rbf, parent string) string {
	return `<misterromdescription><name>` + name + `</name><setname>` + setname + `</setname>` +
		`<parent>` + parent + `</parent><rbf>` + rbf + `</rbf><rotation>vertical (cw)</rotation>` +
		`<rom index="0" zip="` + setname + `.zip"><part name="x"/></rom></misterromdescription>`
}

// localCard is fakeCard plus the files the local walk is about.
func localCard(t *testing.T) string {
	t.Helper()
	card := fakeCard(t)
	mk := func(rel string, content string) {
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}
	mk("_Arcade/Galaxian Local (World).mra", mra("Galaxian Local", "galaxloc", "galaxian", "galaxloc"))
	mk("_Arcade/_Organized/_1 A-E/Galaxian Local (World).mra", mra("Galaxian Local", "galaxloc", "galaxian", "galaxloc"))
	mk("_Arcade/_Extra/Orphan (set 2).mra", mra("Orphan", "orphan", "defender", "orphan"))
	mk("_Arcade/_Extra/deeper/Orphan.mra", mra("Orphan", "orphan", "defender", "orphan"))
	mk("_Arcade/_Extra/Colony Clone.mra", mra("Colony 7 (clone)", "colony7x", "defender", "colony7"))
	mk("_Arcade/_Extra/Colony Known.mra", mra("Colony 7 (known)", "colony7a", "defender", ""))
	mk("_Arcade/_Extra/nameless.mra", `<misterromdescription><rbf>defender</rbf><rom index="0" zip="q.zip"><part name="x"/></rom></misterromdescription>`)
	mk("_Arcade/_Extra/broken.mra", "not xml at all")
	mk("_Arcade/_Extra/.hidden/Ghost.mra", mra("Ghost", "ghost", "defender", "ghost"))
	mk("_Arcade/_alternatives/_Orphan/Orphan (alt).mra", mra("Orphan alt", "orphana", "defender", "orphan"))
	return card
}

func TestScanArcadeMRAsSkipsOrganizedAndAlternatives(t *testing.T) {
	card := localCard(t)
	alts, skipped, err := ScanArcadeMRAs(card, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range alts {
		got = append(got, a.Path)
	}
	want := []string{
		"_Arcade/Galaxian Local (World).mra",
		"_Arcade/_Extra/Colony Clone.mra",
		"_Arcade/_Extra/Colony Known.mra",
		"_Arcade/_Extra/Orphan (set 2).mra",
		"_Arcade/_Extra/deeper/Orphan.mra",
		"_Arcade/_Extra/nameless.mra",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("walked %v\nwant   %v", got, want)
	}
	// fakeCard's two stub main MRAs have no <rbf>, plus the broken one.
	if len(skipped) != 3 {
		t.Fatalf("skipped %+v", skipped)
	}
	if a := alts[0]; a.Name != "Galaxian Local" || a.Rotation != "vertical (cw)" {
		t.Fatalf("header not kept: %+v", a)
	}
	if err := os.Symlink(filepath.Join(card, "_Arcade", "_Extra"), filepath.Join(card, "_Arcade", "_Linked")); err == nil {
		again, _, _ := ScanArcadeMRAs(card, "")
		if len(again) != len(alts) {
			t.Fatalf("symlinked folder walked: %d files", len(again))
		}
	}
	if alts, _, err := ScanArcadeMRAs(t.TempDir(), ""); err != nil || len(alts) != 0 {
		t.Fatalf("card without _Arcade: %v %v", alts, err)
	}
}

func TestLocalCacheWarm(t *testing.T) {
	card := localCard(t)
	cache := filepath.Join(card, "cache", "local.json")
	cold, _, err := ScanArcadeMRAs(card, cache)
	if err != nil {
		t.Fatal(err)
	}
	// Make every file unreadable in place: a warm run must not need them.
	warm, _, err := ScanArcadeMRAs(card, cache)
	if err != nil || !reflect.DeepEqual(cold, warm) {
		t.Fatalf("warm run differs: %v", err)
	}
	b, _ := os.ReadFile(cache)
	if !strings.Contains(string(b), `"version":5`) || !strings.Contains(string(b), `"_Arcade/_Extra/deeper"`) {
		t.Fatalf("cache: %s", b)
	}
	// A new file in an existing folder is noticed (the folder's mtime moves).
	p := filepath.Join(card, "_Arcade", "_Extra", "New.mra")
	os.WriteFile(p, []byte(mra("New", "newgame", "defender", "newgame")), 0644)
	dir := filepath.Join(card, "_Arcade", "_Extra")
	st, _ := os.Stat(dir)
	os.Chtimes(dir, st.ModTime().Add(1e9), st.ModTime().Add(1e9))
	after, _, _ := ScanArcadeMRAs(card, cache)
	if len(after) != len(cold)+1 {
		t.Fatalf("new file missed: %d vs %d", len(after), len(cold))
	}
}

func TestDiscoverLocalMatchesCatalogue(t *testing.T) {
	card := localCard(t)
	catalogue := []data.Row{
		{K: "colony7", Title: "Colony 7", Base: "Arcade", Core: "defender", SN: "colony7", Family: "colony7", FamilySets: []string{"colony7a"}, MRA: "_Arcade/Colony 7 (Set 1).mra"},
		{K: "1942", Title: "1942", Base: "Arcade", Core: "jt1942", SN: "1942", MRA: "_Arcade/1942 (Revision B).mra"},
		{K: "C64", Title: "Commodore 64", Base: "Computer", Core: "C64", SN: "orphan"}, // not arcade: no identity
	}
	alts, _, _ := ScanAlternativesWithError(card, "")
	var fc FamilyCache
	attached := AttachedPaths(fc.Resolve(card, alts, catalogue))
	res := DiscoverLocal(card, "", catalogue, alts, attached, false)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	var keys []string
	for _, r := range res.Rows {
		keys = append(keys, r.K)
	}
	// colony7x (parent colony7) and colony7a (known clone) are catalogue
	// games; the alternatives' own files are attached; nameless.mra takes
	// its filename as setname.
	if want := []string{"local:galaxloc", "local:nameless", "local:orphan"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("rows %v, want %v", keys, want)
	}
	if res.Files != 6 {
		t.Fatalf("files %d", res.Files)
	}
	o := res.Rows[2]
	if o.MRA != "_Arcade/_Extra/deeper/Orphan.mra" || o.Title != "Orphan" || o.Core != "defender" || o.SN != "orphan" {
		t.Fatalf("primary copy: %+v", o)
	}
	// The other copy and the _alternatives file for the same family follow it.
	if want := []string{"_Arcade/_Extra/Orphan (set 2).mra", "_Arcade/_alternatives/_Orphan/Orphan (alt).mra"}; !reflect.DeepEqual(res.Alts["local:orphan"], want) {
		t.Fatalf("alts %v", res.Alts["local:orphan"])
	}
	if res.Rows[1].Title != "nameless" || res.Rows[1].MRA != "_Arcade/_Extra/nameless.mra" {
		t.Fatalf("nameless: %+v", res.Rows[1])
	}
	if res.Rows[0].Img != "" || res.Rows[0].ImgSlots != nil {
		t.Fatal("no image service: no picture requested")
	}

	with := DiscoverLocal(card, "", catalogue, alts, attached, true)
	g := with.Rows[0]
	if g.Img != "galaxloc" || !reflect.DeepEqual(g.ImgSlots, []string{SlotLocalSnap}) || g.ImgW != 3 || g.ImgH != 4 {
		t.Fatalf("service picture: %+v", g)
	}
}

func TestDiscoverLocalDedupeOrder(t *testing.T) {
	card := t.TempDir()
	mk := func(rel string) {
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(mra("Same", "same", "core", "")), 0644)
	}
	mk("_Arcade/_alternatives/_Same/a.mra")
	mk("_Arcade/_Long folder name/Same.mra")
	mk("_Arcade/_S/Same.mra")
	alts, _, _ := ScanAlternativesWithError(card, "")
	res := DiscoverLocal(card, "", nil, alts, nil, false)
	if len(res.Rows) != 1 || res.Rows[0].MRA != "_Arcade/_S/Same.mra" {
		t.Fatalf("rows %+v", res.Rows)
	}
	if want := []string{"_Arcade/_Long folder name/Same.mra", "_Arcade/_alternatives/_Same/a.mra"}; !reflect.DeepEqual(res.Alts["local:same"], want) {
		t.Fatalf("alts %v", res.Alts["local:same"])
	}
}

func TestScanArcadeMRAsCap(t *testing.T) {
	old := localMaxFiles
	localMaxFiles = 40
	defer func() { localMaxFiles = old }()
	card := t.TempDir()
	dir := filepath.Join(card, "_Arcade", "_Many")
	os.MkdirAll(dir, 0755)
	body := []byte(mra("Many", "many", "core", ""))
	for i := 0; i <= localMaxFiles; i++ {
		os.WriteFile(filepath.Join(dir, "m"+strconv.Itoa(i)+".mra"), body, 0644)
	}
	_, _, err := ScanArcadeMRAs(card, "")
	if !errors.Is(err, ErrTooManyMRAs) {
		t.Fatalf("cap not reported: %v", err)
	}
}
