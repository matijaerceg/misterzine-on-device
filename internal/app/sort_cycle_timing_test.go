package app

import (
	"os"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// TestSortCycleTiming reports how long each Y press (sort change plus the
// repaint) takes on the real catalogue; run with -v to see the numbers.
// MZ_DATA points at another data.json (the device run).
func TestSortCycleTiming(t *testing.T) {
	path := os.Getenv("MZ_DATA")
	if path == "" {
		path = "../../testdata/data.json"
	}
	f, err := os.Open(path)
	if err != nil {
		t.Skip(err)
	}
	rows, err := data.DecodeRows(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	ds := data.Ingest(rows, "", now)
	var recents []data.Recent
	for i := 0; i < 40 && i < len(rows); i++ {
		recents = append(recents, data.Recent{K: rows[i].K, At: now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339)})
	}
	a := New(Config{PhysW: 320, PhysH: 240, SafeInsetX: 15, SafeInsetY: 15, Now: func() time.Time { return now }, ClockTrusted: true,
		RecentLaunches: recents, Favorites: map[string]bool{rows[3].K: true, rows[10].K: true},
		Status: func(i int) data.Status { return data.Status(1 + i%4) }}, ds, nil)
	a.Paint()
	t.Logf("%d rows", len(rows))
	for round := 0; round < 3; round++ {
		for i := 0; i < 6; i++ {
			t0 := time.Now()
			a.actList(platform.KeySpace)
			t1 := time.Now()
			a.Paint()
			t2 := time.Now()
			t.Logf("round %d: -> %-12v sort+rebuild %6s  paint %6s  view %d", round, a.Sort(), t1.Sub(t0).Round(100*time.Microsecond), t2.Sub(t1).Round(100*time.Microsecond), len(a.view))
		}
	}
	// the same with a search and a filter active
	a.setSearch("ar")
	f2 := a.Filters()
	f2.Since = false
	for i := 0; i < 6; i++ {
		t0 := time.Now()
		a.actList(platform.KeySpace)
		t1 := time.Now()
		a.Paint()
		t.Logf("search: -> %-12v sort+rebuild %6s  paint %6s  view %d", a.Sort(), t1.Sub(t0).Round(100*time.Microsecond), time.Since(t1).Round(100*time.Microsecond), len(a.view))
	}
}
