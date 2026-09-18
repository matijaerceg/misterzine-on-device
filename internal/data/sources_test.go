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
	if want := map[string]bool{"coinop": true, "meathax": true, "rmcores": true, "slopcore": true, "theypsilon_unofficial_distribution": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	// the db_url identifies a renamed section; the old Coin-Op name still counts
	got = HiddenSources([]DB{
		{ID: "mine", URL: "https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip"},
		{ID: "atrac17/coin-op_collection"},
	})
	if want := map[string]bool{"distribution_mister": true, "jtbindb": true, "rmcores": true, "slopcore": true, "theypsilon_unofficial_distribution": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hidden = %v, want %v", got, want)
	}
	got = HiddenSources([]DB{{ID: "distribution_mister"}, {ID: "jtcores"}, {ID: "coin-opcollection/distribution-misterfpga"}, {ID: "meathax/meatcores"}, {ID: "rmonic79/rmcores"}, {ID: "TheJesusFish/Slop-Core"}, {ID: "theypsilon_unofficial_distribution"}})
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

// Every source the site names has a short chip and a Downloader database
// here, so a new site source cannot ship half-wired: mzgen brings the name,
// this test demands the rest.
func TestSourceTablesComplete(t *testing.T) {
	for src := range gen.SrcNames {
		if _, ok := srcShort[src]; !ok {
			t.Errorf("%s: no short chip in srcShort", src)
		}
		s, ok := sourceDBs[src]
		if !ok {
			t.Errorf("%s: no database in sourceDBs", src)
			continue
		}
		if !strings.HasPrefix(s.url, "https://") || len(s.ids) == 0 {
			t.Errorf("%s: incomplete database entry %+v", src, s)
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
	for src := range sourceDBs {
		if _, ok := gen.SrcNames[src]; !ok {
			t.Errorf("%s: database for a source the site does not name", src)
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
