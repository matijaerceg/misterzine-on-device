package app

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// L and R (PageUp/PageDown) in Options jump to the first row of the
// previous or next section, as they jump letters on the list and
// headings in Filters; from inside a section L goes to the previous
// section; the jumps stop at either end; Home and End still go to the
// first and last row.
func TestOptionsShouldersJumpSections(t *testing.T) {
	a, _, _, _ := shotsApp()
	a.openPanel(ScreenOptions)
	kind := func() string { return a.panel.entries[a.panel.cursor].kind }
	press := func(k platform.Key) {
		if !a.actPanel(k) {
			t.Fatalf("%v ignored on %s", k, kind())
		}
	}
	if kind() != "refresh" {
		t.Fatalf("Options opens on %s", kind())
	}
	for _, want := range []string{"sources", "follow-rotation", "button-labels", "launcher", "launcher"} {
		press(platform.KeyPageDown)
		if kind() != want {
			t.Fatalf("R landed on %s, want %s", kind(), want)
		}
	}
	press(platform.KeyDown)
	press(platform.KeyDown)
	if kind() != "return-after-game" {
		t.Fatalf("down twice from the shortcut: %s", kind())
	}
	for _, want := range []string{"button-labels", "follow-rotation", "sources", "refresh", "refresh"} {
		press(platform.KeyPageUp)
		if kind() != want {
			t.Fatalf("L landed on %s, want %s", kind(), want)
		}
	}
	press(platform.KeyEnd)
	if kind() != "quit" {
		t.Fatalf("End landed on %s", kind())
	}
	press(platform.KeyHome)
	if kind() != "refresh" {
		t.Fatalf("Home landed on %s", kind())
	}
	// a hidden section row (Screensaver brightness only with the
	// screenshots) changes nothing about the jumps
	a.cfg.SaverStyle = "shots"
	a.buildPanel()
	press(platform.KeyPageDown)
	press(platform.KeyPageDown)
	press(platform.KeyPageDown)
	if kind() != "button-labels" {
		t.Fatalf("three R with the brightness row: %s", kind())
	}
}
