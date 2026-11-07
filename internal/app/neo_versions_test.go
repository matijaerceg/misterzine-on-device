package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestNeoGeoVersionsFromListSearchAndSavedChoice(t *testing.T) {
	f, err := os.Open("../../testdata/neogeo-versions.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := data.DecodeRows(f)
	if err != nil {
		t.Fatal(err)
	}
	ds := data.Ingest(rows, "", time.Now())
	var launched string
	config := Config{PhysW: 320, PhysH: 240, NeoGeoSupport: true, NeoGeoView: true,
		Launch: func(p string) { launched = p }, Versions: map[string]string{},
		Favorites: map[string]bool{rows[0].K: true}}
	a := New(config, ds, nil)
	for _, mode := range []data.SortMode{data.SortAlphabetical, data.SortCore, data.SortNeoGeo} {
		a.SetSort(mode)
		if a.shown != 2 || a.total != 2 {
			t.Fatalf("%v: %d/%d families", mode, a.shown, a.total)
		}
	}
	a.MoveToKey(rows[0].K) // an old file key still reaches the family
	a.actList(platform.KeyStart)
	if launched != rows[1].MRA {
		t.Fatalf("default launched %q", launched)
	}
	a.actList(platform.KeyEnter)
	a.actDetails(platform.KeyRight)
	if entries := a.launchEntries(&rows[1], 1); entries[a.detail.pick].label != "Home cartridge" {
		t.Fatal(entries)
	}
	a.actDetails(platform.KeyBack)
	a = New(config, ds, nil)
	a.MoveToKey(rows[1].K)
	a.actList(platform.KeyStart)
	if launched != rows[0].MRA {
		t.Fatalf("restart launched %q", launched)
	}

	a.setSearch("king of gladiator")
	if a.shown != 1 || a.CursorKey() != rows[3].K || a.facetCounts("src")[data.SrcLocal] != 1 {
		t.Fatalf("variant search: %v", viewKeys(a))
	}
	a.actList(platform.KeyEnter)
	entries := a.launchEntries(&rows[3], 3)
	if e := entries[a.detail.pick]; e.path != rows[3].MRA || e.label != "King of Gladiator (bootleg)" {
		t.Fatal(e)
	}
	var info string
	for _, line := range a.detailLines(&rows[3], &ds.Der[3], 3) {
		info += line.text + "\n"
	}
	if !strings.Contains(info, rows[3].MRA) {
		t.Fatal(info)
	}
	a.actDetails(platform.KeyBack)
	a.actList(platform.KeyStart)
	if launched != rows[3].MRA {
		t.Fatalf("search launched %q", launched)
	}
	a.setSearch("")
	a.actList(platform.KeyStart)
	if launched != rows[3].MRA {
		t.Fatal("variant launch not remembered")
	}
	a.SetSort(data.SortFavorites)
	if a.shown != 1 || !a.familyFavorite(&rows[1]) {
		t.Fatal("old variant star lost")
	}
	a.toggleFavorite()
	if a.shown != 0 {
		t.Fatal("family still starred")
	}
	a.SetSort(data.SortAlphabetical)
	a.MoveToKey(rows[1].K)
	a.toggleFavorite()
	a.SetData(data.Ingest(rows[:1], "", time.Now()), nil)
	a.SetSort(data.SortFavorites)
	if a.shown != 1 {
		t.Fatal("family star lost when base cartridge removed")
	}
	a.actList(platform.KeyStart)
	if launched != rows[0].MRA {
		t.Fatal(launched)
	}
	a.SetData(data.Ingest(rows[1:], "", time.Now()), nil)
	a.actList(platform.KeyStart)
	if launched != rows[1].MRA {
		t.Fatal("removed version did not fall back to arcade", launched)
	}
	a.setSearch("king of gladiator")
	a.SetSort(data.SortAlphabetical)
	a.SetData(data.Ingest(rows[:3], "", time.Now()), nil)
	if a.shown != 0 {
		t.Fatal("search matched an absent variant")
	}
}
