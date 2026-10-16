package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// spaceApp has a few fighting games and a keyboard profile hook; typing goes
// in as a physical keyboard delivers it (letters KeyOther with text, Space
// KeySpace with text).
func spaceApp(t *testing.T) (*App, *time.Time) {
	t.Helper()
	clock := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "sf2", Title: "Street Fighter II: The World Warrior", Base: "Arcade", Genre: "Fighting", Updated: "2026-01-01", Date: "2020-01-01"},
		{K: "sf2ce", Title: "Street Fighter II': Champion Edition", Base: "Arcade", Genre: "Fighting", Updated: "2026-01-02", Date: "2020-01-02"},
		{K: "sfa", Title: "Street Smart", Base: "Arcade", Genre: "Fighting", Updated: "2026-01-03", Date: "2020-01-03"},
		{K: "zeta", Title: "Zeta Wing", Base: "Arcade", Genre: "Shoot 'em up", Updated: "2026-01-04", Date: "2020-01-04"},
	}
	a := New(Config{PhysW: 320, PhysH: 240, ShowBetaFeatures: true, AccessMonth: 202610, ClockTrusted: true, ViewsOff: []string{"recents"}, Now: func() time.Time { return clock }}, data.Ingest(rows, "h", clock), nil)
	a.cfg.ControlProfiles = controls.New()
	return a, &clock
}

func keyboardKey(code uint16, text rune) platform.Event {
	k := platform.KeyOther
	if code == 57 {
		k = platform.KeySpace
	}
	return platform.Event{Key: k, Code: code, Text: text, Keyboard: true, Source: "Logitech USB Receiver", Node: "event1", InstanceID: "046d_c548_50805e3d5466e4d6", DeviceID: "046d_c548"}
}

func typeWord(a *App, clock *time.Time, word string) {
	for _, ch := range word {
		code := uint16(30)
		if ch == ' ' {
			code = 57
		}
		ev := keyboardKey(code, ch)
		*clock = clock.Add(80 * time.Millisecond)
		ev.At, ev.Pressed = *clock, true
		a.Handle(ev)
		*clock = clock.Add(40 * time.Millisecond)
		ev.At, ev.Pressed = *clock, false
		a.Handle(ev)
	}
}

// A keyboard whose controls.json profile restates Space as Y (the Logitech
// receiver's Consumer Control profile, which its keyboard node shares by
// identity) still types a space into a search in progress; before a search
// Space stays the view key, and a mapped letter never types.
func TestProfiledKeyboardSpaceTypesIntoSearch(t *testing.T) {
	a, clock := spaceApp(t)
	a.cfg.ControlProfiles.Profiles["046d_c548_50805e3d5466e4d6"] = controls.Profile{MappingVersion: 2, Reviewed: true, Name: "Logitech USB Receiver Consumer Control",
		Inputs: map[uint16]string{1: "back", 28: "enter", 57: "space", 103: "up", 108: "down", 15: "tab"}}
	a.SetSort(data.SortAlphabetical)
	typeWord(a, clock, "street")
	if a.query != "street" || a.mode != data.SortAlphabetical {
		t.Fatalf("typed %q in %v", a.query, a.mode)
	}
	typeWord(a, clock, " ")
	if a.query != "street " || a.mode != data.SortAlphabetical {
		t.Fatalf("after Space: query %q, view %v", a.query, a.mode)
	}
	typeWord(a, clock, "fighter")
	if got := strings.Join(viewKeys(a), " "); a.query != "street fighter" || got != "sf2 sf2ce" {
		t.Fatalf("query %q lists %q", a.query, got)
	}
	// without the space the same titles match: searchText drops spaces
	a.setSearch("streetfighter")
	if got := strings.Join(viewKeys(a), " "); got != "sf2 sf2ce" {
		t.Fatalf("streetfighter lists %q", got)
	}
	// before a search Space is still the view key
	a.setSearch("")
	typeWord(a, clock, " ")
	if a.query != "" || a.mode != data.SortMaker {
		t.Fatalf("Space with no search: query %q, view %v", a.query, a.mode)
	}
	// a letter the profile maps to an action acts, and never types
	a.cfg.ControlProfiles.Profiles["046d_c548_50805e3d5466e4d6"] = controls.Profile{MappingVersion: 2, Inputs: map[uint16]string{30: "space", 57: "space"}}
	a.SetSort(data.SortAlphabetical)
	typeWord(a, clock, "a")
	if a.query != "" || a.mode != data.SortMaker {
		t.Fatalf("mapped letter: query %q, view %v", a.query, a.mode)
	}
	// an arcade panel's Space is a button, never text
	a.cfg.ControlProfiles.Profiles["046d_c548_50805e3d5466e4d6"] = controls.Profile{MappingVersion: 2, Panel: true, Inputs: map[uint16]string{57: "space"}}
	a.SetSort(data.SortAlphabetical)
	a.setSearch("street")
	typeWord(a, clock, " ")
	if a.query != "street" || a.mode != data.SortMaker {
		t.Fatalf("panel Space: query %q, view %v", a.query, a.mode)
	}
}

// Main types every keyboard press again on its virtual input, where Space
// arrives as a textless Space: during a search that echo must not change
// the view, whether it lands while the key is held or just after a quick
// tap, while a pad's Y through the same node keeps working.
func TestVirtualInputSpaceEchoDoesNotChangeTheViewWhileSearching(t *testing.T) {
	a, clock := spaceApp(t)
	a.SetSort(data.SortAlphabetical)
	echo := func(pressed bool, at time.Time) {
		a.Handle(platform.Event{Key: platform.KeySpace, Code: 57, Pressed: pressed, At: at, Source: "MiSTer virtual input"})
	}
	kbd := func(pressed bool, at time.Time) {
		ev := platform.Event{Key: platform.KeySpace, Code: 57, Text: ' ', Pressed: pressed, At: at, Source: "Logitech USB Receiver", Keyboard: true}
		a.Handle(ev)
	}
	typeWord(a, clock, "street")
	// the echo lands while the key is down
	kbd(true, clock.Add(10*time.Millisecond))
	echo(true, clock.Add(12*time.Millisecond))
	kbd(false, clock.Add(50*time.Millisecond))
	echo(false, clock.Add(52*time.Millisecond))
	if a.query != "street " || a.mode != data.SortAlphabetical {
		t.Fatalf("held echo: query %q, view %v", a.query, a.mode)
	}
	// a quick tap: the echo lands after the release
	*clock = clock.Add(time.Second)
	kbd(true, *clock)
	kbd(false, clock.Add(20*time.Millisecond))
	echo(true, clock.Add(25*time.Millisecond))
	echo(false, clock.Add(45*time.Millisecond))
	if a.query != "street  " || a.mode != data.SortAlphabetical {
		t.Fatalf("late echo: query %q, view %v", a.query, a.mode)
	}
	// a pad's Y through Main, well after any typing, still changes the view
	*clock = clock.Add(time.Second)
	echo(true, *clock)
	echo(false, clock.Add(40*time.Millisecond))
	if a.query != "street  " || a.mode != data.SortMaker {
		t.Fatalf("pad Y: query %q, view %v", a.query, a.mode)
	}
	// with no search the keyboard's Space and its echo change the view once
	a.setSearch("")
	a.SetSort(data.SortAlphabetical)
	*clock = clock.Add(time.Second)
	kbd(true, *clock)
	echo(true, clock.Add(2*time.Millisecond))
	kbd(false, clock.Add(40*time.Millisecond))
	echo(false, clock.Add(42*time.Millisecond))
	if a.mode != data.SortMaker {
		t.Fatalf("Space with no search moved to %v", a.mode)
	}
}
