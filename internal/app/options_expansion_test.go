package app

import (
	"strings"
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
	if has("refresh") || has("sources") || has("rotation") || has("scroll") || has("launcher") {
		t.Fatal("startup must collapse every section")
	}
	a.actPanel(platform.KeyEnter)
	if !has("refresh") {
		t.Fatal("Data did not open")
	}
	a.actPanel(platform.KeyEnter)
	if has("refresh") || a.panel.entries[a.panel.cursor].value != "Data" {
		t.Fatal("collapse lost selection")
	}
	for _, kind := range []string{"credits", "quit"} {
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
	if fresh.optionSectionOpen("Data") || fresh.optionSectionOpen("List") {
		t.Fatal("expansion survived restart")
	}
}

// Access controls belong to Operation; ROM report belongs to Data even locked.
func TestAccessOptionsSections(t *testing.T) {
	a, _, _, _ := shotsApp()
	a.cfg.ShowBetaFeatures = true
	a.cfg.AccessMonth = 0
	a.optionsOpen = map[string]bool{}
	has := func(text string) bool {
		for _, e := range a.visibleOptionsEntries() {
			if strings.HasPrefix(e.text, text) {
				return true
			}
		}
		return false
	}
	for _, text := range []string{yourAccessRow, "Show beta features", "Troubleshooting", "ROM report"} {
		if has(text) {
			t.Fatalf("collapsed sections expose %s", text)
		}
	}
	a.optionsOpen["Operation"] = true
	for _, text := range []string{yourAccessRow, "Show beta features", "Troubleshooting"} {
		if !has(text) {
			t.Fatalf("Operation lacks %s", text)
		}
	}
	if has("ROM report") {
		t.Fatal("Operation exposes ROM report")
	}
	a.optionsOpen = map[string]bool{"Data": true}
	if !has("ROM report") {
		t.Fatal("Data lacks locked ROM report")
	}
	a.cfg.ShowBetaFeatures = false
	if has("ROM report") {
		t.Fatal("beta-off exposes ROM report")
	}
}
