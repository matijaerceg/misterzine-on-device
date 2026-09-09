package data

import (
	"testing"
	"time"
)

func TestSeenColdBootUsesPreviousSnapshot(t *testing.T) {
	old := SeenRecord{T: "2026-09-01T00:00:00Z", Cur: map[string]string{"a": "old"}}
	rows := []Row{{K: "a", Updated: "new"}}
	s := InitSeen(&old, rows, time.Time{}, false)
	if !s.Unseen(&rows[0]) || s.BaseTime != old.T || s.State.T != old.T {
		t.Fatalf("lost previous visit: %+v", s)
	}
	// Reopening without a clock must hold that baseline, not bank away the change.
	again := InitSeen(&s.State, rows, time.Time{}, false)
	if !again.Unseen(&rows[0]) {
		t.Fatal("cold return advanced baseline")
	}
	first := InitSeen(nil, rows, time.Time{}, false)
	if first.BaseRows != nil || first.State.T != "" {
		t.Fatal("invented first visit baseline/date")
	}
}
