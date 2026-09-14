package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"testing"
	"time"
)

func TestDefaultViewStartupAndChoices(t *testing.T) {
	ds := data.Ingest(nil, "", time.Now())
	for _, m := range data.ViewOrder {
		a := New(Config{PhysW: 320, PhysH: 240, DefaultSort: m, LastSort: data.SortDebut}, ds, nil)
		if a.Sort() != m {
			t.Fatalf("default %v: got %v", m, a.Sort())
		}
		b := New(Config{PhysW: 320, PhysH: 240, DefaultSort: m, LastSort: data.SortDebut, RememberSort: true}, ds, nil)
		if b.Sort() != data.SortDebut {
			t.Fatal("default overrode remembering")
		}
	}
	a := New(Config{PhysW: 320, PhysH: 240, DefaultSort: data.SortYear, ViewsOff: []string{"year", "updated"}}, ds, nil)
	if a.Sort() != data.SortDebut {
		t.Fatal("disabled default did not fall back")
	}
	e := a.defaultViewEntry()
	if len(e.vals) != len(data.ViewOrder)-2 || e.depth != 1 || !e.child {
		t.Fatal("default choices or indentation")
	}
	a.cfg.RememberSort = true
	for _, e := range a.optionsEntries() {
		if e.kind == "default-view" {
			t.Fatal("default visible while remembering")
		}
	}
	a.cfg.RememberSort = false
	entries := a.optionsEntries()
	for i, e := range entries {
		if e.kind == "default-view" {
			if entries[i-1].kind != "remember-sort" || entries[i-2].kind != "views" {
				t.Fatal("hierarchy order")
			}
			return
		}
	}
	t.Fatal("missing default row")
}
