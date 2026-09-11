package data

import "testing"

func TestLikelyOutdatedCountsAsOlder(t *testing.T) {
	if !StatusLikelyOutdated.Found() || !StatusLikelyOutdated.Older() {
		t.Fatal("likely outdated must count as found and older")
	}
	if StatusFoundUndated.Older() || StatusCurrent.Older() {
		t.Fatal("undated/current are not older")
	}
	// the harness cycles statuses 1..4; the new value must sit after them
	if StatusLikelyOutdated <= StatusNotFound {
		t.Fatalf("StatusLikelyOutdated = %d, must follow StatusNotFound", StatusLikelyOutdated)
	}
	row := Row{Base: "Arcade", Title: "x"}
	d := &Derived{}
	older := &Filters{Install: InstallOlder}
	for _, st := range []Status{StatusOutdated, StatusLikelyOutdated} {
		if !older.Pass(&row, d, st, false, false) {
			t.Errorf("older filter must include %v", st)
		}
	}
	undated := &Filters{Install: InstallUndated}
	if undated.Pass(&row, d, StatusLikelyOutdated, false, false) {
		t.Error("undated filter must exclude likely outdated")
	}
}
