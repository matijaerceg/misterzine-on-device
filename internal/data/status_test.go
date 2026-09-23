package data

import (
	"strings"
	"testing"
)

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
		if !older.Pass(&row, d, st, false, false, ROMUnknown) {
			t.Errorf("older filter must include %v", st)
		}
	}
	undated := &Filters{Install: InstallUndated}
	if undated.Pass(&row, d, StatusLikelyOutdated, false, false, ROMUnknown) {
		t.Error("undated filter must exclude likely outdated")
	}
}

// Why names the rule that hides a row, in the words the diagnostic report
// prints, and Pass agrees with it whatever the rule.
func TestFiltersWhy(t *testing.T) {
	r := Row{Base: "Arcade", Src: "local", Rot: "Vertical (CW)", Year: "1989", Plr: "2", Genre: "Shooter", Res: "15kHz", Deprecated: true}
	d := Derived{RotGroup: "v", Year: "1989", Directions: "8-way", Buttons: "2"}
	for _, c := range []struct {
		f    Filters
		st   Status
		fav  bool
		want string
	}{
		{Filters{}, StatusCurrent, false, ""},
		{Filters{MatchRotation: "h"}, StatusCurrent, false, "Filter by rotation (the screen shows horizontal games; this one is vertical)"},
		{Filters{HideDeprecated: true}, StatusCurrent, false, "deprecated"},
		{Filters{SrcOff: map[string]bool{"local": true}}, StatusCurrent, false, "Filters -> Source (local off)"},
		{Filters{SrcHidden: map[string]bool{"local": true}}, StatusCurrent, false, "installed only"},
		{Filters{BetaOff: map[string]bool{"stable": true}}, StatusCurrent, false, "Filters -> Type (stable off)"},
		{Filters{YearOff: map[string]bool{"1989": true}}, StatusCurrent, false, "Filters -> Year (1989 off)"},
		{Filters{RotOff: map[string]bool{"v": true}}, StatusCurrent, false, "Filters -> Rotation (vertical off)"},
		{Filters{PlrOff: map[string]bool{"2": true}}, StatusCurrent, false, "Filters -> Players"},
		{Filters{GenreOff: map[string]bool{"Shooter": true}}, StatusCurrent, false, "Filters -> Genre"},
		{Filters{Install: InstallMissing}, StatusCurrent, false, "only games not on the card"},
		{Filters{FavOnly: true}, StatusCurrent, false, "favorites only"},
		{Filters{FavOnly: true}, StatusCurrent, true, ""},
	} {
		got := c.f.Why(&r, &d, c.st, c.fav, false, ROMUnknown)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%+v: Why %q, want %q", c.f, got, c.want)
		}
		if c.f.Pass(&r, &d, c.st, c.fav, false, ROMUnknown) != (got == "") {
			t.Errorf("%+v: Pass disagrees with Why %q", c.f, got)
		}
	}
	var none *Filters
	if none.Why(&r, &d, StatusNotFound, false, false, ROMUnknown) != "" || !none.Pass(&r, &d, StatusNotFound, false, false, ROMUnknown) {
		t.Fatal("nil filters hide nothing")
	}
	unknown := Derived{}
	if got := (&Filters{MatchRotation: "h"}).Why(&r, &unknown, StatusCurrent, false, false, ROMUnknown); !strings.Contains(got, "unknown rotation") {
		t.Fatalf("a row without a rotation: %q", got)
	}
}
