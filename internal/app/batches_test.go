package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// batchApp has three games one core build shipped together and a game
// alone, with the beta shown.
func batchApp(beta bool) *App {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "alpha", Title: "Alpha", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-06"},
		{K: "bravo", Title: "Bravo", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-06"},
		{K: "charlie", Title: "Charlie", Base: "Arcade", Core: "gnet", Updated: "2026-10-15", Date: "2026-10-01"},
		{K: "solo", Title: "Solo", Base: "Arcade", Core: "solo", Updated: "2026-10-01", Date: "2025-06-01"},
	}
	return New(Config{PhysW: 320, PhysH: 240, ClockTrusted: true, ShowBetaFeatures: beta, Now: func() time.Time { return now }},
		data.Ingest(rows, "h", now), nil)
}

func TestBatchFoldsUnderBuildDate(t *testing.T) {
	a := batchApp(true)
	a.SetSort(data.SortUpdated)
	if got := strings.Join(viewKeys(a), " "); got != "b:gnet@2026-10-15 solo" {
		t.Fatalf("folded view %q", got)
	}
	a.cursor = 0
	if got := a.umbrellaTail(a.umbrellaAt(0)); got != "update (3)" {
		t.Fatalf("tail %q", got)
	}
	a.act(platform.KeyEnter)
	if got := strings.Join(viewKeys(a), " "); got != "b:gnet@2026-10-15 alpha bravo charlie solo" || a.CursorKey() != "b:gnet@2026-10-15" {
		t.Fatalf("open view %q cursor %q", got, a.CursorKey())
	}
	// under the debut order only two share a day: no batch
	a.SetSort(data.SortDebut)
	if got := strings.Join(viewKeys(a), " "); strings.Contains(got, "b:") {
		t.Fatalf("debut view %q", got)
	}
	// beta hidden: flat
	a = batchApp(false)
	a.SetSort(data.SortUpdated)
	if got := strings.Join(viewKeys(a), " "); got != "alpha bravo charlie solo" {
		t.Fatalf("beta off view %q", got)
	}
}

func TestBatchSearchOpensUntilClosed(t *testing.T) {
	a := batchApp(true)
	a.SetSort(data.SortUpdated)
	a.query = "brav"
	a.rebuild()
	if got := strings.Join(viewKeys(a), " "); got != "b:gnet@2026-10-15 bravo" {
		t.Fatalf("search view %q", got)
	}
	if got := a.umbrellaTail(a.umbrellaAt(0)); got != "update (1/3)" {
		t.Fatalf("search tail %q", got)
	}
	a.cursor = 0
	a.act(platform.KeyEnter)
	if got := strings.Join(viewKeys(a), " "); got != "b:gnet@2026-10-15" {
		t.Fatalf("closed during search %q", got)
	}
	// a new query opens every batch again
	a.query = "bravo"
	a.rebuild()
	if got := strings.Join(viewKeys(a), " "); got != "b:gnet@2026-10-15 bravo" {
		t.Fatalf("new query %q", got)
	}
	// the search over, the batch is closed as it was before
	a.query = ""
	a.rebuild()
	if got := strings.Join(viewKeys(a), " "); got != "b:gnet@2026-10-15 solo" {
		t.Fatalf("after search %q", got)
	}
}
