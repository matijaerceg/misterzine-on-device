package data

import (
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
)

func TestCoreGroupsAndOrder(t *testing.T) {
	cat := []Row{
		{K: "sf2", Title: "Street Fighter II", Base: "Arcade", Core: "jtcps1"},
		{K: "ffight", Title: "Final Fight", Base: "Arcade", Core: "jtcps1"},
		{K: "ddsom", Title: "Dungeons & Dragons: Shadow over Mystara", Base: "Arcade", Core: "jtcps2"},
		{K: "gwarrior", Title: "Galactic Warriors", Base: "Arcade", Core: "BubSys"},
		{K: "twinbee", Title: "TwinBee", Base: "Arcade", Core: "BubSysROM"},
		{K: "galaga", Title: "Galaga", Base: "Arcade", Core: "galaga", SN: "galaga"},
		{K: "congo", Title: "Congo Bongo", Base: "Arcade", Core: "CongoBongo"},
		{K: "mystery", Title: "Mystery", Base: "Arcade"},
		{K: "snes", Title: "SNES", Base: "Console", Core: "SNES"},
		{K: "c64", Title: "Commodore 64", Base: "Computer", Core: "C64"},
		{K: "chess", Title: "Chess", Base: "Other", Core: "Chess"},
		{K: "nes", Title: "NES", Base: "Console", Core: "NES"},
	}
	// a card's own CPS-1 game joins the core's group; a standin for Galaga
	// on another core rides with Galaga and leaves it a single-game core
	sf2ce := localRow("sf2ce", "jtcps1", "_Arcade/sf2ce.mra")
	sf2ce.Title = "Street Fighter II' Champion Edition"
	stand := localRow("galaga", "galagamw", "_Arcade/Galaga (bazset).mra")
	stand.Title, stand.Standin = "Galaga (bazset)", true
	ds := Ingest(MergeLocal(cat, []Row{sf2ce, stand}), "h", time.Time{})
	for k, want := range map[string]string{
		"sf2": "Capcom CPS-1", "ffight": "Capcom CPS-1", "local:sf2ce": "Capcom CPS-1",
		"gwarrior": "Konami Bubble System", "twinbee": "Konami Bubble System", // two rbfs, one label
		"ddsom": CoreSingleGame, "galaga": CoreSingleGame, "congo": CoreSingleGame, // a named core with one game too
		"mystery": CoreUnknown,
		"snes":    "Console cores", "nes": "Console cores", "c64": "Computer cores", "chess": "Other cores",
	} {
		if got := ds.Der[ds.Index(k)].CoreGroup; got != want {
			t.Errorf("%s: group %q, want %q", k, got, want)
		}
	}
	got := []string{}
	for _, i := range ds.Order(SortCore) {
		got = append(got, ds.Rows[i].K)
	}
	// the multi-game cores A-Z, the single-game ones, no core, then the
	// console, computer and other cores; titles A-Z inside each
	want := []string{"ffight", "sf2", "local:sf2ce", "gwarrior", "twinbee",
		"congo", "ddsom", "galaga", "local:galaga", "mystery", "nes", "snes", "c64", "chess"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("core order %v, want %v", got, want)
	}
}

// The Core view is the Patreon beta's: a free build does not know it by
// value or by name, so a saved one falls back like any unknown view.
func TestCoreViewIsTheBetasOnly(t *testing.T) {
	free := []SortMode{SortUpdated, SortDebut, SortYear, SortAlphabetical, SortMaker, SortFavorites, SortRecents}
	if SortCore.Valid() || !reflect.DeepEqual(ViewOrder(), free) {
		t.Fatalf("free build: core valid %v, views %v", SortCore.Valid(), ViewOrder())
	}
	if m, ok := ParseSort("core"); ok || m != SortUpdated {
		t.Fatalf("free build parses core as %v %v", m, ok)
	}
	restore := beta.Set(true)
	defer restore()
	want := []SortMode{SortUpdated, SortDebut, SortYear, SortAlphabetical, SortMaker, SortCore, SortFavorites, SortRecents}
	if !SortCore.Valid() || !reflect.DeepEqual(ViewOrder(), want) {
		t.Fatalf("beta: core valid %v, views %v", SortCore.Valid(), ViewOrder())
	}
	for _, m := range ViewOrder() {
		if back, ok := ParseSort(m.Name()); !ok || back != m {
			t.Errorf("beta: %v named %q parses to %v %v", m, m.Name(), back, ok)
		}
	}
	if SortMode(SortCore+1).Valid() || SortCore.String() != "Core" {
		t.Fatal("beta: nothing past Core, and Core is its name")
	}
}
