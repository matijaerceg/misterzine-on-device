package app

import (
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The background ROM check marks the list by the version Start launches,
// counts the other versions' problems for Details, and feeds the ROM
// filter; a remembered version moves the mark with it.
func TestROMStateMarkAndFilter(t *testing.T) {
	answers := map[string]string{
		"_Arcade/Good.mra":                     "",
		"_Arcade/Bad.mra":                      "Missing game ROM: bad.zip",
		"_Arcade/_alternatives/Good (alt).mra": "Incomplete ROM: goodalt.zip",
		"_Arcade/Mixed.mra":                    "",
	}
	rows := []data.Row{
		{K: "good", Title: "Good", Base: "Arcade", MRA: "_Arcade/Good.mra", Core: "good"},
		{K: "bad", Title: "Bad", Base: "Arcade", MRA: "_Arcade/Bad.mra", Core: "bad"},
		{K: "mixed", Title: "Mixed", Base: "Arcade", MRA: "_Arcade/Mixed.mra", Core: "mixed"},
		{K: "off", Title: "Off card", Base: "Arcade", MRA: "_Arcade/Off.mra", Core: "off"},
		{K: "cons", Title: "Console", Base: "Console", Core: "cons"},
	}
	st := map[string]data.Status{"good": data.StatusCurrent, "bad": data.StatusCurrent, "mixed": data.StatusOutdated, "off": data.StatusNotFound, "cons": data.StatusCurrent}
	versions := map[string]string{}
	clock := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	ds := data.Ingest(rows, "test", clock)
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock }, TimerNow: func() time.Time { return clock },
		Versions: versions,
		Status:   func(i int) data.Status { return st[ds.Rows[i].K] },
		Exists:   func(string) bool { return true },
		ROMKnown: func(rel string) (string, bool, bool) {
			text, ok := answers[rel]
			return text, text != "", ok
		},
		Alternatives: func(r *data.Row) []string {
			switch r.K {
			case "good":
				return []string{"_Arcade/_alternatives/Good (alt).mra"}
			case "mixed":
				return []string{"_Arcade/_alternatives/Mixed (unchecked).mra"}
			}
			return nil
		},
	}, ds, nil)
	idx := func(k string) int {
		for i := range ds.Rows {
			if ds.Rows[i].K == k {
				return i
			}
		}
		t.Fatalf("no row %s", k)
		return -1
	}
	want := map[string]data.ROMState{"good": data.ROMOtherIssue, "bad": data.ROMLaunchIssue, "mixed": data.ROMUnknown, "off": data.ROMUnknown, "cons": data.ROMUnknown}
	for k, w := range want {
		if got := a.romState(idx(k)); got != w {
			t.Errorf("%s: state %d, want %d", k, got, w)
		}
	}
	if m, _, ok := a.romMark(idx("bad")); !ok || m != "!" {
		t.Errorf("bad: mark %q %v", m, ok)
	}
	if _, _, ok := a.romMark(idx("good")); ok {
		t.Error("good: marked although only an alternative has a problem")
	}
	if bad, total := a.romVersions(&ds.Rows[idx("good")], a.launchEntries(&ds.Rows[idx("good")], idx("good"))); bad != 1 || total != 2 {
		t.Errorf("good: %d of %d versions bad", bad, total)
	}

	// the filters
	keys := func() []string {
		var out []string
		for _, i := range a.view {
			out = append(out, ds.Rows[i].K)
		}
		return out
	}
	a.filters.ROM = data.ROMLaunch
	a.Refilter()
	if got := keys(); len(got) != 1 || got[0] != "bad" {
		t.Errorf("launch filter: %v", got)
	}
	a.filters.ROM = data.ROMAny
	a.Refilter()
	if got := keys(); len(got) != 2 {
		t.Errorf("any filter: %v", got)
	}
	a.filters.ROM = data.ROMNone
	a.Refilter()
	if got := keys(); len(got) != 0 {
		t.Errorf("none filter: %v (good has a bad alternative, mixed is unchecked)", got)
	}
	a.filters.ROM = data.ROMAll

	// choosing the bad alternative moves the mark onto Good
	versions["good"] = "_Arcade/_alternatives/Good (alt).mra"
	a.ROMChanged()
	if got := a.romState(idx("good")); got != data.ROMLaunchIssue {
		t.Errorf("good with the bad alternative chosen: state %d", got)
	}
	// and a new answer for Mixed's alternative settles it
	answers["_Arcade/_alternatives/Mixed (unchecked).mra"] = ""
	a.ROMChanged()
	if got := a.romState(idx("mixed")); got != data.ROMClean {
		t.Errorf("mixed once every version answered: state %d", got)
	}
	// without the hook nothing is marked
	b := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock }, TimerNow: func() time.Time { return clock }}, ds, nil)
	if got := b.romState(0); got != data.ROMUnknown {
		t.Errorf("no hook: state %d", got)
	}
}
