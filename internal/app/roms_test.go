package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"testing"
	"time"
)

func TestMissingROMBlocksLaunch(t *testing.T) {
	launched, fresh := false, false
	a := New(Config{Launch: func(string) { launched = true }, Exists: func(string) bool { return true }, ROMIssue: func(_ string, f bool) (string, bool) { fresh = f; return "Missing game ROM: jpark.zip", true }}, data.Ingest(nil, "test", time.Now()), nil)
	row := data.Row{MRA: "_Arcade/Jurassic Park.mra"}
	if !a.launchRow(&row, 0, 0) || launched || !fresh || a.notice != "Missing game ROM: jpark.zip" {
		t.Fatalf("missing ROM not blocked: launched=%v fresh=%v notice=%q", launched, fresh, a.notice)
	}
}

// A warning shows in Details but Start still hands the game to MiSTer.
func TestROMWarningStillLaunches(t *testing.T) {
	var launched string
	warn := "Wrong ROM version: bagman.zip (p3.bin)"
	a := New(Config{Launch: func(p string) { launched = p }, Exists: func(string) bool { return true },
		ROMIssue: func(string, bool) (string, bool) { return warn, false }}, data.Ingest(nil, "test", time.Now()), nil)
	row := data.Row{MRA: "_Arcade/Bagman.mra"}
	found := false
	for _, line := range a.detailLines(&row, &data.Derived{}, 0) {
		found = found || line.text == warn
	}
	if !found {
		t.Fatal("Details did not show the warning")
	}
	a.launchRow(&row, 0, 0)
	if launched != "_Arcade/Bagman.mra" || a.notice == warn {
		t.Fatalf("warning blocked the launch: launched=%q notice=%q", launched, a.notice)
	}
}

func TestROMWarningFollowsSelectedVersion(t *testing.T) {
	var launched string
	a := New(Config{Launch: func(p string) { launched = p }, Exists: func(string) bool { return true },
		Alternatives: func(*data.Row) []string { return []string{"alt.mra"} },
		ROMIssue: func(p string, _ bool) (string, bool) {
			if p == "alt.mra" {
				return "Missing game ROM: alternate.zip", true
			}
			return "", false
		},
	}, data.Ingest(nil, "test", time.Now()), nil)
	row := data.Row{MRA: "main.mra"}
	a.detail.pick = 1
	lines := a.detailLines(&row, &data.Derived{}, 0)
	found := false
	for _, line := range lines {
		if line.text == "Missing game ROM: alternate.zip" {
			found = true
		}
	}
	if !found {
		t.Fatal("Details did not warn about selected alternative")
	}
	a.launchRow(&row, 0, 1)
	if launched != "" {
		t.Fatal("missing alternative launched")
	}
	a.launchRow(&row, 0, 0)
	if launched != "main.mra" {
		t.Fatal("present main version was blocked")
	}
}
