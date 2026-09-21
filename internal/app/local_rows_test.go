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

// A standin row sits beside the catalogue row whose core the card has not
// got. Its Details say why it is there, rather than claiming the catalogue
// has never heard of the game.
func TestStandinRowDetails(t *testing.T) {
	cat := []data.Row{{K: "volfied", Title: "Volfied", Base: "Arcade", Src: "jtbindb", Core: "jtvlfied", SN: "volfied",
		MRA: "_Arcade/Volfied (World, rev 1).mra", Beta: true, Gate: "jtbeta", Date: "2026-08-16", Updated: "2026-09-04"}}
	stand := data.Row{K: "local:volfied", Title: "Volfied", Base: "Arcade", Src: data.SrcLocal, SN: "volfied",
		Core: "taitox", MRA: "_Arcade/_Extra/Volfied.mra", Standin: true}
	rows := data.MergeLocal(cat, []data.Row{stand})
	if len(rows) != 2 {
		t.Fatalf("the standin row must survive the merge: %+v", rows)
	}
	a := New(Config{PhysW: 320, PhysH: 240, Status: func(int) data.Status { return data.StatusFoundUndated }},
		data.Ingest(rows, "h", time.Now()), nil)
	i := a.ds.Index("local:volfied")
	if i < 0 {
		t.Fatal("standin row not indexed")
	}
	row, d := &a.ds.Rows[i], &a.ds.Der[i]
	var texts []string
	for _, l := range a.detailLines(row, d, i) {
		texts = append(texts, l.text)
	}
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "The catalogue lists this game only for cores that are not on this card") {
		t.Errorf("details do not say why the row is here:\n%s", joined)
	}
	if strings.Contains(joined, "Not in the MisterZine catalogue") {
		t.Errorf("details deny a catalogue the game is in:\n%s", joined)
	}
	if !strings.Contains(joined, "File:     _Arcade/_Extra/Volfied.mra") {
		t.Errorf("details lack the file line:\n%s", joined)
	}
	if ch := chips(row, d); len(ch) != 1 || ch[0] != "local" {
		t.Fatalf("chips %v", ch)
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

// A local game whose core is not installed is greyed and must not be handed
// to MiSTer: the menu would sit on a load that never completes. Start shows
// a notice naming the core instead, for the file and its alternatives alike.
func TestLocalRowMissingCoreDoesNotLaunch(t *testing.T) {
	var launched []string
	a := New(Config{PhysW: 320, PhysH: 240,
		Status: func(int) data.Status { return data.StatusNotFound },
		Exists: func(string) bool { return true },
		Launch: func(p string) { launched = append(launched, p) },
		Alternatives: func(r *data.Row) []string {
			return []string{"_Arcade/_alternatives/_Orphan Fighter/Orphan Fighter (alt).mra"}
		},
	}, data.Ingest(localTestRows(), "h", time.Now()), nil)
	i := a.ds.Index("local:orphanf")
	row := &a.ds.Rows[i]
	entries := a.launchEntries(row, i)
	if len(entries) != 2 || entries[0].ok || entries[1].ok {
		t.Fatalf("entries %+v", entries)
	}
	for pick := range entries {
		a.notice = ""
		if !a.launchRow(row, i, pick) {
			t.Fatalf("pick %d: launched", pick)
		}
		if a.notice != "the defender core is not on the card" {
			t.Fatalf("pick %d: notice %q", pick, a.notice)
		}
	}
	if len(launched) != 0 {
		t.Fatalf("launched %v", launched)
	}
	// The file missing keeps its own notice.
	a.cfg.Exists = func(string) bool { return false }
	a.launchRow(row, i, 0)
	if a.notice != "that file is not on the card" {
		t.Fatalf("notice %q", a.notice)
	}
}

// Local games join the screensaver with their gameplay shot.
func TestSaverPoolIncludesLocalRows(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240, SaverStyle: "shots"}, data.Ingest(localTestRows(), "h", time.Now()), nil)
	var got []string
	for _, p := range a.saverPool() {
		got = append(got, a.ds.Rows[p.row].K+"/"+p.slot)
	}
	if strings.Join(got, " ") != "colony7/snap local:locpuz/lsnap" {
		t.Fatalf("pool %v", got)
	}
}
