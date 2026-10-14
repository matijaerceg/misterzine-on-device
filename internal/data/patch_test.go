package data

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPatchFieldAndFilter(t *testing.T) {
	var rows []Row
	src := `[{"title":"A","base":"Arcade","pt":[{"t":"translation","n":"N","s":"","v":"rc3","d":"2026-10-11","u":"https://arcadepatches.com/a/","sn":"aen"},{"t":"arrange","n":"N2","s":"","v":"","d":"","u":"","sn":""}]},
	{"title":"B","base":"Arcade"},{"title":"C","base":"Console"}]`
	if err := json.Unmarshal([]byte(src), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows[0].PT) != 2 || rows[0].PT[0].SN != "aen" || rows[0].PT[0].V != "rc3" {
		t.Fatalf("pt = %+v", rows[0].PT)
	}
	ds := Ingest(rows, "", time.Time{})
	if ds.Facets.Patch["translation"] != 1 || ds.Facets.Patch["arrange"] != 1 || ds.Facets.Patch[""] != 1 || len(ds.Facets.Patch) != 3 {
		t.Fatalf("facet = %v", ds.Facets.Patch)
	}
	pass := func(f Filters) []string {
		var out []string
		for i := range ds.Rows {
			if f.Pass(&ds.Rows[i], &ds.Der[i], StatusUnknown, false, false, ROMUnknown) {
				out = append(out, ds.Rows[i].Title)
			}
		}
		return out
	}
	if got := pass(Filters{PatchOff: map[string]bool{}}); len(got) != 3 {
		t.Fatalf("all on: %v", got)
	}
	if got := pass(Filters{PatchOff: map[string]bool{"": true}}); len(got) != 2 || got[0] != "A" || got[1] != "C" {
		t.Fatalf("None off (non-arcade unaffected): %v", got)
	}
	if got := pass(Filters{PatchOff: map[string]bool{"translation": true}}); len(got) != 3 {
		t.Fatalf("one of two kinds off keeps the row: %v", got)
	}
	f := Filters{PatchOff: map[string]bool{"translation": true, "arrange": true}}
	if got := pass(f); len(got) != 2 || got[0] != "B" {
		t.Fatalf("both kinds off: %v", got)
	}
	if !f.Active() {
		t.Fatal("PatchOff not active")
	}
	if w := f.Why(&ds.Rows[0], &ds.Der[0], StatusUnknown, false, false, ROMUnknown); w != "Filters -> Patches (Translation, Arranged soundtrack off)" {
		t.Fatalf("why = %q", w)
	}
	b, _ := json.Marshal(Filters{PatchOff: map[string]bool{"": true}})
	if string(b) != `{"patch_off":{"":true}}` {
		t.Fatalf("persisted as %s", b)
	}
}
