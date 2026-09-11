package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func fakeCard(t *testing.T) string {
	t.Helper()
	card := t.TempDir()
	mk := func(rel string, content string) {
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}
	mk("_Arcade/cores/Defender_20260714.rbf", "x")
	mk("_Arcade/cores/Defender_20240101.rbf", "x") // older build still around
	mk("_Arcade/cores/jt1942.rbf", "x")            // undated Jotego
	mk("_Arcade/cores/Arcade-Bucky_20260812.rbf", "x")
	mk("_Console/SNES_20260611_22baeda_DB9.rbf", "x")
	mk("_Computer/BK0011M_20260603.rbf", "x")
	mk("_Arcade/Colony 7 (Set 1).mra", "<misterromdescription/>")
	mk("_Arcade/1942 (Revision B).mra", "<misterromdescription/>")
	mk("_Arcade/_alternatives/_Colony 7/Colony 7 (Set 2).mra", `<?xml version="1.0" encoding="ISO-8859-1"?>
<misterromdescription>
 <name>Colony 7 (Set 2)</name>
 <setname>colony7a</setname>
 <rbf>defender</rbf>
 <rom index="0" zip="colony7a.zip|colony7.zip" md5="None">
  <part name="cs03.bin"/>
 </rom>
 <rom index="1"><part>00</part></rom>
</misterromdescription>`)
	mk("_Arcade/_alternatives/_1942/1942 (set 2).mra", `<misterromdescription>
 <setname>1942a</setname>
 <rbf alt="jt1942">jt1942</rbf>
 <rom index="0" zip="1942a.zip|1942.zip"><part name="x"/></rom>
</misterromdescription>`)
	return card
}

func TestScanCoresAndStatus(t *testing.T) {
	card := fakeCard(t)
	idx := ScanCores(card)
	if c, ok := idx.Lookup("defender"); !ok || c.Date != "20260714" {
		t.Fatalf("defender = %+v %v", c, ok)
	}
	if c, ok := idx.Lookup("jt1942"); !ok || c.Date != "" {
		t.Fatalf("jt1942 = %+v %v", c, ok)
	}
	if c, ok := idx.Lookup("SNES"); !ok || c.Date != "20260611" {
		t.Fatalf("SNES = %+v %v", c, ok)
	}
	if c, ok := idx.Lookup("bucky"); !ok || !strings.Contains(c.Path, "Arcade-Bucky") {
		t.Fatalf("bucky via arcade- prefix = %+v %v", c, ok)
	}
	if c, ok := idx.Lookup("zerowing_20240404"); ok {
		t.Fatalf("unexpected %+v", c)
	}
	cases := []struct {
		row  data.Row
		want data.Status
	}{
		{data.Row{Base: "Arcade", MRA: "_Arcade/Colony 7 (Set 1).mra", Core: "defender", Updated: "2026-07-14"}, data.StatusCurrent},
		{data.Row{Base: "Arcade", MRA: "_Arcade/Colony 7 (Set 1).mra", Core: "defender", Updated: "2026-09-01"}, data.StatusOutdated},
		{data.Row{Base: "Arcade", MRA: "_Arcade/Missing.mra", Core: "defender", Updated: "2026-07-14"}, data.StatusNotFound},
		{data.Row{Base: "Arcade", MRA: "_Arcade/1942 (Revision B).mra", Core: "jt1942", Updated: "2026-07-14"}, data.StatusFoundUndated},
		{data.Row{Base: "Console", Core: "SNES", Updated: "2026-06-11"}, data.StatusCurrent},
		{data.Row{Base: "Computer", Core: "BK0011M", Updated: "2026-07-01"}, data.StatusOutdated},
		{data.Row{Base: "Console", Core: "Genesis", Updated: "2026-07-01"}, data.StatusNotFound},
	}
	for i, c := range cases {
		if got := Status(card, idx, &c.row); got != c.want {
			t.Errorf("case %d: status %v, want %v", i, got, c.want)
		}
	}
}

// md5 of the one-byte "x" every fake rbf holds.
const xMD5 = "9dd4e461268c8034f5c8564e155c67a6"

func TestStatusShippedBuild(t *testing.T) {
	card := fakeCard(t)
	mk := func(rel string, content string) {
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}
	mk("_Arcade/cores/BoogieWings_20260708.rbf", "x")
	mk("_Arcade/Boogie Wings.mra", "<misterromdescription/>")
	boogie := func(bd string) data.Row {
		return data.Row{Base: "Arcade", MRA: "_Arcade/Boogie Wings.mra", Core: "BoogieWings", Updated: "2026-08-08", BD: bd}
	}
	jt := func(bh string) data.Row {
		return data.Row{Base: "Arcade", MRA: "_Arcade/1942 (Revision B).mra", Core: "jt1942", Updated: "2026-08-21", BH: bh}
	}
	cases := []struct {
		name string
		row  data.Row
		want data.Status
	}{
		{"mra-only fix: bd decides, not updated", boogie("2026-07-08"), data.StatusCurrent},
		{"old feed without bd falls back to updated", boogie(""), data.StatusOutdated},
		{"genuinely newer shipped build", boogie("2026-09-01"), data.StatusOutdated},
		{"undated rbf, md5 matches", jt(xMD5), data.StatusCurrent},
		{"undated rbf, md5 differs", jt("0123456789abcdef0123456789abcdef"), data.StatusLikelyOutdated},
		{"undated rbf, feed has no md5", jt(""), data.StatusFoundUndated},
		{"dated rbf, md5 differs: dates decide", data.Row{Base: "Console", Core: "SNES", Updated: "2026-06-11", BH: "ffff"}, data.StatusCurrent},
		{"dated rbf, md5 matches despite later updated", data.Row{Base: "Console", Core: "SNES", Updated: "2026-09-01", BH: xMD5}, data.StatusCurrent},
	}
	idx := ScanCores(card)
	for _, c := range cases {
		if got := Status(card, idx, &c.row); got != c.want {
			t.Errorf("%s: status %v, want %v", c.name, got, c.want)
		}
	}

	// A file installed after the feed was generated cannot be behind it.
	idx = ScanCores(card)
	idx.FeedAt = time.Now().Add(-time.Hour)
	if got := Status(card, idx, &[]data.Row{jt("0123456789abcdef0123456789abcdef")}[0]); got != data.StatusCurrent {
		t.Errorf("newer than feed: status %v, want current", got)
	}
	idx = ScanCores(card)
	idx.FeedAt = time.Now().Add(time.Hour)
	if got := Status(card, idx, &[]data.Row{jt("0123456789abcdef0123456789abcdef")}[0]); got != data.StatusLikelyOutdated {
		t.Errorf("older than feed: status %v, want likely outdated", got)
	}
}

func TestStatusDownloaderStore(t *testing.T) {
	card := fakeCard(t)
	mk := func(rel string, content string) {
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}
	// the ledger's md5 is trusted when the size still matches (no hashing),
	// ignored when it does not (hand-copied file under the same name)
	mk(storePath, `{"dbs":{"jtcores":{"files":{"_Arcade/cores/jt1942.rbf":{"hash":"ledger","size":1},
		"|_Console/SNES_20260611_22baeda_DB9.rbf":{"hash":"stale","size":999}}}}}`)
	jt := data.Row{Base: "Arcade", MRA: "_Arcade/1942 (Revision B).mra", Core: "jt1942", BH: "ledger"}
	idx := ScanCores(card)
	if got := Status(card, idx, &jt); got != data.StatusCurrent {
		t.Errorf("ledger match: status %v, want current", got)
	}
	snes := data.Row{Base: "Console", Core: "SNES", Updated: "2026-09-01", BH: xMD5}
	if got := Status(card, idx, &snes); got != data.StatusCurrent {
		t.Errorf("ledger size mismatch falls back to hashing: status %v, want current", got)
	}

	// own computations persist, keyed by size and mtime
	cache := filepath.Join(card, "cache", "hashes.json")
	idx = ScanCores(card)
	idx.HashCache = cache
	Statuses(card, idx, []data.Row{snes})
	raw, err := os.ReadFile(cache)
	if err != nil || !strings.Contains(string(raw), xMD5) {
		t.Fatalf("hash cache = %q, %v", raw, err)
	}
	mk("_Console/SNES_20260611_22baeda_DB9.rbf", "yy") // changed on disk: cache entry must not be reused
	idx = ScanCores(card)
	idx.HashCache = cache
	if got := Status(card, idx, &snes); got != data.StatusOutdated {
		t.Errorf("dated rbf after change: status %v, want outdated (md5 differs, dates decide)", got)
	}
	jtCache := data.Row{Base: "Arcade", MRA: "_Arcade/1942 (Revision B).mra", Core: "jt1942", BH: xMD5}
	mk(storePath, `{}`)
	idx = ScanCores(card)
	if got := Status(card, idx, &jtCache); got != data.StatusCurrent {
		t.Errorf("no ledger, hashed: status %v, want current", got)
	}
}

func TestAlternatives(t *testing.T) {
	card := fakeCard(t)
	cache := filepath.Join(card, "cache", "alts.json")
	alts := ScanAlternatives(card, cache)
	if len(alts) != 2 {
		t.Fatalf("alts = %+v", alts)
	}
	// cached second run gives the same answer
	if again := ScanAlternatives(card, cache); len(again) != 2 || again[0].Path != alts[0].Path {
		t.Fatalf("cached alts = %+v", again)
	}
	row := data.Row{Base: "Arcade", Core: "defender", SN: "colony7"}
	got := Alternatives(alts, &row)
	if len(got) != 1 || !strings.HasSuffix(got[0], "Colony 7 (Set 2).mra") {
		t.Fatalf("colony7 alts = %v", got)
	}
	row2 := data.Row{Base: "Arcade", Core: "jt1942", SN: "1942"}
	if got := Alternatives(alts, &row2); len(got) != 1 {
		t.Fatalf("1942 alts = %v", got)
	}
	other := data.Row{Base: "Arcade", Core: "defender", SN: "defender"}
	if got := Alternatives(alts, &other); len(got) != 0 {
		t.Fatalf("defender alts = %v", got)
	}
}

func TestParseMRAHeaderStopsEarly(t *testing.T) {
	src := `<misterromdescription><rbf>jtcps2</rbf><setname>sfz2al</setname>
<rom index="0" zip="sfz2al.zip|qsound.zip"><part>` + strings.Repeat("00 ", 50000) + `</part></rom></misterromdescription>`
	a, ok := parseMRA(strings.NewReader(src))
	if !ok || a.RBF != "jtcps2" || a.Setname != "sfz2al" || len(a.Zips) != 2 {
		t.Fatalf("parse = %+v %v", a, ok)
	}
}
