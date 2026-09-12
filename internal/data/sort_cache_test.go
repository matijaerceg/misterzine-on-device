package data

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Order sorts each mode once per Dataset and hands the same slice back
// after that; a fresh Ingest of the same rows starts over and agrees.
func TestOrderCachedPerMode(t *testing.T) {
	src := `[
		{"k":"a","title":"Zed","core":"z","updated":"2026-09-08","date":"2026-09-01","year":"1989"},
		{"k":"b","title":"Alpha","core":"a","updated":"2026-09-07","date":"2026-09-02","year":"1991"},
		{"k":"c","title":"Mid","core":"m","updated":"2026-09-09","date":"2026-08-30","year":"19xx"},
		{"k":"d","title":"Beta","core":"b","updated":"","date":"","year":"1991"}
	]`
	rows, err := DecodeRows(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	ds := Ingest(rows, "", time.Time{})
	rows2, _ := DecodeRows(strings.NewReader(src))
	fresh := Ingest(rows2, "", time.Time{})
	for _, mode := range append(append([]SortMode{}, SortCycle...), SortRecents) {
		first := ds.Order(mode)
		again := ds.Order(mode)
		if len(first) > 0 && &first[0] != &again[0] {
			t.Errorf("%v: second Order call sorted again", mode)
		}
		if got, want := keysOf(ds, again), keysOf(fresh, fresh.Order(mode)); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: cached %v, fresh %v", mode, got, want)
		}
	}
	if got, want := keysOf(ds, ds.Order(SortYear)), []string{"b", "d", "a", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("year: %v", got)
	}
	if got, want := keysOf(ds, ds.Order(SortUpdated)), []string{"c", "a", "b", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("updated: %v", got)
	}
}

// BenchmarkOrder sorts the real catalogue in each mode from scratch.
func BenchmarkOrder(b *testing.B) {
	f, err := os.Open(filepath.Join("..", "..", "testdata", "data.json"))
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	rows, err := DecodeRows(f)
	if err != nil {
		b.Fatal(err)
	}
	ds := Ingest(rows, "", time.Time{})
	for _, mode := range SortCycle {
		b.Run(mode.String(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ds.orders = nil
				ds.Order(mode)
			}
		})
	}
}
