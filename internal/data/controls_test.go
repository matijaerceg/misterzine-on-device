package data

import (
	"reflect"
	"testing"
	"time"
)

func TestControlFacets(t *testing.T) {
	for _, tc := range []struct{ ctl, directions, buttons string }{
		{"8-way · 3 buttons", "8-way", "3"},
		{"2-way horizontal · 1 button", "2-way horizontal", "1"},
		{"2-way vertical · 20 buttons", "2-way vertical", "20"},
		{"Double 8-way · 2 buttons", "8-way double", "2"},
		{"8-way double · 2 buttons", "8-way double", "2"},
		{"4-way diagonal", "4-way diagonal", ""},
		{"dial · 2 buttons", "dial", "2"},
		{"Dial · 2 buttons", "dial", "2"},
		{"buttons · 5 buttons", "buttons only", "5"},
		{"8-way,Positional · 2 buttons", "8-way + positional", "2"},
		{"2 buttons", "", "2"},
		{"8-way · 0 buttons", "8-way", "0"},
		{"2", "", ""},
		{"", "", ""},
	} {
		d, b := ControlFacets(tc.ctl)
		if d != tc.directions || b != tc.buttons {
			t.Errorf("%q => %q, %q", tc.ctl, d, b)
		}
	}
}

func TestControlFiltersCombineAndKeepUnspecified(t *testing.T) {
	ds := Ingest([]Row{
		{K: "a", Base: "Arcade", Ctl: "8-way · 2 buttons"},
		{K: "b", Base: "Arcade", Ctl: "4-way · 2 buttons"},
		{K: "c", Base: "Arcade", Ctl: "8-way · 3 buttons"},
		{K: "d", Base: "Arcade", Ctl: ""},
	}, "", time.Time{})
	f := &Filters{DirectionsOff: map[string]bool{"4-way": true}, ButtonsOff: map[string]bool{"3": true}}
	if !f.Active() {
		t.Fatal("controls alone must activate filtering")
	}
	if got := Apply(ds, []int{0, 1, 2, 3}, f, nil, nil, nil); !reflect.DeepEqual(got, []int{0, 3}) {
		t.Fatal(got)
	}
	f.ButtonsOff[""] = true
	if got := Apply(ds, []int{0, 1, 2, 3}, f, nil, nil, nil); !reflect.DeepEqual(got, []int{0}) {
		t.Fatal(got)
	}
	if ds.Facets.Directions["8-way"] != 2 || ds.Facets.Buttons["2"] != 2 || ds.Facets.Buttons[""] != 1 {
		t.Fatal("incorrect control counts")
	}
}
