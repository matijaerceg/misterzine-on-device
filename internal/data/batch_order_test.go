package data

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestArrivalBatchOrder(t *testing.T) {
	// The later batch's Z core must precede the earlier batch's A core.
	// Within one batch, core and title grouping still apply; date outranks b.
	rows, err := DecodeRows(strings.NewReader(`[
		{"k":"morning","title":"A","core":"a","updated":"2026-09-08","date":"2026-09-01","b":10},
		{"k":"later-z","title":"Z","core":"z","updated":"2026-09-08","date":"2026-09-01","b":11},
		{"k":"later-a","title":"B","core":"z","updated":"2026-09-08","date":"2026-09-01","b":11},
		{"k":"later-core","title":"C","core":"b","updated":"2026-09-08","date":"2026-09-01","b":11},
		{"k":"old-date","title":"D","core":"a","updated":"2026-09-07","date":"2026-09-01","b":99},
		{"k":"missing-date","title":"E","core":"a","date":"2026-09-01","b":100},
		{"k":"legacy","title":"F","core":"a","updated":"2026-09-08","date":"2026-09-01"}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	ds := Ingest(rows, "", time.Time{})
	if got, want := keysOf(ds, ds.Order(SortUpdated)), []string{"later-core", "later-a", "later-z", "morning", "legacy", "old-date", "missing-date"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("updated: %v", got)
	}
	if got, want := keysOf(ds, ds.Order(SortDebut)), []string{"morning", "later-a", "later-core", "old-date", "missing-date", "legacy", "later-z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("debut: %v", got)
	}
}
