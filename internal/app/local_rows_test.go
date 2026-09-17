package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func localTestRows() []data.Row {
	cat := []data.Row{
		{K: "colony7", Title: "Colony 7", Base: "Arcade", Src: "distribution_mister", Core: "defender", SN: "colony7", MRA: "_Arcade/Colony 7 (Set 1).mra", Updated: "2026-07-14", Img: "colony7", ImgSlots: []string{"title", "snap"}, ImgW: 240, ImgH: 292},
	}
	two := 2
	local := []data.Row{
		{K: "local:orphanf", Title: "Orphan Fighter", Base: "Arcade", Src: data.SrcLocal, SN: "orphanf", Core: "defender", MRA: "_Arcade/_Extra/Orphan Fighter.mra",
			Year: "1983", Manufacturer: "Homebrew Works", Rot: "Vertical (CW)", Plr: "2", Ctl: "8-way · 2 buttons", Buttons: &two, Note: "MRA category: Shooter"},
		{K: "local:locpuz", Title: "Local Puzzle", Base: "Arcade", Src: data.SrcLocal, SN: "locpuz", Core: "puzzlecore", MRA: "_Arcade/Local Puzzle.mra",
			Img: "locpuz", ImgSlots: []string{"lsnap"}, ImgW: 4, ImgH: 3},
	}
	return data.MergeLocal(cat, local)
}

func TestLocalRowDetails(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, Status: func(int) data.Status { return data.StatusFoundUndated }}, data.Ingest(localTestRows(), "h", time.Now()), nil)
	i := a.ds.Index("local:orphanf")
	if i < 0 {
		t.Fatal("local row not indexed")
	}
	row, d := &a.ds.Rows[i], &a.ds.Der[i]
	var texts []string
	for _, l := range a.detailLines(row, d, i) {
		texts = append(texts, l.text)
	}
	joined := strings.Join(texts, "\n")
	for _, want := range []string{
		"Not in the MisterZine catalogue",
		"File:     _Arcade/_Extra/Orphan Fighter.mra",
		"Source:   " + data.SrcLocalFull,
		"Mfr:      Homebrew Works",
		"Controls: 8-way / 2 buttons",
		"Note:     MRA category: Shooter",
		"Card:     ",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("details lack %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "Updated:") || strings.Contains(joined, "Debut:") {
		t.Fatalf("a local row has no release dates:\n%s", joined)
	}
	if ch := chips(row, d); len(ch) != 1 || ch[0] != "local" {
		t.Fatalf("chips %v", ch)
	}
	if ch := chips(&a.ds.Rows[0], &a.ds.Der[0]); len(ch) != 0 {
		t.Fatalf("catalogue row chips %v", ch)
	}
	// The catalogue row keeps its picture; the local rows show a placeholder
	// or ask the service slot.
	if k, s := thumbSlot(&a.ds.Rows[0], "gameplay"); k != "colony7" || s != "snap" {
		t.Fatalf("catalogue thumb %q %q", k, s)
	}
	if k, s := thumbSlot(row, "gameplay"); k != "" || s != "" {
		t.Fatalf("local row without a picture: %q %q", k, s)
	}
	j := a.ds.Index("local:locpuz")
	if k, s := thumbSlot(&a.ds.Rows[j], "gameplay"); k != "locpuz" || s != "lsnap" {
		t.Fatalf("service slot: %q %q", k, s)
	}
	// Launch entries: the file itself, verified on the card.
	a.cfg.Exists = func(rel string) bool { return rel == row.MRA }
	if e := a.launchEntries(row, i); len(e) != 1 || e[0].path != row.MRA || !e[0].ok {
		t.Fatalf("launch entries %+v", e)
	}
}

func TestFiltersSourceLocal(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(localTestRows(), "h", time.Now()), nil)
	if a.ds.Facets.Src[data.SrcLocal] != 2 {
		t.Fatalf("facet %v", a.ds.Facets.Src)
	}
	openExpandedFilters(a)
	found := false
	for _, e := range a.panel.entries {
		if e.kind == "src" && e.value == data.SrcLocal && !e.header {
			found = e.text == "Local" && e.count == 2 && e.checked
		}
	}
	if !found {
		t.Fatal("Filters lack a checked Local source with its count")
	}
	if n := len(a.view); n != 3 {
		t.Fatalf("visible rows %d", n)
	}
	a.SetFilters(data.Filters{SrcOff: map[string]bool{data.SrcLocal: true}})
	if n := len(a.view); n != 1 {
		t.Fatalf("Local off: visible rows %d", n)
	}
	// Installed-only hides sources without a Downloader database, never Local.
	a.cfg.InstalledOnly = true
	a.SetHiddenSources(data.HiddenSources([]data.DB{{ID: "distribution_mister"}}), true)
	a.SetFilters(data.Filters{})
	if n := len(a.view); n != 3 {
		t.Fatalf("installed only hid local rows: %d visible", n)
	}
}
