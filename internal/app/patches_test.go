package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

const patchedPath = "_Arcade/_Arcade Patches/_Restorations/Night Warriors 2 (USA) [English Translation].mra"

func patchRows() []data.Row {
	return []data.Row{
		{K: "vhunt2", Title: "Vampire Hunter 2", Base: "Arcade", Core: "jtcps2", SN: "vhunt2", Genre: "Fighter",
			FamilySets: []string{"vhunt2", "nwarr2en"}, MRA: "_Arcade/Vampire Hunter 2 (Japan).mra",
			PT: []data.Patch{
				{T: "translation", N: "Night Warriors 2", S: "English translation", V: "rc3", D: "2026-10-11", U: "https://arcadepatches.com/x/", SN: "nwarr2en"},
				{T: "arrange", N: "Vampire Hunter 2 Arrange", V: "rc1", D: "2026-09-01", SN: "vhunt2ar"},
			}},
		{K: "sfa2", Title: "Street Fighter Alpha 2", Base: "Arcade", Core: "jtcps2", SN: "sfa2", Genre: "Fighter",
			PT: []data.Patch{{T: "enhanced", N: "SFA2 Gold", SN: ""}}},
		{K: "pacman", Title: "Pac-Man", Base: "Arcade", Core: "pacman", SN: "puckman", Genre: "Maze"},
	}
}

func TestPatchDetailLines(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240,
		Alternatives: func(r *data.Row) []string {
			if r.K == "vhunt2" {
				return []string{patchedPath}
			}
			return nil
		},
		AltSetname: func(p string) string {
			if p == patchedPath {
				return "nwarr2en"
			}
			return ""
		},
	}, data.Ingest(patchRows(), "", time.Now()), nil)
	text := func(i int) string {
		var b []string
		for _, l := range a.detailLines(&a.ds.Rows[i], &a.ds.Der[i], i) {
			b = append(b, l.text)
		}
		return strings.Join(b, "\n")
	}
	got := text(0)
	for _, want := range []string{
		"Patches: Arcade Patches (strygo)",
		"Translation: Night Warriors 2 rc3 (2026-10-11), on card",
		"Arranged soundtrack: Vampire Hunter 2 Arrange rc1 (2026-09-01), not on card, arcadepatches.com",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("vhunt2 details lack %q:\n%s", want, got)
		}
	}
	if g := text(1); !strings.Contains(g, "Enhanced: SFA2 Gold, not on card, arcadepatches.com") {
		t.Errorf("sfa2 details:\n%s", g)
	}
	if g := text(2); strings.Contains(g, "Patches") {
		t.Errorf("unpatched game shows a Patches group:\n%s", g)
	}
	for _, r := range got {
		if r > 126 {
			t.Fatalf("non-ASCII in details: %q", r)
		}
	}
}

func TestPatchFilterSection(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(patchRows(), "", time.Now()), nil)
	var labels []string
	in := false
	for _, e := range a.rawFilterEntries() {
		if e.header {
			in = e.kind == "patch"
			continue
		}
		if in {
			labels = append(labels, e.text+"="+itoa(e.count))
		}
	}
	want := "Translation=1,Enhanced=1,Arranged soundtrack=1,None=1"
	if strings.Join(labels, ",") != want {
		t.Fatalf("Patches section %v, want %s", labels, want)
	}
	// unchecking None leaves the patched games
	a.SetFilters(data.Filters{PatchOff: map[string]bool{"": true}})
	if n := len(a.view); n != 2 {
		t.Fatalf("None off: %d rows, want 2", n)
	}
	// a game with two kinds stays while one of them is checked
	a.SetFilters(data.Filters{PatchOff: map[string]bool{"translation": true, "": true, "enhanced": true}})
	if n := len(a.view); n != 1 || a.ds.Rows[a.view[0]].K != "vhunt2" {
		t.Fatalf("arrange only: %v", a.view)
	}

	// a catalogue without any patch shows no Patches section
	b := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(patchRows()[2:], "", time.Now()), nil)
	for _, e := range b.rawFilterEntries() {
		if e.kind == "patch" {
			t.Fatal("Patches section without patches")
		}
	}
}
