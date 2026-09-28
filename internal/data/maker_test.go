package data

import (
	"reflect"
	"testing"
	"time"
)

func TestMakerStemAndKey(t *testing.T) {
	for raw, want := range map[string]string{
		"Sega":                                "Sega",
		"Sega / Westone":                      "Sega",
		"Sega/Gremlin":                        "Sega",
		"Sega Enterprises/Gremlin Industries": "Sega",
		"Cave (Capcom license)":               "Cave",
		"CAVE":                                "CAVE",
		"Taito Corporation Japan":             "Taito",
		"Taito America Corporation":           "Taito",
		"Data East USA":                       "Data East",
		"Data East Corporation (licensed from X)": "Data East",
		"Alpha Denshi Co.":                        "Alpha Denshi",
		"Atari Games":                             "Atari",
		"Williams Electronics":                    "Williams",
		"Technos Japan (Taito America License)":   "Technos",
		"Bally Midway":                            "Bally Midway",
		"Dave Nutting Associates":                 "Dave Nutting Associates",
		"bootleg (Darksoft)":                      "bootleg",
		"Crux/Kyugo?":                             "Crux",
		"DIY (Grant Searle)":                      "DIY",
		"A wave inc. (Able license)":              "A wave",
		"Rait Electronics Ltd":                    "Rait",
		"Japan":                                   "Japan",
		"":                                        "",
		"  ":                                      "",
		"Capcom / Cave / Victor Interactive Software": "Capcom",
	} {
		if got := MakerStem(raw); got != want {
			t.Errorf("MakerStem(%q) = %q, want %q", raw, got, want)
		}
	}
	if MakerKey("CAVE") != MakerKey("Cave (Capcom license)") || MakerKey("TEHKAN") != "tehkan" || MakerKey("Sega") != MakerKey("Sega Enterprises Ltd. / Foo") {
		t.Fatal("keys should fold case and suffixes")
	}
	if MakerKey("Bootleg") != MakerKey("bootleg") {
		t.Fatal("bootleg spellings should share a group")
	}
}

func TestMakerOrderGroupsAndLabels(t *testing.T) {
	rows := []Row{
		{K: "1", Title: "Zed", Manufacturer: "Taito Corporation"},
		{K: "2", Title: "Alpha", Manufacturer: "Taito"},
		{K: "3", Title: "Beta", Manufacturer: "Taito Corporation Japan"},
		{K: "4", Title: "Gamma", Manufacturer: ""},
		{K: "5", Title: "Delta", Manufacturer: "bootleg"},
		{K: "6", Title: "Echo", Manufacturer: "Bootleg"},
		{K: "7", Title: "Foxtrot", Manufacturer: "Cave (Capcom license)"},
		{K: "8", Title: "Golf", Manufacturer: "CAVE"},
		{K: "9", Title: "Hotel", Manufacturer: "CAVE"},
		{K: "10", Title: "India", Manufacturer: "Atari Games"},
	}
	ds := Ingest(rows, "", time.Time{})
	// the most common stem names the group; a tie goes to the capital
	for k, want := range map[string]string{"1": "Taito", "2": "Taito", "3": "Taito", "4": "", "5": "Bootleg", "6": "Bootleg", "7": "CAVE", "8": "CAVE", "10": "Atari"} {
		if got := ds.Der[ds.Index(k)].Maker; got != want {
			t.Errorf("row %s maker %q, want %q", k, got, want)
		}
	}
	got := []string{}
	for _, i := range ds.Order(SortMaker) {
		got = append(got, rows[i].K)
	}
	// makers A-Z, titles A-Z inside, the unknown maker last
	want := []string{"10", "5", "6", "7", "8", "9", "2", "3", "1", "4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("maker order %v, want %v", got, want)
	}
}

func TestSortNamesRoundTrip(t *testing.T) {
	for _, m := range ViewOrder {
		back, ok := ParseSort(m.Name())
		if !ok || back != m || !m.Valid() {
			t.Errorf("%v: name %q parses to %v %v", m, m.Name(), back, ok)
		}
	}
	if _, ok := ParseSort("bogus"); ok || SortMode(99).Valid() || SortMode(-1).Valid() {
		t.Fatal("unknown names and values must be rejected")
	}
	if NextSort(SortAlphabetical) != SortMaker || NextSort(SortMaker) != SortFavorites {
		t.Fatal("Maker sits between A-Z and Favorites in the cycle")
	}
}
