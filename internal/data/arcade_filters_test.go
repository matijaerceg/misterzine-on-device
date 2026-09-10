package data

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestStructuredControlsAndLegacyFeed(t *testing.T) {
	var rows []Row
	err := json.Unmarshal([]byte(`[
		{"title":"Pac-Man", "ctl":"4-way", "buttons":0},
		{"title":"Legacy unknown", "ctl":"4-way"},
		{"title":"Arkanoid", "ctl":"1 button", "spc":"Spinner", "buttons":1},
		{"title":"Pong", "spc":"Positional", "buttons":0},
		{"title":"Computer Space", "ctl":"buttons only · 4 buttons"},
		{"title":"Count alone", "ctl":"4 buttons"},
		{"title":"Combined", "ctl":"8-way · 2 buttons", "spc":"Rotary"},
		{"title":"Duplicate", "ctl":"Trackball", "spc":"trackball"}
	]`), &rows)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{{"4-way", "0"}, {"4-way", ""}, {"spinner", "1"}, {"positional", "0"}, {"buttons only", "4"}, {"", "4"}, {"8-way + rotary", "2"}, {"trackball", ""}}
	for i := range rows {
		controls, buttons := rows[i].ControlFacets()
		if got := [2]string{controls, buttons}; got != want[i] {
			t.Errorf("%s: got %v want %v", rows[i].Title, got, want[i])
		}
	}
}

func TestArcadeFiltersLeaveSystemCoresAndProvisionalValues(t *testing.T) {
	ds := Ingest([]Row{
		{K: "vertical", Base: "Arcade", Rot: "Vertical", Res: "15kHz", Plr: "2", Genre: "Shooter", Ctl: "8-way · 2 buttons", Prov: []string{"rot", "plr", "ctl"}},
		{K: "unknown", Base: "Arcade"},
		{K: "console", Base: "Console"}, {K: "computer", Base: "Computer"}, {K: "other", Base: "Other"},
	}, "", time.Time{})
	f := Filters{RotOff: map[string]bool{"": true}, ResOff: map[string]bool{"": true}, PlrOff: map[string]bool{"": true}, GenreOff: map[string]bool{"": true}, DirectionsOff: map[string]bool{"": true}, ButtonsOff: map[string]bool{"": true}}
	if got := keysOf(ds, Apply(ds, []int{0, 1, 2, 3, 4}, &f, nil, nil, nil)); !reflect.DeepEqual(got, []string{"vertical", "console", "computer", "other"}) {
		t.Fatal(got)
	}
	for _, facet := range []map[string]int{ds.Facets.Rot, ds.Facets.Res, ds.Facets.Plr, ds.Facets.Genre, ds.Facets.Directions, ds.Facets.Buttons} {
		if facet[""] != 1 {
			t.Fatalf("system cores inflated unknown count: %v", facet)
		}
	}
	f.BaseOff = map[string]bool{"Console": true, "Computer": true, "Other": true}
	if got := keysOf(ds, Apply(ds, []int{0, 1, 2, 3, 4}, &f, nil, nil, nil)); !reflect.DeepEqual(got, []string{"vertical"}) {
		t.Fatal(got)
	}
}
