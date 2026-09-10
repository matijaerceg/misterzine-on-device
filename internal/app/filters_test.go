package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestControlFilterSelectionAndClear(t *testing.T) {
	now := time.Now()
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "a", Title: "A", Ctl: "8-way · 2 buttons"},
		{K: "b", Title: "B", Ctl: "4-way · 3 buttons"},
	}, "", now), nil)
	a.openPanel(ScreenFilter)
	choose := func(kind, value string, header bool) {
		t.Helper()
		for i, e := range a.panel.entries {
			if e.kind == kind && e.value == value && e.header == header {
				a.panel.cursor = i
				a.actPanel(platform.KeyEnter)
				return
			}
		}
		t.Fatalf("missing %s/%s", kind, value)
	}
	choose("directions", "", true) // deselect the whole section
	if len(a.view) != 0 {
		t.Fatal("direction header did not hide all")
	}
	choose("directions", "8-way", false)
	if len(a.view) != 1 || a.CursorKey() != "a" {
		t.Fatal("direction selection")
	}
	choose("buttons", "2", false)
	if len(a.view) != 0 {
		t.Fatal("buttons must combine with directions")
	}
	choose("clear", "", false)
	if len(a.view) != 2 || a.filters.Active() {
		t.Fatal("clear did not reset controls")
	}
	lastHeader := ""
	for _, e := range a.panel.entries {
		if e.header {
			lastHeader = e.text
		}
	}
	if lastHeader != "Players" {
		t.Fatalf("last section is %q", lastHeader)
	}
}

func TestResolutionFilterSelectionAndRestore(t *testing.T) {
	rows := []data.Row{
		{K: "a", Title: "A", Res: "15kHz", Rot: "Horizontal"},
		{K: "b", Title: "B", Res: "31kHz", Rot: "Horizontal"},
		{K: "c", Title: "C"},
		{K: "d", Title: "D", Res: "15kHz", Rot: "Vertical"},
	}
	ds := data.Ingest(rows, "", time.Now())
	a := New(Config{PhysW: 320, PhysH: 240}, ds, nil)
	a.openPanel(ScreenFilter)
	choose := func(kind, value string, header bool) {
		t.Helper()
		for i, e := range a.panel.entries {
			if e.kind == kind && e.value == value && e.header == header {
				a.panel.cursor = i
				a.actPanel(platform.KeyEnter)
				return
			}
		}
		t.Fatalf("missing %s/%s", kind, value)
	}
	for _, e := range a.panel.entries {
		if e.kind == "res" && !e.header {
			if !e.checked || e.count != ds.Facets.Res[e.value] {
				t.Fatalf("bad resolution entry: %+v", e)
			}
			if e.value == "" && e.text != "unknown" {
				t.Fatal("missing unknown label")
			}
		}
	}
	choose("res", "", true)
	if len(a.view) != 0 || !a.filters.Active() {
		t.Fatal("header must hide every resolution")
	}
	choose("res", "15kHz", false)
	if len(a.view) != 2 {
		t.Fatal("15kHz selection")
	}
	choose("rot", "v", false)
	if len(a.view) != 1 || a.CursorKey() != "a" {
		t.Fatal("resolution must combine with rotation")
	}
	saved, err := json.Marshal(a.filters)
	if err != nil {
		t.Fatal(err)
	}
	var restored data.Filters
	if err := json.Unmarshal(saved, &restored); err != nil {
		t.Fatal(err)
	}
	b := New(Config{PhysW: 320, PhysH: 240}, ds, nil)
	b.SetFilters(restored)
	if len(b.view) != 1 || b.CursorKey() != "a" {
		t.Fatal("saved filters changed results")
	}
	// A resolution added by a later catalogue remains visible by default.
	row := data.Row{Res: "24kHz"}
	if !restored.Pass(&row, &data.Derived{}, data.StatusUnknown, false, false) {
		t.Fatal("new resolution hidden")
	}
	choose("res", "", false)
	if len(a.view) != 2 {
		t.Fatal("unknown selection")
	}
	choose("res", "", true)
	if len(a.view) != 3 {
		t.Fatal("header must restore all resolutions")
	}
	choose("res", "31kHz", false)
	choose("clear", "", false)
	if len(a.view) != 4 || a.filters.Active() {
		t.Fatal("clear must reset resolution")
	}
}
