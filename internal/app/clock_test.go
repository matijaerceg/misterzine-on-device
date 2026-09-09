package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"testing"
	"time"
)

func TestClockRecoveryStampsVisitOnceAndKeepsBaseline(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	stored := &data.SeenRecord{T: "2026-09-01T00:00:00Z", Cur: map[string]string{"a": "old"}}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest([]data.Row{{K: "a", Updated: "new"}}, "", now), stored)
	baseline := a.seen.BaseTime
	a.SetClockTrusted(true)
	if a.seen.State.T != now.Format(time.RFC3339) || a.seen.BaseTime != baseline || !a.seen.Unseen(&a.ds.Rows[0]) {
		t.Fatal("clock recovery changed comparison or failed to stamp visit")
	}
	stamp := a.seen.State.T
	now = now.Add(time.Hour)
	a.SetClockTrusted(true)
	if a.seen.State.T != stamp {
		t.Fatal("repeated trust moved visit date")
	}
}

func TestNoticeUsesInputClockDespiteCalendarCorrection(t *testing.T) {
	timer := time.Unix(100, 0)
	calendar := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	for _, frame := range []bool{false, true} {
		a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return calendar }, TimerNow: func() time.Time { return timer }}, data.Ingest(nil, "", calendar), nil)
		a.Notice("test", time.Second)
		deadline := timer.Add(time.Second)
		if !a.NextTick().Equal(deadline) {
			t.Fatal("notice scheduled in calendar clock")
		}
		calendar = calendar.Add(50 * 365 * 24 * time.Hour)
		a.Tick(deadline.Add(-time.Millisecond))
		if a.notice == "" {
			t.Fatal("calendar jump expired notice")
		}
		if frame {
			a.Frame(deadline)
		} else {
			a.Tick(deadline)
		}
		if next := a.NextTick(); a.notice != "" || (!next.IsZero() && !next.After(deadline)) {
			t.Fatal("notice did not expire on input clock")
		}
	}
}
