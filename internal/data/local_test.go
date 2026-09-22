package data

import (
	"testing"
	"time"
)

func localRow(sn, core, mra string) Row {
	return Row{Title: sn, Base: "Arcade", Src: SrcLocal, K: LocalKey(sn), SN: sn, Core: core, MRA: mra}
}

func TestLocalKeyAndIsLocal(t *testing.T) {
	r := localRow("Orphan", "defender", "_Arcade/Orphan.mra")
	if r.K != "local:orphan" || !r.IsLocal() {
		t.Fatalf("K=%q local=%v", r.K, r.IsLocal())
	}
	c := Row{Src: "distribution_mister"}
	if c.IsLocal() {
		t.Fatal("catalogue row read as local")
	}
}

func TestMergeLocalDropsCovered(t *testing.T) {
	cat := []Row{
		{Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", K: "colony7", SN: "colony7", Core: "defender", Family: "colony7", FamilySets: []string{"colony7a"}},
		{Title: "C64", Base: "Computer", Src: "distribution_mister", K: "C64", SN: "c64x"}, // non-arcade identities do not count
	}
	local := []Row{
		localRow("zzz", "defender", "_Arcade/zzz.mra"),
		localRow("colony7a", "defender", "_Arcade/x.mra"), // known clone
		localRow("colony7", "defender", "_Arcade/y.mra"),  // setname
		localRow("c64x", "c64", "_Arcade/c.mra"),          // only a computer row names it: kept
		localRow("aaa", "defender", "_Arcade/aaa.mra"),
		localRow("aaa", "defender", "_Arcade/dup.mra"), // duplicate K dropped
		{Title: "not local", Base: "Arcade", Src: "coinop", K: "x"},
	}
	got := MergeLocal(cat, local)
	want := []string{"colony7", "C64", "local:aaa", "local:c64x", "local:zzz"}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, k := range want {
		if got[i].K != k {
			t.Fatalf("row %d = %q, want %q", i, got[i].K, k)
		}
	}
	if len(MergeLocal(cat, nil)) != 2 {
		t.Fatal("empty local set must return the catalogue unchanged")
	}
}

func TestMergeLocalKeepsStandin(t *testing.T) {
	cat := []Row{
		{Title: "Volfied", Base: "Arcade", Src: "jtbindb", K: "volfied", SN: "volfied", Core: "jtvlfied", Family: "volfied", FamilySets: []string{"volfiedu"}},
	}
	stand := localRow("volfied", "taitox", "_Arcade/Volfied.mra")
	stand.Standin = true
	clone := localRow("volfiedu", "taitox", "_Arcade/Volfied (US).mra")
	clone.Standin = true
	dup := localRow("volfied", "taitox", "_Arcade/_Extra/Volfied.mra")
	dup.Standin = true
	got := MergeLocal(cat, []Row{stand, clone, dup})
	want := []string{"volfied", "local:volfied", "local:volfiedu"}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, k := range want {
		if got[i].K != k {
			t.Fatalf("row %d = %q, want %q", i, got[i].K, k)
		}
	}
	// The flag is what keeps it: the same file without one is the catalogue's
	// game and goes, as it always did.
	stand.Standin = false
	if got := MergeLocal(cat, []Row{stand}); len(got) != 1 {
		t.Fatalf("plain local row kept: %+v", got)
	}
}

func TestLocalDigestStable(t *testing.T) {
	a := []Row{localRow("b", "c1", "p1"), localRow("a", "c2", "p2")}
	b := []Row{localRow("a", "c2", "p2"), localRow("b", "c1", "p1")}
	if LocalDigest(a) != LocalDigest(b) {
		t.Fatal("order must not change the digest")
	}
	c := []Row{localRow("a", "c2", "p2"), localRow("b", "c1", "p9")}
	if LocalDigest(a) == LocalDigest(c) {
		t.Fatal("a different launch path must change the digest")
	}
	if LocalDigest(nil) != "" {
		t.Fatal("no locals: empty digest")
	}
	if Generation("h", nil) != "h" || Generation("h", a) == "h" {
		t.Fatal("Generation must equal the hash only without locals")
	}
}

func TestIngestNCatGen(t *testing.T) {
	cat := []Row{{Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", K: "colony7", Core: "defender", Updated: "2026-07-14"}}
	ds := Ingest(cat, "h", time.Time{})
	if ds.NCat != 1 || ds.Gen != "h" || len(ds.Catalogue()) != 1 {
		t.Fatalf("no locals: NCat=%d Gen=%q", ds.NCat, ds.Gen)
	}
	rows := MergeLocal(cat, []Row{localRow("orphan", "defender", "_Arcade/o.mra")})
	ds = Ingest(rows, "h", time.Time{})
	if ds.NCat != 1 || ds.Gen != "h:"+LocalDigest(rows[1:]) || len(ds.Rows) != 2 {
		t.Fatalf("with a local: NCat=%d Gen=%q rows=%d", ds.NCat, ds.Gen, len(ds.Rows))
	}
	if ds.Hash != "h" {
		t.Fatal("Hash must stay the catalogue hash")
	}
	if ds.Facets.Src[SrcLocal] != 1 || ds.Index("local:orphan") != 1 {
		t.Fatalf("local row not indexed: src=%v idx=%d", ds.Facets.Src, ds.Index("local:orphan"))
	}
	if ds.Der[1].SrcShort != "Local" || SrcFull(SrcLocal) != SrcLocalFull {
		t.Fatalf("labels: %q %q", ds.Der[1].SrcShort, SrcFull(SrcLocal))
	}
}

func TestSoleTitlesIgnoresLocal(t *testing.T) {
	// a core gen.CoreNames does not know, so the label comes from Sole
	rows := []Row{
		{Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", K: "colony7", Core: "zzcore"},
		localRow("orphan", "zzcore", "_Arcade/o.mra"),
	}
	ds := Ingest(rows, "h", time.Time{})
	if ds.Der[0].CoreLabel != "Colony 7" || ds.Der[1].CoreLabel != "Colony 7" {
		t.Fatalf("core labels %q %q", ds.Der[0].CoreLabel, ds.Der[1].CoreLabel)
	}
}

func TestSeenIgnoresLocal(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	cat := []Row{{Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", K: "colony7", Core: "defender", Updated: "2026-07-14"}}
	stored := &SeenRecord{T: now.Add(-2 * time.Hour).Format(time.RFC3339), Cur: rowMap(cat)}
	rows := MergeLocal(cat, []Row{localRow("orphan", "defender", "_Arcade/o.mra")})
	s := InitSeen(stored, rows, now, true)
	if _, ok := s.State.Cur["local:orphan"]; ok {
		t.Fatal("local row banked")
	}
	if s.Unseen(&rows[1]) {
		t.Fatal("local row must never read as unseen")
	}
	ds := Ingest(rows, "h", time.Time{})
	if s.AnyUnseen(ds, ds.Order(SortUpdated)) {
		t.Fatal("nothing but a local row changed: no unseen")
	}
}

func TestDiffNewsIgnoresLocal(t *testing.T) {
	cat := []Row{{Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", K: "colony7", Core: "defender", Updated: "2026-07-14"}}
	old := Ingest(cat, "h", time.Time{})
	rows := MergeLocal(cat, []Row{localRow("orphan", "defender", "_Arcade/o.mra")})
	if n := DiffNews(old, rows); n != "" {
		t.Fatalf("local row reported as news: %q", n)
	}
}

func TestSortLocalSinks(t *testing.T) {
	cat := []Row{
		{Title: "Zed", Base: "Arcade", Src: "distribution_mister", K: "zed", Core: "defender", Updated: "2026-01-01", Date: "2020-01-01", Year: "1990"},
		{Title: "Able", Base: "Arcade", Src: "distribution_mister", K: "able", Core: "defender", Updated: "2026-07-14", Date: "2019-01-01", Year: "1980"},
	}
	l := localRow("Mid", "defender", "_Arcade/m.mra")
	l.Year = "1985"
	rows := MergeLocal(cat, []Row{l})
	ds := Ingest(rows, "h", time.Time{})
	for _, mode := range []SortMode{SortUpdated, SortDebut} {
		o := ds.Order(mode)
		if ds.Rows[o[len(o)-1]].K != "local:mid" {
			t.Fatalf("mode %v: local row not last: %v", mode, o)
		}
	}
	o := ds.Order(SortYear)
	if ds.Rows[o[1]].K != "local:mid" {
		t.Fatalf("year sort should interleave by year: %v", o)
	}
}

// A standin sorts right under the greyed catalogue row it stands in for in
// every catalogue order, whatever its own title, maker, year or missing dates
// say; a clone filed under its parent finds the row through the family. The
// player's own lists keep it to its own title, and a plain local row, or a
// standin whose game the catalogue no longer names, still sinks.
func TestStandinSortsUnderAnchor(t *testing.T) {
	cat := []Row{
		{Title: "Arkanoid", Base: "Arcade", Src: "distribution_mister", K: "arkanoid", SN: "arkanoid", Core: "arkanoid",
			Manufacturer: "Taito Corporation", Year: "1986", Updated: "2026-09-10", Date: "2020-01-01"},
		{Title: "Volfied", Base: "Arcade", Src: "jtbindb", K: "volfied", SN: "volfied", Core: "jtvlfied", Family: "volfied",
			FamilySets: []string{"volfiedu"}, Manufacturer: "Taito Corporation Japan", Year: "1989", Updated: "2026-09-04", Date: "2026-08-16"},
		{Title: "Zaxxon", Base: "Arcade", Src: "distribution_mister", K: "zaxxon", SN: "zaxxon", Core: "zaxxon",
			Manufacturer: "Sega", Year: "1982", Updated: "2026-01-01", Date: "2019-01-01"},
		{Title: "Last Striker", Base: "Arcade", Src: "jtbindb", K: "kyustrkr", SN: "kyustrkr", Core: "jttaitox",
			Manufacturer: "East Technology", Year: "1989", Updated: "2026-09-04", Date: "2026-09-04"},
	}
	stand := localRow("volfied", "Volfied", "_Arcade/Volfied (bazset).mra")
	stand.Title, stand.Manufacturer, stand.Standin = "Volfied (bazset)", "Taito", true
	clone := localRow("volfiedj2", "Volfied", "_Arcade/Volfied (US, bazset).mra")
	clone.Title, clone.Family, clone.Standin = "Volfied (US, bazset)", "volfied", true
	kyu := localRow("kyustrkr", "taitox", "_Arcade/Kyuukyoku no Striker.mra")
	kyu.Title, kyu.Manufacturer, kyu.Year, kyu.Standin = "Kyuukyoku no Striker", "East Technology", "1989", true
	plain := localRow("orphan", "defender", "_Arcade/orphan.mra")
	plain.Title, plain.Year = "Mid Orphan", "1985"
	gone := localRow("gone", "defender", "_Arcade/gone.mra")
	gone.Standin = true
	ds := Ingest(MergeLocal(cat, []Row{stand, clone, kyu, plain, gone}), "h", time.Time{})
	at := func(o []int, k string) int {
		for p, i := range o {
			if ds.Rows[i].K == k {
				return p
			}
		}
		t.Fatalf("%s not in the order", k)
		return -1
	}
	for _, mode := range []SortMode{SortUpdated, SortDebut, SortYear, SortAlphabetical, SortMaker} {
		o := ds.Order(mode)
		p := at(o, "volfied")
		if at(o, "local:volfied") != p+1 || at(o, "local:volfiedj2") != p+2 {
			t.Fatalf("%v: standins not under Volfied: %v", mode, o)
		}
		if at(o, "local:kyustrkr") != at(o, "kyustrkr")+1 {
			t.Fatalf("%v: standin not under Last Striker: %v", mode, o)
		}
		if mode == SortUpdated || mode == SortDebut {
			if at(o, "local:gone") < len(o)-2 || at(o, "local:orphan") < len(o)-2 {
				t.Fatalf("%v: undated rows without an anchor must sink: %v", mode, o)
			}
		}
	}
	if o := ds.Order(SortFavorites); at(o, "local:kyustrkr") > at(o, "kyustrkr") {
		t.Fatalf("favorites sorts a standin by its own title: %v", o)
	}
	if i := ds.Index("local:gone"); ds.SortRow(i) != i {
		t.Fatal("a standin the catalogue no longer names has no anchor")
	}
	if ds.SortRow(ds.Index("local:volfiedj2")) != ds.Index("volfied") {
		t.Fatal("a clone standin anchors through its parent")
	}
}

func TestHiddenSourcesNeverHidesLocal(t *testing.T) {
	if h := HiddenSources([]DB{}); h[SrcLocal] {
		t.Fatal("local hidden with no databases configured")
	}
}
