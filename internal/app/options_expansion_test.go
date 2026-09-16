package app

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Existing setting tests inspect every row; first reveal them through the
// same navigation users use, preserving the current selection.
func expandOptionsForTest(a *App) {
	if a.screen != ScreenOptions {
		return
	}
	was := a.panel.entries[a.panel.cursor]
	a.actPanel(platform.KeyHome)
	for i := 0; i < 5; i++ {
		a.actPanel(platform.KeyRight)
		a.actPanel(platform.KeyPageDown)
	}
	for i, e := range a.panel.entries {
		if e.kind == was.kind && e.value == was.value {
			a.panel.cursor = i
			break
		}
	}
}

func TestOptionsExpansion(t *testing.T) {
	a, _, _, _ := shotsApp()
	a.openOptions()
	has := func(kind string) bool {
		for _, e := range a.panel.entries {
			if e.kind == kind {
				return true
			}
		}
		return false
	}
	if !has("refresh") || has("sources") || has("rotation") || has("scroll") || has("launcher") {
		t.Fatal("startup must reveal only Data")
	}
	a.actPanel(platform.KeyEnter)
	if has("refresh") || a.panel.entries[a.panel.cursor].value != "Data" {
		t.Fatal("collapse lost selection")
	}
	for _, kind := range []string{"troubleshooting", "credits", "quit"} {
		if !has(kind) {
			t.Fatalf("collapsed page hides %s", kind)
		}
	}
	a.actPanel(platform.KeyPageDown)
	a.actPanel(platform.KeyRight)
	a.actPanel(platform.KeyPageDown)
	a.actPanel(platform.KeyEnter)
	if !has("sources") || !has("rotation") {
		t.Fatal("sections must open independently")
	}
	a.actPanel(platform.KeyLeft)
	if has("rotation") || !has("sources") {
		t.Fatal("closing one section changed another")
	}
	a.actPanel(platform.KeyBack)
	a.openOptions()
	if a.panel.entries[a.panel.cursor].value != "Display" || !has("sources") || has("refresh") {
		t.Fatal("reopening lost browsing state")
	}
	a.actPanel(platform.KeyBack)
	a.openPanel(ScreenFilter)
	a.openOptions()
	a.actPanel(platform.KeyRight)
	a.actPanel(platform.KeyBack)
	a.openOptions()
	if !has("rotation") || !has("sources") || has("refresh") {
		t.Fatal("Filters restored stale Options expansion")
	}
	fresh, _, _, _ := shotsApp()
	if !fresh.optionSectionOpen("Data") || fresh.optionSectionOpen("List") {
		t.Fatal("expansion survived restart")
	}
}
