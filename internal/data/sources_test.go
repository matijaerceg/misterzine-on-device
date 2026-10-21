package data

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
)

func TestHiddenSourcesFollowDownloaderDatabases(t *testing.T) {
	if HiddenSources(nil) != nil {
		t.Fatal("an unknown configuration must hide nothing")
	}
	got := HiddenSources([]DB{
		{ID: "distribution_mister", URL: "https://raw.githubusercontent.com/mister-devel/distribution_mister/main/db.json.zip"},
		{ID: "jtcores"},
		{ID: "other/db", URL: "https://example.com/db.json.zip"},
	})
	if want := map[string]bool{"bazset": true, "blahm1d": true, "coinop": true, "decoryah": true, "gm0rk": true, "jlrh": true, "kuzecores": true, "meathax": true, "ongo": true, "ppriest": true, "rmcores": true, "rossops": true, "shmupfan": true, "slopcore": true, "theypsilon_unofficial_distribution": true, "tomsvr": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	// the db_url identifies a renamed section; the old Coin-Op name still counts
	got = HiddenSources([]DB{
		{ID: "mine", URL: "https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip"},
		{ID: "atrac17/coin-op_collection"},
	})
	if want := map[string]bool{"bazset": true, "blahm1d": true, "decoryah": true, "distribution_mister": true, "gm0rk": true, "jlrh": true, "jtbindb": true, "kuzecores": true, "ongo": true, "ppriest": true, "rmcores": true, "rossops": true, "shmupfan": true, "slopcore": true, "theypsilon_unofficial_distribution": true, "tomsvr": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	got = HiddenSources([]DB{{ID: "distribution_mister"}, {ID: "jtcores"}, {ID: "coin-opcollection/distribution-misterfpga"}, {ID: "meathax/meatcores"}, {ID: "rmonic79/rmcores"}, {ID: "TheJesusFish/Slop-Core"}, {ID: "kuzearcade/kuzecores"}, {ID: "blahm1d"}, {ID: "bazset/MiSTer_BAZSET"}, {ID: "rossops/Arcade-SegaYBoard_MiSTer"}, {ID: "OngoGablogian/MiSTer_Ongo"}, {ID: "TomSVR-Mister/TomSVR-Arcade-Cores"}, {ID: "decoryah/MiSTer_decoryah"}, {ID: "ppriest/MiSTer_ppriest"}, {ID: "Gm0rk/Gm0rkCores"}, {ID: "jlrh/jlrh-misterfpga-db"}, {ID: "shmupfan"}, {ID: "theypsilon_unofficial_distribution"}})
	if got == nil || len(got) != 0 {
		t.Fatalf("every database present must hide nothing: %v", got)
	}
}

func TestHiddenSourcesFilter(t *testing.T) {
	rows := []Row{
		{K: "a", Base: "Arcade", Src: "jtbindb"},
		{K: "b", Base: "Console", Src: "distribution_mister"},
		{K: "c", Base: "Arcade", Src: "newsource"},
	}
	ds := Ingest(rows, "", time.Time{})
	f := &Filters{SrcHidden: map[string]bool{"jtbindb": true, "meathax": true}}
	if !f.Active() {
		t.Fatal("hidden sources must count as narrowing")
	}
	if got := Apply(ds, []int{0, 1, 2}, f, nil, nil, nil, nil); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("hidden sources filter = %v", got)
	}
}

func TestUnofficialSourceRecognizesUpdateAllAndRenamedSections(t *testing.T) {
	const src = "theypsilon_unofficial_distribution"
	for _, db := range []DB{
		{ID: src},
		{ID: "custom", URL: "https://raw.githubusercontent.com/theypsilon/Unofficial_Distribution_MiSTer/main/unofficialdb.json.zip"},
		{ID: "custom", URL: "https://raw.githubusercontent.com/theypsilon/Distribution_Unofficial_MiSTer/main/unofficialdb.json.zip"},
	} {
		if HiddenSources([]DB{db})[src] {
			t.Fatalf("installed source hidden: %+v", db)
		}
	}
	if !HiddenSources([]DB{})[src] {
		t.Fatal("absent source must be hidden")
	}
	if SrcShort(src) != "theypsilon" || SrcFull(src) != "theypsilon Unofficial Distribution" {
		t.Fatal("source labels missing")
	}
}

// Every source the site names has a short chip that fits the column and a
// Downloader database with a db_url and lower-case section names, so a new
// site source cannot ship half-wired: mzgen brings the name, section and
// opt-in db_url, sourceExtras the rest, and this test demands it all.
func TestSourceTablesComplete(t *testing.T) {
	for src := range gen.SrcNames {
		if chip := SrcShort(src); chip == "" || len([]rune(chip)) > srcShortMax {
			t.Errorf("%s: short chip %q is empty or too wide", src, chip)
		}
		s, ok := tables.Load().dbs[src]
		if !ok {
			t.Errorf("%s: no database in the site tables", src)
			continue
		}
		if !strings.HasPrefix(s.url, "https://") || len(s.ids) == 0 {
			t.Errorf("%s: incomplete database entry %+v (an update_all-toggled database needs its db_url in sourceExtras)", src, s)
		}
		for _, id := range s.ids {
			if id != strings.ToLower(id) {
				t.Errorf("%s: section %q must be lower-case to match", src, id)
			}
		}
	}
	for src := range srcShort {
		if src == SrcLocal {
			continue
		}
		if _, ok := gen.SrcNames[src]; !ok {
			t.Errorf("%s: short chip for a source the site does not name", src)
		}
	}
	for src := range sourceExtras {
		if _, ok := gen.SrcNames[src]; !ok {
			t.Errorf("%s: extras for a source the site does not name", src)
		}
	}
	if len(tables.Load().dbs) != len(gen.SrcNames) {
		t.Errorf("the site tables have %d sources, the site names %d", len(tables.Load().dbs), len(gen.SrcNames))
	}
}

// The short chip derives from the full name when no hand-picked one exists,
// so a new site source shows a sensible chip without a labels.go line.
func TestSrcShortDerived(t *testing.T) {
	cases := map[string]string{
		"kuzecores":                     "kuzecores", // "kuzecores (kuzearcade)" minus the author
		"slopcore":                      "Slop",      // hand-picked wins
		"newsource":                     "newsource", // unknown to the site: the raw id
		"a_very_long_unknown_source_id": "a_very_lon",
	}
	for src, want := range cases {
		if got := SrcShort(src); got != want {
			t.Errorf("SrcShort(%q) = %q, want %q", src, got, want)
		}
	}
}

// Slop Cores (TheJesusFish): an opt-in database update_all has no toggle for.
func TestSlopCoreSource(t *testing.T) {
	const src = "slopcore"
	for _, db := range []DB{
		{ID: "TheJesusFish/Slop-Core"},
		{ID: "custom", URL: "https://raw.githubusercontent.com/TheJesusFish/Slop-Core/db/db.json.zip"},
	} {
		if HiddenSources([]DB{db})[src] {
			t.Fatalf("installed source hidden: %+v", db)
		}
	}
	if !HiddenSources([]DB{{ID: "distribution_mister"}})[src] {
		t.Fatal("absent source must be hidden")
	}
	if SrcShort(src) != "Slop" || SrcFull(src) != "Slop Cores (TheJesusFish)" {
		t.Fatalf("source labels: %q %q", SrcShort(src), SrcFull(src))
	}
}

// kuzecores (kuzearcade): the first source wired entirely from the site's
// tables, with no hand-written line in the app. Its section is matched in
// lower case, as update_all writes it in the repository's own case.
func TestKuzecoresSource(t *testing.T) {
	const src = "kuzecores"
	for _, db := range []DB{
		{ID: "kuzearcade/kuzecores"},
		{ID: "KuzeArcade/KuzeCores"},
		{ID: "custom", URL: "https://raw.githubusercontent.com/kuzearcade/kuzecores/db/db.json.zip"},
	} {
		if HiddenSources([]DB{db})[src] {
			t.Fatalf("installed source hidden: %+v", db)
		}
	}
	if !HiddenSources([]DB{{ID: "distribution_mister"}})[src] {
		t.Fatal("absent source must be hidden")
	}
	if _, ok := srcShort[src]; ok {
		t.Fatal("kuzecores must prove the derived chip; drop the srcShort line or pick another source for this test")
	}
	if _, ok := sourceExtras[src]; ok {
		t.Fatal("an opt-in database needs no sourceExtras entry")
	}
	if SrcShort(src) != "kuzecores" || SrcFull(src) != "kuzecores (kuzearcade)" {
		t.Fatalf("source labels: %q %q", SrcShort(src), SrcFull(src))
	}
	for _, core := range []string{"Arcade-NMK16_Gunnail", "Arcade-NMK16_Macross2", "Arcade-NMK16_Raphero", "Arcade-NMK16_Afega"} {
		if _, ok := gen.CoreNames[core]; !ok {
			t.Errorf("%s: no core name from the site", core)
		}
	}
}

// blahm1d: the one database not hosted on GitHub; his downloader_blahm1d.ini
// names the section [blahm1d] and the db_url below.
func TestBlahm1dSource(t *testing.T) {
	const src = "blahm1d"
	for _, db := range []DB{
		{ID: "blahm1d"},
		{ID: "Blahm1d"},
		{ID: "custom", URL: "https://mister.blahm1d.com/db.json.zip"},
	} {
		if HiddenSources([]DB{db})[src] {
			t.Fatalf("installed source hidden: %+v", db)
		}
	}
	if !HiddenSources([]DB{{ID: "distribution_mister"}})[src] {
		t.Fatal("absent source must be hidden")
	}
	if SrcShort(src) != "blahm1d" || SrcFull(src) != "blahm1d" {
		t.Fatalf("source labels: %q %q", SrcShort(src), SrcFull(src))
	}
	for _, core := range []string{"blahm1d_yunit", "blahm1d_tunit", "blahm1d_wolfunit", "blahm1d_exidy440"} {
		if _, ok := gen.CoreNames[core]; !ok {
			t.Errorf("%s: no core name from the site", core)
		}
	}
}

// bazset: an opt-in database like MeatCores; his downloader ini names the
// section [bazset/MiSTer_BAZSET] and the db_url below.
func TestBazsetSource(t *testing.T) {
	const src = "bazset"
	for _, db := range []DB{
		{ID: "bazset/MiSTer_BAZSET"},
		{ID: "Bazset/MiSTer_BAZSET"},
		{ID: "custom", URL: "https://raw.githubusercontent.com/bazset/MiSTer_BAZSET/db/db.json.zip"},
	} {
		if HiddenSources([]DB{db})[src] {
			t.Fatalf("installed source hidden: %+v", db)
		}
	}
	if !HiddenSources([]DB{{ID: "distribution_mister"}})[src] {
		t.Fatal("absent source must be hidden")
	}
	if SrcShort(src) != "bazset" || SrcFull(src) != "bazset" {
		t.Fatalf("source labels: %q %q", SrcShort(src), SrcFull(src))
	}
	if _, ok := gen.CoreNames["EXPRO02_bazset"]; !ok {
		t.Error("EXPRO02_bazset: no core name from the site")
	}
}

// Opt-in sources tracked by the site tables: their Downloader identity and
// display name must flow through mzgen without hand-maintained app metadata.
func TestOptInSiteSources(t *testing.T) {
	gh := func(section string) string { return "https://raw.githubusercontent.com/" + section + "/db/db.json.zip" }
	for _, c := range []struct{ src, section, url, short, full string }{
		{"ongo", "OngoGablogian/MiSTer_Ongo", gh("OngoGablogian/MiSTer_Ongo"), "Ongo", "Ongo"},
		{"tomsvr", "TomSVR-Mister/TomSVR-Arcade-Cores", gh("TomSVR-Mister/TomSVR-Arcade-Cores"), "TomSVR", "TomSVR"},
		{"decoryah", "decoryah/MiSTer_decoryah", gh("decoryah/MiSTer_decoryah"), "decoryah", "decoryah"},
		{"ppriest", "ppriest/MiSTer_ppriest", gh("ppriest/MiSTer_ppriest"), "ppriest", "ppriest"},
		{"gm0rk", "Gm0rk/Gm0rkCores", gh("Gm0rk/Gm0rkCores"), "Gm0rk", "Gm0rk"},
		{"jlrh", "jlrh/jlrh-misterfpga-db", gh("jlrh/jlrh-misterfpga-db"), "jlrh", "jlrh"},
		// a plain db.json on main, under a section without a slash
		{"shmupfan", "shmupfan", "https://raw.githubusercontent.com/shmupfan/Distribution/main/db.json", "shmupfan", "shmupfan"},
	} {
		for _, db := range []DB{
			{ID: c.section},
			{ID: strings.ToLower(c.section)},
			{ID: "custom", URL: c.url},
		} {
			if HiddenSources([]DB{db})[c.src] {
				t.Fatalf("%s: installed source hidden: %+v", c.src, db)
			}
		}
		if !HiddenSources([]DB{{ID: "distribution_mister"}})[c.src] {
			t.Fatalf("%s: absent source must be hidden", c.src)
		}
		if SrcShort(c.src) != c.short || SrcFull(c.src) != c.full {
			t.Fatalf("%s: source labels: %q %q", c.src, SrcShort(c.src), SrcFull(c.src))
		}
	}
	// shmupfan's five multi-game cores and jlrh's shared ones, keyed as the feed's core field
	for _, core := range []string{"1945kiii", "dec8", "dooyong", "tecmo16", "TaitoGNET", "ffaltair", "ffmegaphx", "ffspeedbal"} {
		if _, ok := gen.CoreNames[core]; !ok {
			t.Errorf("%s: no core name from the site", core)
		}
	}
}
