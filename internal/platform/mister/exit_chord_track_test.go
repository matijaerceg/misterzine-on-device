package mister

import (
	"testing"
	"time"
)

func TestChordHoldFiresOncePerHold(t *testing.T) {
	c := chordHold{hold: time.Second}
	t0 := time.Unix(1000, 0)
	steps := []struct {
		at   time.Duration
		down bool
		want bool
	}{
		{0, true, false},
		{500 * time.Millisecond, true, false},
		{900 * time.Millisecond, false, false}, // let go before the second: nothing
		{1 * time.Second, true, false},         // a new hold starts here
		{1900 * time.Millisecond, true, false},
		{2 * time.Second, true, true}, // held a full second: fires
		{2100 * time.Millisecond, true, false},
		{9 * time.Second, true, false}, // still held: never twice
		{9100 * time.Millisecond, false, false},
		{9200 * time.Millisecond, true, false},
		{10200 * time.Millisecond, true, true}, // released and held again: fires again
	}
	for i, s := range steps {
		if got := c.update(s.down, t0.Add(s.at)); got != s.want {
			t.Fatalf("step %d at %v down=%v: fired %v, want %v", i, s.at, s.down, got, s.want)
		}
	}
}

func TestExitChordSlots(t *testing.T) {
	if got := ExitChordSlots(ExitChordOff); got != nil {
		t.Fatalf("off has slots %v", got)
	}
	if got := ExitChordSlots("nonsense"); got != nil {
		t.Fatalf("an unknown variant has slots %v", got)
	}
	if got := ExitChordSlots(ExitChordSelectStart); len(got) != 2 || got[0] != "Select" || got[1] != "Start" {
		t.Fatalf("Select+Start slots %v", got)
	}
	if got := ExitChordSlots(ExitChordShouldersSelect); len(got) != 4 || got[0] != "L" || got[3] != "Start" {
		t.Fatalf("L+R+Select+Start slots %v", got)
	}
	if ExitChordName(ExitChordOff) != "off" || ExitChordName(ExitChordSelectStart) != "Select+Start" || ExitChordName(ExitChordShouldersSelect) != "L+R+Select+Start" {
		t.Fatal("variant names")
	}
}
