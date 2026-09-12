package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// B from the list reopens Options on the row B left from, even after the
// rows shift, while Filters still opens at its top.
func TestOptionsReopensOnLastRow(t *testing.T) {
	rows := []data.Row{{K: "a", Title: "Alpha"}, {K: "b", Title: "Beta"}}
	a := New(Config{PhysW: 320, PhysH: 240, Launcher: func() bool { return true }}, data.Ingest(rows, "", time.Now()), nil)
	a.actList(platform.KeyBack)
	if a.screen != ScreenOptions || a.panel.cursor > 1 { // the first row is the Data: heading
		t.Fatal("first visit not at the top")
	}
	for i := 0; i < 9; i++ {
		a.actPanel(platform.KeyDown)
	}
	kind := a.panel.entries[a.panel.cursor].kind
	if kind == "" {
		t.Fatal("test row has no kind")
	}
	a.actPanel(platform.KeyBack)
	a.actList(platform.KeyDown)
	a.actList(platform.KeyBack)
	if a.screen != ScreenOptions || a.panel.entries[a.panel.cursor].kind != kind {
		t.Fatalf("reopened on %q, want %q", a.panel.entries[a.panel.cursor].kind, kind)
	}
	// the row is found by identity when rows appear above it
	a.actPanel(platform.KeyBack)
	a.SetAppUpdate("v9.9.9")
	a.actList(platform.KeyBack)
	if a.panel.entries[a.panel.cursor].kind != kind {
		t.Fatalf("after a new row: %q, want %q", a.panel.entries[a.panel.cursor].kind, kind)
	}
	a.actPanel(platform.KeyBack)
	a.actList(platform.KeyTab)
	fresh := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	fresh.actList(platform.KeyTab)
	if a.screen != ScreenFilter || a.panel.cursor != fresh.panel.cursor {
		t.Fatal("Filters did not open at the top")
	}
}
