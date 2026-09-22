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
	mk("_Arcade/PGM (Polygame Master) System BIOS.mra", mra("PGM (Polygame Master) System BIOS", "pgm", "igspgm", ""))
	mk("_Arcade/_Extra/neogeo-bios.mra", mra("Neo Geo", "neogeo", "neogeo", ""))
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
		"_Arcade/PGM (Polygame Master) System BIOS.mra",
		"_Arcade/_Extra/Colony Clone.mra",
		"_Arcade/_Extra/Colony Known.mra",
		"_Arcade/_Extra/Orphan (set 2).mra",
		"_Arcade/_Extra/deeper/Orphan.mra",
		"_Arcade/_Extra/nameless.mra",
		"_Arcade/_Extra/neogeo-bios.mra",
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
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), alts, attached, false)
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
	if res.Files != 8 {
		t.Fatalf("files %d", res.Files)
	}
	// The two BIOS files (one named so in the header, one only in the file
	// name) are skipped with a reason, not listed.
	bios := 0
	for _, s := range res.Skipped {
		if s.Reason == "BIOS, not a game" {
			bios++
		}
	}
	if bios != 2 {
		t.Fatalf("BIOS skips %d: %+v", bios, res.Skipped)
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

	with := DiscoverLocal(card, "", catalogue, ScanCores(card), alts, attached, true)
	g := with.Rows[0]
	if g.Img != "galaxloc" || !reflect.DeepEqual(g.ImgSlots, []string{SlotLocalSnap}) || g.ImgW != 3 || g.ImgH != 4 {
		t.Fatalf("service picture: %+v", g)
	}
}

// A game the catalogue knows only through a core the card has not got:
// the file that does run it stands in, rather than falling between the
// greyed catalogue row and the setname rule.
func TestDiscoverLocalStandin(t *testing.T) {
	card := fakeCard(t)
	mk := func(rel, content string) {
		t.Helper()
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk("_Arcade/cores/taitox_20260904.rbf", "x") // a core no catalogue row names
	mk("_Arcade/_Extra/Gigandes.mra", mra("Gigandes", "gigandes", "taitox", "gigandes"))
	mk("_Arcade/_Extra/Superman.mra", mra("Superman", "superman", "nosuch", "superman"))
	mk("_Arcade/_Extra/Colony 7 (other).mra", mra("Colony 7", "colony7", "taitox", "colony7"))
	catalogue := []data.Row{
		{K: "gigandes", Title: "Gigandes", Base: "Arcade", Core: "jttaitox", SN: "gigandes", MRA: "_Arcade/Gigandes.mra"},
		{K: "superman", Title: "Superman", Base: "Arcade", Core: "jttaitox", SN: "superman", MRA: "_Arcade/Superman.mra"},
		{K: "colony7", Title: "Colony 7", Base: "Arcade", Core: "defender", SN: "colony7", Family: "colony7", FamilySets: []string{"colony7a"}, MRA: "_Arcade/Colony 7 (Set 1).mra"},
		{K: "1942", Title: "1942", Base: "Arcade", Core: "jt1942", SN: "1942", FamilySets: []string{"1942a"}, MRA: "_Arcade/1942 (Revision B).mra"},
	}
	rows := func(idx *Index) []data.Row {
		t.Helper()
		alts, _, _ := ScanAlternativesWithError(card, "")
		var fc FamilyCache
		res := DiscoverLocal(card, "", catalogue, idx, alts, AttachedPaths(fc.Resolve(card, alts, catalogue)), false)
		if res.Err != nil {
			t.Fatal(res.Err)
		}
		return res.Rows
	}
	// Gigandes stands in for a core that is not here. Superman's own copy
	// needs an absent core too, so its greyed catalogue row says it best,
	// and Colony 7 already runs from the catalogue's own defender build.
	got := rows(ScanCores(card))
	if len(got) != 1 {
		t.Fatalf("rows %+v", got)
	}
	if r := got[0]; r.K != "local:gigandes" || !r.Standin || r.Core != "taitox" || r.MRA != "_Arcade/_Extra/Gigandes.mra" {
		t.Fatalf("standin row: %+v", r)
	}
	// Without a core index nothing is on the card, so nothing stands in.
	if got := rows(nil); len(got) != 0 {
		t.Fatalf("no index: %+v", got)
	}
	// Once the catalogue's own core arrives, the standin gives way to it.
	mk("_Arcade/cores/jttaitox_20260907.rbf", "x")
	if got := rows(ScanCores(card)); len(got) != 0 {
		t.Fatalf("core installed: %+v", got)
	}
}

// The diagnostic report says why a file is not a row: a catalogue game whose
// own core is here, a catalogue game nobody here can run, a version of a
// catalogue game (counted), and the folders the walk left out.
func TestDiscoverLocalReasons(t *testing.T) {
	card := fakeCard(t)
	mk := func(rel, content string) {
		t.Helper()
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk("_Arcade/cores/taitox_20260904.rbf", "x")
	mk("_Arcade/_Extra/Gigandes.mra", mra("Gigandes", "gigandes", "taitox", "gigandes"))
	mk("_Arcade/_Extra/Superman.mra", mra("Superman", "superman", "nosuch", "superman"))
	mk("_Arcade/_Extra/Colony 7 (other).mra", mra("Colony 7", "colony7", "taitox", "colony7"))
	mk("_Arcade/_Organized/_A/Organised.mra", mra("Organised", "organised", "defender", ""))
	mk("_Arcade/.hidden/Hidden.mra", mra("Hidden", "hidden", "defender", ""))
	mk("_Arcade/a/b/c/d/e/f/g/Deep.mra", mra("Deep", "deep", "defender", ""))
	linked := os.Symlink(filepath.Join(card, "_Arcade", "_Extra"), filepath.Join(card, "_Arcade", "Linked")) == nil
	catalogue := []data.Row{
		{K: "gigandes", Title: "Gigandes", Base: "Arcade", Core: "jttaitox", SN: "gigandes", MRA: "_Arcade/Gigandes.mra"},
		{K: "superman", Title: "Superman", Base: "Arcade", Core: "jttaitox", SN: "superman", MRA: "_Arcade/Superman.mra"},
		{K: "colony7", Title: "Colony 7", Base: "Arcade", Core: "defender", SN: "colony7", Family: "colony7", FamilySets: []string{"colony7a"}, MRA: "_Arcade/Colony 7 (Set 1).mra"},
		{K: "1942", Title: "1942", Base: "Arcade", Core: "jt1942", SN: "1942", FamilySets: []string{"1942a"}, MRA: "_Arcade/1942 (Revision B).mra"},
	}
	alts, _, _ := ScanAlternativesWithError(card, "")
	var fc FamilyCache
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), alts, AttachedPaths(fc.Resolve(card, alts, catalogue)), false)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(res.Rows) != 1 || res.Rows[0].K != "local:gigandes" || !res.Rows[0].Standin {
		t.Fatalf("rows %+v", res.Rows)
	}
	got := map[string]Accounted{}
	for _, a := range res.Accounted {
		got[a.Path] = a
	}
	if a := got["_Arcade/_Extra/Colony 7 (other).mra"]; a.K != "colony7" || !strings.Contains(a.Reason, "its own core is on the card") || !strings.Contains(a.Reason, "taitox") {
		t.Fatalf("runnable catalogue game: %+v", a)
	}
	if a := got["_Arcade/_Extra/Superman.mra"]; a.K != "superman" || !strings.Contains(a.Reason, "neither") || !strings.Contains(a.Reason, "nosuch") {
		t.Fatalf("unrunnable catalogue game: %+v", a)
	}
	if len(res.Accounted) != 2 || res.VersionFiles != 2 { // Colony 7 (Set 2) and 1942 (set 2)
		t.Fatalf("accounted %+v, versions %d", res.Accounted, res.VersionFiles)
	}
	dirs := map[string]string{}
	for _, d := range res.SkippedDirs {
		dirs[d.Path] = d.Reason
	}
	want := map[string]string{"_Arcade/_Organized": "organiser", "_Arcade/.hidden": "hidden", "_Arcade/a/b/c/d/e/f/g": "deep"}
	if linked {
		want["_Arcade/Linked"] = "symbolic link"
	}
	for p, word := range want {
		if !strings.Contains(dirs[p], word) {
			t.Fatalf("skipped dir %s: %q (all %v)", p, dirs[p], dirs)
		}
	}
	if len(dirs) != len(want) {
		t.Fatalf("cores or _alternatives listed as skipped: %v", dirs)
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
	res := DiscoverLocal(card, "", nil, ScanCores(card), alts, nil, false)
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
