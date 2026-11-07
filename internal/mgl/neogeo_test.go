package mgl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestNeoGeoCatalogueCoverageAndAliases(t *testing.T) {
	c := NeoGeoCatalogue()
	for target, want := range map[string]string{
		"garou.neo": "garou", "Garou - Mark of the Wolves.neo": "garou",
		"kof97h.zip": "kof97", "mslug5.rom": "mslug5", "tws96": "twsoc96",
		"Darksoft/garoup": "garou", "pack.zip/tphuntrh.neo": "tophuntr",
		"NAM-1975.neo": "nam1975", "Samurai Shodown V Special.neo": "samsh5sp",
		"Neo Mr. Do!": "neomrdo", "SNK vs. Capcom - SVC Chaos": "svc",
		"XenoCrisis.neo": "xenocris", "kof2003.neo": "kof2003",
		"/games/NEOGEO/3 Count Bout (3countb).neo":   "3countb",
		"pack.zip/Custom title (Japan) (KOF97H).NEO": "kof97",
		"My soccer game (tws96).rom":                 "twsoc96",
	} {
		d := Document{RBF: "_Console/NeoGeo", Files: []File{{Type: "f", Index: "1", Path: target}}}
		if ok, why := d.Cartridge(); !ok {
			t.Fatalf("%s: %s", target, why)
		}
		if g, ok := c.Match(d); !ok || g.ID != want {
			t.Errorf("%s: got %+v, %v; want %s", target, g, ok, want)
		}
	}
	for _, target := range []string{
		"homebrew.neo", "montest.neo", "badapple.neo", "totc.neo", "lasthope.neo", "zintrkcd.neo", "sonic.neo",
		"3 Count Bout (USA).neo", "Unknown (Fire Suplex).neo", "Unknown (lasthope).neo",
		"Unknown (3countb) (USA).neo", "Unknown (3countb)/other.neo", "Unknown (3countb.neo",
		"Unknown (3countb).chd", "(3countb).neo",
	} {
		if g, ok := c.Match(Document{Files: []File{{Path: target}}}); ok {
			t.Errorf("non-arcade/unknown %s matched %s", target, g.ID)
		}
	}
}

func TestExactNeoGeoNameWinsOverSetSuffix(t *testing.T) {
	c, err := ReadCatalogue(strings.NewReader(`[{"id":"one","title":"Title (two)"},{"id":"two","title":"Other title"}]`))
	if err != nil {
		t.Fatal(err)
	}
	g, ok := c.Match(Document{Files: []File{{Path: "Title (two).zip"}}})
	if !ok || g.ID != "one" {
		t.Fatalf("got %v, %v; want exact title", g, ok)
	}
}

func TestMatchedNeoGeoVariant(t *testing.T) {
	for _, tc := range []struct{ target, set, label string }{
		{"aof2.neo", "aof2", "Arcade cartridge"},
		{"Library.zip/Art of Fighting 2 (AOF2A).neo", "aof2a", "Home cartridge"},
		{"Ryuuko no Ken 2 (NGH-056).neo", "aof2a", "Home cartridge"},
		{"King of Gladiator.neo", "kog", "King of Gladiator (bootleg)"},
	} {
		g, ok := NeoGeoCatalogue().matchTarget(tc.target)
		if !ok || g.set != tc.set || g.variant.Label != tc.label {
			t.Errorf("%s: %q %q %v", tc.target, g.set, g.variant.Label, ok)
		}
	}
}

// Exercise every family through XML parsing and real menu discovery, not only
// the map lookup. No ROMs/cores are needed for this discovery-only test.
func TestScanWholeNeoGeoCatalogue(t *testing.T) {
	var games []Game
	if err := json.Unmarshal(neoGeoJSON, &games); err != nil {
		t.Fatal(err)
	}
	if len(games) < 150 {
		t.Fatalf("catalogue unexpectedly shrank to %d games", len(games))
	}
	card := t.TempDir()
	menu := filepath.Join(card, "_NeoGeo")
	folder := filepath.Join(card, "games", "NeoGeo")
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(menu, 0755); err != nil {
		t.Fatal(err)
	}
	for _, g := range games {
		if err := os.WriteFile(filepath.Join(folder, g.ID+".neo"), nil, 0644); err != nil {
			t.Fatal(err)
		}
		content := sample("_Console/NeoGeo", `<file type="f" index="1" path="`+g.ID+`.neo"/>`)
		if err := os.WriteFile(filepath.Join(menu, g.ID+".mgl"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Scan(card, NeoGeoCatalogue())
	if err != nil || len(result.Problems) != 0 || len(result.Rows) != len(games) {
		t.Fatalf("scan: %d rows for %d games, %v, %v", len(result.Rows), len(games), err, result.Problems)
	}
	byID := map[string]data.Row{}
	for _, r := range result.Rows {
		wantKey := r.SN
		switch r.SN {
		case "gladmort", "cyborgforce", "knightsch", "xenocris", "captainbarrel":
			wantKey = "fbneo_" + r.SN
		}
		wantSlots := []string{"ltitle", "lsnap"}
		if r.SN == "gladmort" {
			wantSlots = []string{"lsnap"}
		}
		if r.Img != wantKey || !reflect.DeepEqual(r.ImgSlots, wantSlots) || r.ImgW != 320 || r.ImgH != 224 {
			t.Fatalf("%s: wrong screenshot identity, slots or size: %+v", r.SN, r)
		}
		if !strings.HasSuffix(r.MRA, "/"+r.SN+".mgl") {
			t.Fatalf("changed launch target: %+v", r)
		}
		if r.Year == "" || r.Manufacturer == "" || r.Genre == "" || r.Plr == "" || r.Rot == "" || r.Res != "15kHz" || (r.Ctl == "" && r.Spc == "") {
			t.Fatalf("missing arcade details for %s: %+v", r.SN, r)
		}
		byID[r.SN] = r
	}
	// Scan the cartridges independently so shortcut precedence doesn't hide them.
	var direct Result
	if err := ScanCartridges(card, &direct); err != nil || len(direct.Rows) != len(games) {
		t.Fatalf("folder scan: %d rows, %v", len(direct.Rows), err)
	}
	for _, r := range direct.Rows {
		want := byID[r.SN]
		want.K, want.MRA = r.K, r.MRA
		if !r.IsNeoGeoFile() || !reflect.DeepEqual(r, want) {
			t.Fatalf("%s: folder and shortcut disagree: %+v / %+v", r.SN, r, want)
		}
	}
	slug := byID["mslug"]
	if slug.Year != "1996" || slug.Manufacturer != "Nazca" || slug.Genre != "Run 'n gun" || slug.Plr != "2 (simultaneous)" || slug.Ctl != "8-way · 3 buttons" || slug.Buttons == nil || *slug.Buttons != 3 {
		t.Fatalf("incorrect Metal Slug details: %+v", slug)
	}
	if byID["irrmaze"].Spc != "Trackball" || byID["janshin"].Mat != 1 {
		t.Fatal("special controls or mature classification lost")
	}
	for id, year := range map[string]string{"knightsch": "2014", "xenocris": "2021", "cyborgforce": "2024", "captainbarrel": "2024", "gladmort": "2025"} {
		if byID[id].Year != year {
			t.Errorf("%s: want MVS release year %s, got %s", id, year, byID[id].Year)
		}
	}
}

func TestPilotIdentitiesRemainStable(t *testing.T) {
	f, err := os.Open("../../testdata/mgl/arcade-pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	old, err := ReadCatalogue(f)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range old.names {
		got, ok := NeoGeoCatalogue().names[name]
		if !ok || got.ID != want.ID || got.Title != want.Title {
			t.Fatalf("old identity %s changed: %+v -> %+v", name, want, got)
		}
	}
}
