package mgl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := os.Mkdir(menu, 0755); err != nil {
		t.Fatal(err)
	}
	for _, g := range games {
		content := sample("_Console/NeoGeo", `<file type="f" index="1" path="`+g.ID+`.neo"/>`)
		if err := os.WriteFile(filepath.Join(menu, g.ID+".mgl"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Scan(card, NeoGeoCatalogue())
	if err != nil || len(result.Problems) != 0 || len(result.Rows) != len(games) {
		t.Fatalf("scan: %d rows for %d games, %v, %v", len(result.Rows), len(games), err, result.Problems)
	}
	for _, r := range result.Rows {
		if !strings.HasSuffix(r.MRA, "/"+r.SN+".mgl") {
			t.Fatalf("changed launch target: %+v", r)
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
