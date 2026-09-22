package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func versionsFixture(t *testing.T, versions map[string]string, alts []string) (*App, func(platform.Key), *string, *int) {
	t.Helper()
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "game", Title: "Example Game", Base: "Arcade", Core: "Example", MRA: "_Arcade/Example.mra", Updated: "2026-09-08", SN: "example"},
		{K: "other", Title: "Other Game", Base: "Arcade", Core: "Other", MRA: "_Arcade/Other.mra", Updated: "2026-09-07", SN: "other"},
	}
	launched, changed := "", 0
	a := New(Config{
		PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15,
		Now: func() time.Time { return now }, ClockTrusted: true,
		Status: func(int) data.Status { return data.StatusCurrent },
		Exists: func(string) bool { return true },
		Alternatives: func(r *data.Row) []string {
			if r.K == "game" {
				return alts
			}
			return nil
		},
		Versions:       versions,
		VersionChanged: func() { changed++ },
		Launch:         func(path string) { launched = path },
	}, data.Ingest(rows, "test", now), nil)
	tap := func(key platform.Key) {
		a.Handle(platform.Event{Key: key, Pressed: true, At: now})
		a.Handle(platform.Event{Key: key, At: now.Add(time.Millisecond)})
		a.Paint()
		now = now.Add(50 * time.Millisecond) // past the bounce guard
	}
	return a, tap, &launched, &changed
}

// Choosing a version in Details is remembered per game: Details reopens on
// it, Start from the list launches it, and the main version clears the record.
func TestChosenVersionIsRememberedPerGame(t *testing.T) {
	alts := []string{"_Arcade/_alternatives/Example A.mra", "_Arcade/_alternatives/Example B.mra"}
	a, tap, launched, changed := versionsFixture(t, nil, alts)
	tap(platform.KeyEnter)
	tap(platform.KeyRight)
	tap(platform.KeyRight)
	tap(platform.KeyRight) // past the end stays on the last entry
	if a.detail.pick != 2 || *changed != 2 || a.Versions()["game"] != alts[1] {
		t.Fatalf("after choosing the second alternative: pick=%d changed=%d versions=%v", a.detail.pick, *changed, a.Versions())
	}
	tap(platform.KeyBack)
	tap(platform.KeyEnter)
	if a.detail.pick != 2 {
		t.Fatalf("Details reopened on entry %d, not the remembered alternative", a.detail.pick)
	}
	tap(platform.KeyBack)
	tap(platform.KeyStart)
	if *launched != alts[1] {
		t.Fatalf("Start from the list launched %q, not the remembered %q", *launched, alts[1])
	}
	// another game is untouched and starts on its main version
	tap(platform.KeyDown)
	tap(platform.KeyEnter)
	if a.detail.pick != 0 || len(a.Versions()) != 1 {
		t.Fatalf("the other game: pick=%d versions=%v", a.detail.pick, a.Versions())
	}
	tap(platform.KeyBack)
	tap(platform.KeyUp)
	// back to the main version forgets the choice
	tap(platform.KeyEnter)
	tap(platform.KeyLeft)
	tap(platform.KeyLeft)
	tap(platform.KeyLeft)
	if a.detail.pick != 0 || len(a.Versions()) != 0 || *changed != 4 {
		t.Fatalf("after returning to the main version: pick=%d versions=%v changed=%d", a.detail.pick, a.Versions(), *changed)
	}
	tap(platform.KeyBack)
	*launched = ""
	tap(platform.KeyStart)
	if *launched != "_Arcade/Example.mra" {
		t.Fatalf("Start launched %q instead of the main version", *launched)
	}
}

// A remembered version from a previous run opens and launches; one whose
// file has left the card falls back to the main version. Launching from
// Details records the launched entry too.
func TestRememberedVersionRestoresAndFallsBack(t *testing.T) {
	alts := []string{"_Arcade/_alternatives/Example A.mra", "_Arcade/_alternatives/Example B.mra"}
	a, tap, launched, changed := versionsFixture(t, map[string]string{"game": alts[0]}, alts)
	tap(platform.KeyStart)
	if *launched != alts[0] || *changed != 0 {
		t.Fatalf("Start launched %q (changed %d); wanted the stored %q with no change", *launched, *changed, alts[0])
	}
	tap(platform.KeyEnter)
	if a.detail.pick != 1 {
		t.Fatalf("Details opened on entry %d, not the stored alternative", a.detail.pick)
	}
	tap(platform.KeyRight)
	tap(platform.KeyStart)
	if *launched != alts[1] || a.Versions()["game"] != alts[1] {
		t.Fatalf("Details launch: launched %q, remembered %q", *launched, a.Versions()["game"])
	}

	gone, tap2, launched2, _ := versionsFixture(t, map[string]string{"game": "_Arcade/_alternatives/Removed.mra"}, alts)
	tap2(platform.KeyEnter)
	if gone.detail.pick != 0 {
		t.Fatalf("a removed alternative opened Details on entry %d", gone.detail.pick)
	}
	tap2(platform.KeyBack)
	tap2(platform.KeyStart)
	if *launched2 != "_Arcade/Example.mra" {
		t.Fatalf("a removed alternative launched %q instead of the main version", *launched2)
	}
	if gone.Versions()["game"] != "_Arcade/_alternatives/Removed.mra" { // launching the main version keeps it: the alternatives may not be scanned yet
		t.Fatal("the stale record was dropped before the user chose again")
	}
}

// A rescan can put a new version ahead of the chosen one: the choice
// follows its file, Start launches that file, and a chosen file that has
// gone falls back to the main version rather than onto its neighbour.
// Versions on another core say which, and two files of one name say where.
func TestChosenVersionFollowsItsFile(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{{K: "pleiads", Title: "Pleiads", Base: "Arcade", Core: "phoenix", MRA: "_Arcade/Pleiads (Tehkan).mra", SN: "pleiads", Updated: "2026-09-08"}}
	centuri, irecsa := "_Arcade/_alternatives/_Pleiads/Pleiads (Centuri).mra", "_Arcade/_alternatives/_Pleiads/Pleiads (Irecsa).mra"
	alts := []string{irecsa}
	launched := ""
	a := New(Config{
		PhysW: 320, PhysH: 240,
		Now: func() time.Time { return now }, ClockTrusted: true,
		Status:       func(int) data.Status { return data.StatusCurrent },
		Exists:       func(string) bool { return true },
		Alternatives: func(*data.Row) []string { return alts },
		AltCore: func(p string) string {
			if p == centuri || p == irecsa {
				return "pleiads"
			}
			return ""
		},
		Versions: map[string]string{},
		Launch:   func(p string) { launched = p },
	}, data.Ingest(rows, "test", now), nil)
	tap := func(key platform.Key) {
		a.Handle(platform.Event{Key: key, Pressed: true, At: now})
		a.Handle(platform.Event{Key: key, At: now.Add(time.Millisecond)})
		a.Paint()
		now = now.Add(50 * time.Millisecond)
	}
	tap(platform.KeyEnter)
	tap(platform.KeyRight) // choose Irecsa, the only alternative
	if e := a.launchEntries(&a.ds.Rows[0], 0); a.detail.pickPath != irecsa || !strings.HasSuffix(e[1].label, "[pleiads]") || strings.Contains(e[0].label, "[") {
		t.Fatalf("pick %q, labels %q / %q", a.detail.pickPath, e[0].label, e[1].label)
	}
	alts = []string{centuri, irecsa} // a rescan found Centuri, which sorts first
	a.Invalidate()                   // as the host does when a scan result arrives
	a.Paint()
	if a.detail.pick != 2 {
		t.Fatalf("the choice moved to entry %d instead of following Irecsa", a.detail.pick)
	}
	tap(platform.KeyStart)
	if launched != irecsa {
		t.Fatalf("Start launched %q, not the chosen %q", launched, irecsa)
	}
	alts = []string{centuri} // Irecsa's core left the card
	a.screen = ScreenDetails
	a.Invalidate()
	a.Paint()
	if a.detail.pick != 0 {
		t.Fatalf("a chosen file that has gone must fall back to the main version, not entry %d", a.detail.pick)
	}

	// two files of one name in different folders are told apart
	alts = []string{"_Arcade/_Extra/Pleiads (Tehkan).mra"}
	e := a.launchEntries(&a.ds.Rows[0], 0)
	if !strings.Contains(e[0].label, "_Arcade/Pleiads (Tehkan).mra") || !strings.Contains(e[1].label, "_Extra/Pleiads (Tehkan).mra") {
		t.Fatalf("labels %q / %q", e[0].label, e[1].label)
	}
}
