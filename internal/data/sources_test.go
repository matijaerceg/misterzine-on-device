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
	if want := map[string]bool{"coinop": true, "kuzecores": true, "meathax": true, "rmcores": true, "slopcore": true, "theypsilon_unofficial_distribution": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	// the db_url identifies a renamed section; the old Coin-Op name still counts
	got = HiddenSources([]DB{
		{ID: "mine", URL: "https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip"},
		{ID: "atrac17/coin-op_collection"},
	})
	if want := map[string]bool{"distribution_mister": true, "jtbindb": true, "kuzecores": true, "rmcores": true, "slopcore": true, "theypsilon_unofficial_distribution": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	got = HiddenSources([]DB{{ID: "distribution_mister"}, {ID: "jtcores"}, {ID: "coin-opcollection/distribution-misterfpga"}, {ID: "meathax/meatcores"}, {ID: "rmonic79/rmcores"}, {ID: "TheJesusFish/Slop-Core"}, {ID: "kuzearcade/kuzecores"}, {ID: "theypsilon_unofficial_distribution"}})
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
	if got := Apply(ds, []int{0, 1, 2}, f, nil, nil, nil); !reflect.DeepEqual(got, []int{1, 2}) {
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
		s, ok := sourceDBs[src]
		if !ok {
			t.Errorf("%s: no database in sourceDBs", src)
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
	if len(sourceDBs) != len(gen.SrcNames) {
		t.Errorf("sourceDBs has %d sources, the site names %d", len(sourceDBs), len(gen.SrcNames))
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
