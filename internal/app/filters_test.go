package app

import (
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
