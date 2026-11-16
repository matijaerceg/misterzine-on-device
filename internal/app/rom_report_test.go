package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// Options -> Data -> ROM report lands on the list of every game with a ROM
// problem: other filters, the search and a narrowing view are put aside.
// Until the check has finished, or when the list could show nothing, it
// leaves everything as it was and says why.
func TestROMReportShowsProblemGames(t *testing.T) {
	answers := map[string]string{
		"_Arcade/Good.mra":                     "",
		"_Arcade/Bad.mra":                      "Missing game ROM: bad.zip",
		"_Arcade/_alternatives/Good (alt).mra": "Incomplete ROM: goodalt.zip",
		"_Arcade/Clean.mra":                    "",
		"_Arcade/Old.mra":                      "Missing game ROM: old.zip",
	}
	rows := []data.Row{
		{K: "good", Title: "Good", Base: "Arcade", MRA: "_Arcade/Good.mra", Core: "good"},
		{K: "bad", Title: "Bad", Base: "Arcade", MRA: "_Arcade/Bad.mra", Core: "bad"},
		{K: "clean", Title: "Clean", Base: "Arcade", MRA: "_Arcade/Clean.mra", Core: "clean"},
		{K: "old", Title: "Old", Base: "Arcade", MRA: "_Arcade/Old.mra", Core: "old", Deprecated: true},
	}
	clock := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	done, total := 1, 4
	filtersSaved, settingsSaved := 0, 0
	newApp := func(answers map[string]string) *App {
		ds := data.Ingest(rows, "test", clock)
		return New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock }, TimerNow: func() time.Time { return clock },
			Status: func(int) data.Status { return data.StatusCurrent },
			Exists: func(string) bool { return true },
			ROMKnown: func(rel string) (string, bool, bool) {
				text, ok := answers[rel]
				return text, text != "", ok
			},
			Alternatives: func(r *data.Row) []string {
				if r.K == "good" {
					return []string{"_Arcade/_alternatives/Good (alt).mra"}
				}
				return nil
			},
			ROMProgress:     func() (int, int) { return done, total },
			ROMList:         "misterzine/rom_problems.txt",
			FiltersChanged:  func() { filtersSaved++ },
			SettingsChanged: func() { settingsSaved++ },
		}, ds, nil)
	}
	press := func(a *App) {
		t.Helper()
		a.openOptions()
		a.panel.entries = a.optionsEntries()
		for i, e := range a.panel.entries {
			if e.kind == "rom-report" {
				// tate at the widest inset: four help lines of 30, the path whole on the last
				if lines := gfx.Wrap(e.help, 30, 5); len(lines) > 4 || lines[len(lines)-1] != "misterzine/rom_problems.txt" {
					t.Fatalf("help does not keep the file in tate: %q", lines)
				}
				if got := optionsActs[e.kind]; got != "Show" {
					t.Fatalf("hint %q", got)
				}
				a.panel.cursor = i
				a.togglePanel()
				return
			}
		}
		t.Fatal("no ROM report row")
	}
	narrowed := func(a *App) {
		a.filters = data.Filters{GenreOff: map[string]bool{"": true}}
		a.query = "go"
		a.mode = data.SortFavorites
		a.rebuild()
	}

	a := newApp(answers)
	narrowed(a)
	press(a)
	if a.screen != ScreenOptions || a.filters.ROMActive() || a.query != "go" || a.mode != data.SortFavorites {
		t.Fatal("a check still running changed the list")
	}
	if want := "ROM check: 1 of 4 files" + gfx.Ellipsis; a.notice != want {
		t.Fatalf("while checking: %q", a.notice)
	}

	done = 4
	filtersSaved, settingsSaved = 0, 0
	press(a)
	if a.screen != ScreenList {
		t.Fatalf("screen %v, want the list", a.screen)
	}
	if a.query != "" || viewNarrows(a.mode) || a.filters.ROM != data.ROMAny || len(a.filters.GenreOff) != 0 {
		t.Fatalf("not cleared: query %q, mode %v, filters %+v", a.query, a.mode, a.filters)
	}
	var keys []string
	for pos := range a.view {
		keys = append(keys, a.keyAt(pos))
	}
	// the deprecated game stays hidden by Options, as in the list
	if got := strings.Join(keys, ","); got != "bad,good" && got != "good,bad" {
		t.Fatalf("listed %q", got)
	}
	if a.notice != "2 games with ROM problems" {
		t.Fatalf("notice %q", a.notice)
	}
	if filtersSaved != 1 || settingsSaved != 1 {
		t.Fatalf("saved filters %d, settings %d times", filtersSaved, settingsSaved)
	}

	// nothing to show: everything stays, the notice tells which case it is
	for _, c := range []struct {
		answers map[string]string
		want    string
	}{
		{map[string]string{"_Arcade/Good.mra": "", "_Arcade/_alternatives/Good (alt).mra": "", "_Arcade/Bad.mra": "", "_Arcade/Clean.mra": "", "_Arcade/Old.mra": ""}, "No ROM problems found"},
		{map[string]string{"_Arcade/Good.mra": "", "_Arcade/_alternatives/Good (alt).mra": "", "_Arcade/Bad.mra": "", "_Arcade/Clean.mra": "", "_Arcade/Old.mra": "Missing game ROM: old.zip"}, "Problems only in hidden games"},
	} {
		a := newApp(c.answers)
		narrowed(a)
		press(a)
		if a.screen != ScreenOptions || a.filters.ROMActive() || a.query != "go" || a.mode != data.SortFavorites {
			t.Fatalf("%s: the list changed", c.want)
		}
		if a.notice != c.want {
			t.Fatalf("notice %q, want %q", a.notice, c.want)
		}
	}

	done, total = 0, 0
	a = newApp(answers)
	press(a)
	if a.screen != ScreenOptions || a.notice != "ROM check runs after the scan" {
		t.Fatalf("before the check: screen %v, notice %q", a.screen, a.notice)
	}
}
