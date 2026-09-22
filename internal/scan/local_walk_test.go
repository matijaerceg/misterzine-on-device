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
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), nil, alts, attached, false)
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

	with := DiscoverLocal(card, "", catalogue, ScanCores(card), nil, alts, attached, true)
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
		res := DiscoverLocal(card, "", catalogue, idx, nil, alts, AttachedPaths(fc.Resolve(card, alts, catalogue)), false)
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

// The diagnostic report says why a file is not a row: a catalogue game nobody
// here can run, a version of a catalogue game (counted), and the folders the
// walk left out; a catalogue game that runs here takes a copy on another
// core on the card as a version.
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
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), nil, alts, AttachedPaths(fc.Resolve(card, alts, catalogue)), false)
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
	// Colony 7 runs here from the catalogue, and this copy's own core is on
	// the card too: it is one more version of the game.
	if vs := res.Versions["colony7"]; len(vs) != 1 || vs[0] != "_Arcade/_Extra/Colony 7 (other).mra" || res.VersionCores[vs[0]] != "taitox" {
		t.Fatalf("runnable catalogue game on another core: %v", res.Versions)
	}
	if a := got["_Arcade/_Extra/Superman.mra"]; a.K != "superman" || !strings.Contains(a.Reason, "neither") || !strings.Contains(a.Reason, "nosuch") {
		t.Fatalf("unrunnable catalogue game: %+v", a)
	}
	if len(res.Accounted) != 1 || res.VersionFiles != 2 { // Colony 7 (Set 2) and 1942 (set 2)
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

// A card file becomes a version of the catalogue game it belongs to when
// that game runs here and the file's own core is on the card, whatever the
// core. Belonging is strict: the exact setname owns a file, and a family
// only when all its claimants are one release, since families span titles.
func TestDiscoverLocalVersions(t *testing.T) {
	card := t.TempDir()
	mk := func(rel, content string) {
		t.Helper()
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []string{"phoenix_20240601", "pleiads_20240601", "riverpatrol_20240601", "silverland_20240601", "sprint1_20240601",
		"sprint2_20240601", "blkheart_mister_20260909", "Arcade-NMK16_Gunnail_20260919", "TaitoSJ_20260321", "nemesis_20240601", "defender_20240601"} {
		mk("_Arcade/cores/"+c+".rbf", "x")
	}
	const (
		centuri  = "_Arcade/_alternatives/_Pleiads/Pleiads (Centuri).mra"
		capitol  = "_Arcade/_alternatives/_Pleiads/Capitol.mra"
		bios     = "_Arcade/_alternatives/_Pleiads/Pleiads BIOS.mra"
		silver   = "_Arcade/Silver Land.mra"
		rpboot   = "_Arcade/River Patrol bootleg.mra"
		sprint2a = "_Arcade/Sprint 2 set 2.mra"
		bhj      = "_Arcade/Black Heart (Japan).mra"
		bhnmk    = "_Arcade/_alternatives/_Black Heart/Black Heart (NMK set).mra"
		jungle   = "_Arcade/Jungle Hunt US.mra"
		gradius  = "_Arcade/Gradius (other).mra"
		mainless = "_Arcade/Mainless copy.mra"
	)
	mk(centuri, mra("Pleiads (Centuri)", "pleiadce", "pleiads", "pleiads"))
	mk(capitol, mra("Capitol", "capitol", "nosuchcore", "pleiads"))
	mk(bios, mra("Pleiads BIOS", "pleiadce", "pleiads", "pleiads"))
	mk(silver, mra("Silver Land", "silvland", "silverland", "rpatrol"))
	mk(rpboot, mra("River Patrol (bootleg)", "rpatrolb", "riverpatrol", "rpatrol"))
	mk(sprint2a, mra("Sprint 2 (set 2)", "sprint2a", "sprint2", "sprint1"))
	mk(bhj, mra("Black Heart (Japan)", "blkheartj", "blkheart_mister", "blkheart"))
	mk(bhnmk, mra("Black Heart (NMK)", "blkheart", "Arcade-NMK16_Gunnail", ""))
	mk(jungle, mra("Jungle Hunt (US)", "jungleh", "taitosj", "junglek"))
	mk(gradius, mra("Gradius", "gradius", "nemesis", "nemesis"))
	mk(mainless, mra("Mainless", "mainless", "defender", ""))
	catalogue := []data.Row{
		{K: "pleiads", Title: "Pleiads", Base: "Arcade", Core: "phoenix", SN: "pleiads", Family: "pleiads", FamilySets: []string{"pleiadce", "capitol"}, MRA: "_Arcade/Pleiads (Tehkan).mra"},
		{K: "rpatrol", Title: "River Patrol", Base: "Arcade", Core: "riverpatrol", SN: "rpatrol", Family: "rpatrol", FamilySets: []string{"rpatrolb"}, MRA: "_Arcade/River Patrol.mra"},
		{K: "silvland", Title: "Silver Land", Base: "Arcade", Core: "silverland", SN: "silvland", Family: "rpatrol", FamilySets: []string{"rpatrolb", "silvland"}, MRA: "_Arcade/Silver Land (catalogue).mra"},
		{K: "sprint1", Title: "Sprint 1", Base: "Arcade", Core: "sprint1", SN: "sprint1", Family: "sprint1", FamilySets: []string{"sprint2", "sprint2a"}, MRA: "_Arcade/Sprint 1.mra"},
		{K: "sprint2", Title: "Sprint 2", Base: "Arcade", Core: "sprint2", SN: "sprint2", Family: "sprint1", FamilySets: []string{"sprint2", "sprint2a"}, MRA: "_Arcade/Sprint 2.mra"},
		{K: "blkheart", Title: "Black Heart (Coin-Op Collection)", Base: "Arcade", Core: "blkheart_mister", SN: "blkheart", Family: "blkheart", FamilySets: []string{"blkheartj"}, MRA: "_Arcade/Black Heart (Coin-Op).mra"},
		{K: "black-heart", Title: "Black Heart", Base: "Arcade", Core: "Arcade-NMK16_Gunnail", SN: "blkheart", Family: "blkheart", FamilySets: []string{"blkheartj"}, MRA: "_Arcade/Black Heart.mra"},
		{K: "junglek", Title: "Jungle King", Base: "Arcade", Core: "TaitoSJ", SN: "junglek", Family: "junglek", FamilySets: []string{"jungleh"}, MRA: "_Arcade/Jungle King (Japan).mra"},
		{K: "gradius", Title: "Gradius", Base: "Arcade", Core: "BubSysROM", SN: "gradius", Family: "nemesis", FamilySets: []string{"gradius", "nemesisuk"}, MRA: "_Arcade/Gradius.mra"},
		{K: "nemesis", Title: "Nemesis", Base: "Arcade", Core: "nemesis", SN: "nemesis", Family: "nemesis", FamilySets: []string{"gradius", "nemesisuk"}, MRA: "_Arcade/Nemesis.mra"},
		{K: "mainless", Title: "Mainless", Base: "Arcade", Core: "defender", SN: "mainless", MRA: "_Arcade/Mainless.mra"},
	}
	// what the list shows: Gradius's core is absent, and Mainless's own MRA
	// is not on the card although its core is
	status := make([]data.Status, len(catalogue))
	for i := range status {
		status[i] = data.StatusCurrent
	}
	status[8], status[10] = data.StatusNotFound, data.StatusNotFound
	alts, _, _ := ScanAlternativesWithError(card, "")
	var fc FamilyCache
	resolved := fc.Resolve(card, alts, catalogue)
	res := DiscoverLocal(card, "", catalogue, ScanCores(card), status, alts, AttachedPaths(resolved), false)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	has := func(k, p string) bool {
		for _, v := range res.Versions[k] {
			if v == p {
				return true
			}
		}
		return false
	}
	why := map[string]string{}
	for _, a := range res.Accounted {
		why[a.Path] = a.Reason
	}
	for _, s := range res.Skipped {
		why[s.Path] = s.Reason
	}
	rows := map[string]bool{}
	for _, r := range res.Rows {
		rows[r.K] = r.Standin
	}

	if !has("pleiads", centuri) || res.VersionCores[centuri] != "pleiads" {
		t.Errorf("a set on another core on the card is a version: %v", res.Versions)
	}
	if !strings.Contains(why[capitol], "nosuchcore") || !strings.Contains(why[capitol], "not in _Arcade/cores") {
		t.Errorf("a set whose core is missing: %q", why[capitol])
	}
	if has("pleiads", bios) || !strings.Contains(why[bios], "BIOS") {
		t.Errorf("a BIOS file is never a version: %q", why[bios])
	}
	if !has("silvland", silver) || has("rpatrol", silver) {
		t.Errorf("the exact setname owns a file, not its family sibling: %v", res.Versions)
	}
	for _, p := range []string{rpboot, sprint2a} {
		if !strings.Contains(why[p], "several catalogue games share") {
			t.Errorf("%s spans titles, so it is no one's version: %q", p, why[p])
		}
	}
	if !has("blkheart", bhj) || !has("black-heart", bhj) || res.VersionOwner[bhj] != "blkheart" {
		t.Errorf("one release in two implementations offers the set in both: %v", res.Versions)
	}
	if !has("blkheart", bhnmk) {
		t.Errorf("a set attached to one implementation reaches the other: %v", res.Versions)
	}
	if !has("junglek", jungle) {
		t.Errorf("a same-core set outside _alternatives is a version: %v", res.Versions)
	}
	if standin, ok := rows["local:gradius"]; !ok || !standin || has("nemesis", gradius) {
		t.Errorf("an exact owner that cannot run is never replaced by a family sibling: rows %v", rows)
	}
	if standin, ok := rows["local:mainless"]; !ok || !standin {
		t.Errorf("a game whose own MRA is missing shows greyed, so the copy stands in: rows %v", rows)
	}

	offers, files := MergeVersions(resolved, res.Versions)
	if !contains(resolved["pleiads"], centuri) || !contains(resolved["black-heart"], bhnmk) {
		t.Fatalf("merged: %v", resolved)
	}
	seen := map[string]int{}
	for _, p := range resolved["black-heart"] {
		seen[p]++
	}
	if seen[bhnmk] != 1 || offers == 0 || files == 0 {
		t.Fatalf("merge must not repeat a file already offered: %v (offers %d, files %d)", resolved["black-heart"], offers, files)
	}

	// Without statuses the core alone decides, as before: Mainless runs.
	res = DiscoverLocal(card, "", catalogue, ScanCores(card), nil, alts, AttachedPaths(resolved), false)
	if !contains(res.Versions["mainless"], mainless) {
		t.Errorf("no statuses: a present core means the row runs: %v", res.Versions)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
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
	res := DiscoverLocal(card, "", nil, ScanCores(card), nil, alts, nil, false)
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
