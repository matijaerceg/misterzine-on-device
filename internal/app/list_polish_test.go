package app

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func TestListDateFormats(t *testing.T) {
	for _, c := range []struct{ format, iso, want string }{
		{"mm-dd", "2026-09-07", "09-07"}, {"mm-dd", "2025-12-31", "2025"},
		{"dd-mm", "2026-09-07", "07-09"}, {"dd-mm", "2025-12-31", "2025"},
		{"mon-d", "2026-09-07", "Sep 7"}, {"mon-d", "2026-12-12", "Dec 12"}, {"mon-d", "2025-12-31", "2025"},
		{"d-mon", "2026-09-07", "7 Sep"}, {"d-mon", "2026-12-12", "12 Dec"},
		{"yymmdd", "2026-09-07", "260907"}, {"yymmdd", "2025-12-31", "251231"},
		{"mm-dd", "", ""}, {"yymmdd", "2026-9", ""},
	} {
		if got := formatListDate(c.format, c.iso, 2026); got != c.want {
			t.Errorf("%s %q = %q, want %q", c.format, c.iso, got, c.want)
		}
	}
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{{K: "a", Title: "Alpha", Updated: "2026-09-07"}, {K: "b", Title: "Beta", Updated: "2025-01-02"}}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return now }}, data.Ingest(rows, "", now), nil)
	if a.DateFormat() != "mm-dd" || a.dateCols() != 5 || a.dateCol("2025-01-02") != " 2025" {
		t.Fatalf("default format: %q cols %d", a.dateCol("2025-01-02"), a.dateCols())
	}
	wide := a.lay.TitleCol
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "date-format" {
			a.panel.cursor = i
		}
	}
	for _, want := range []struct{ format, day, old string }{
		{"dd-mm", "07-09", " 2025"}, {"mon-d", " Sep 7", "  2025"}, {"d-mon", " 7 Sep", "  2025"}, {"yymmdd", "260907", "250102"},
	} {
		a.actPanel(platform.KeyRight)
		if a.DateFormat() != want.format || a.dateCol("2026-09-07") != want.day || a.dateCol("2025-01-02") != want.old {
			t.Fatalf("%s: %q %q", a.DateFormat(), a.dateCol("2026-09-07"), a.dateCol("2025-01-02"))
		}
		if a.lay.TitleCol != wide-(a.dateCols()-5) {
			t.Fatalf("%s: title columns %d, want %d", want.format, a.lay.TitleCol, wide-(a.dateCols()-5))
		}
		if !strings.Contains(a.panel.entries[a.panel.cursor].help, strings.TrimSpace(a.dateCol("2026-09-11"))) {
			t.Fatalf("%s: help %q lacks today's example", want.format, a.panel.entries[a.panel.cursor].help)
		}
	}
	if a.actPanel(platform.KeyRight) {
		t.Fatal("no format past the last")
	}
}

func TestThumbSlotPreference(t *testing.T) {
	row := &data.Row{Img: "g", ImgSlots: []string{"title", "snap", "ingame"}}
	if _, s := thumbSlot(row, "gameplay"); s != "snap" {
		t.Fatalf("gameplay picked %s", s)
	}
	if _, s := thumbSlot(row, "title"); s != "title" {
		t.Fatalf("title picked %s", s)
	}
	row.ImgSlots = []string{"title", "ingame"}
	if _, s := thumbSlot(row, ""); s != "ingame" {
		t.Fatalf("default without a snap picked %s", s)
	}
	row.ImgSlots = []string{"snap"}
	if _, s := thumbSlot(row, "title"); s != "snap" {
		t.Fatalf("title preference without a title picked %s", s)
	}
	a := New(Config{PhysW: 320, PhysH: 240, ListShot: "title"}, data.Ingest([]data.Row{{K: "a", Title: "A"}}, "", time.Now()), nil)
	if a.ListShot() != "title" {
		t.Fatal("preference not carried")
	}
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "list-shot" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyLeft)
	if a.ListShot() != "gameplay" || a.panel.entries[a.panel.cursor].idx != 0 {
		t.Fatal("Left did not choose gameplay")
	}
}

// titleInk counts lit pixels of a colour inside the first row's title area.
func titleInk(a *App, col rgb) int {
	r := a.lay.lineRect(0)
	n := 0
	for y := r.Min.Y - 1; y < r.Max.Y; y++ {
		for x := r.Min.X + a.body.W; x < r.Min.X+a.body.W*(a.lay.TitleCol+1); x++ {
			if a.logical.RGBA.RGBAAt(x, y) == col {
				n++
			}
		}
	}
	return n
}

func TestNarrowTitlesAndBetaSign(t *testing.T) {
	long := strings.Repeat("Mmmmmmmmm ", 8)
	rows := []data.Row{{K: "a", Title: long, Updated: "2026-09-07", Beta: true}}
	a := New(Config{PhysW: 320, PhysH: 240, NarrowTitles: true}, data.Ingest(rows, "", time.Now()), nil)
	a.Paint()
	if titleInk(a, gen.Eva.Warn) == 0 {
		t.Fatal("beta sign missing after a narrow title")
	}
	w := a.lay.TitleCol * a.body.W
	narrow, normal := gfx.FitProp(a.narrow, long, w), gfx.Fit(long, a.lay.TitleCol)
	if !strings.HasSuffix(narrow, gfx.Ellipsis) || len(narrow) < len(normal)*5/4 {
		t.Fatalf("narrow font fits %d characters, normal %d: want at least a quarter more", len(narrow), len(normal))
	}
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "title-font" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyLeft)
	if a.NarrowTitles() {
		t.Fatal("Left did not choose the normal font")
	}
	a.actPanel(platform.KeyBack)
	a.Paint()
	if titleInk(a, gen.Eva.Warn) == 0 {
		t.Fatal("beta sign missing after a normal title")
	}
}

func TestVersionLineMarqueeScrolls(t *testing.T) {
	clock := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	long := "_Arcade/_alternatives/_Galaga/Galaga (Midway set 1, fast shoot hack, bootleg set 2, extra long).mra"
	rows := []data.Row{{K: "a", Title: "Galaga", Base: "Arcade", MRA: "_Arcade/Galaga.mra", SN: "galagamw", Updated: "2026-09-07"}}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock },
		Alternatives: func(*data.Row) []string { return []string{long} }}, data.Ingest(rows, "", clock), nil)
	a.actList(platform.KeyEnter)
	a.Paint()
	if a.marquee.key != "" {
		t.Fatal("a version that fits must not scroll")
	}
	a.actDetails(platform.KeyDown)
	a.Paint()
	if a.marquee.key == "" || a.marquee.span <= 0 {
		t.Fatalf("overlong alternative must start the marquee: %+v", a.marquee)
	}
	if next := a.NextTick(); next.IsZero() || next.Sub(clock) > marqueeFrame {
		t.Fatalf("next tick %v, want within a frame", next)
	}
	clock = clock.Add(marqueePause + 400*time.Millisecond)
	if !a.Tick(clock) {
		t.Fatal("a due frame must repaint")
	}
	before := a.marquee
	a.Paint()
	off := marqueeOffset(clock.Sub(before.start), before.span)
	if off != 16 { // 400 ms at 40 px/s
		t.Fatalf("offset %d after the pause, want 16", off)
	}
	span := before.span
	if marqueeOffset(marqueePause+time.Duration(span)*time.Second/marqueeSpeed+100*time.Millisecond, span) != span {
		t.Fatal("must rest at the far end")
	}
	if marqueeOffset(2*marqueePause+time.Duration(span)*time.Second/marqueeSpeed+time.Second, span) >= span {
		t.Fatal("must return after resting")
	}
	// the drawn label stays inside its line: nothing lands on the tally
	body := a.lay.Body
	body.Min.Y = a.lay.Root.Min.Y
	y := body.Max.Y - (a.sm.H + 1)
	count := image.Rect(body.Max.X-a.sm.Width("2/2"), y, body.Max.X, y+a.sm.H)
	for x := count.Min.X; x < count.Max.X; x++ {
		for py := count.Min.Y; py < count.Max.Y; py++ {
			if a.logical.RGBA.RGBAAt(x, py) == gen.Eva.Accent {
				t.Fatal("scrolling label overlaps the tally")
			}
		}
	}
	a.actDetails(platform.KeyBack)
	a.Paint()
	if a.marquee.key != "" || !a.NextTick().IsZero() && a.NextTick().Sub(clock) <= marqueeFrame {
		t.Fatal("leaving Details must stop the marquee")
	}
}

func TestFilterHeadingOpensAndCloses(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade", Title: "A", Genre: "Shooter"}, {K: "b", Base: "Arcade", Title: "B", Genre: "Puzzle"},
	}, "", time.Now()), nil)
	a.openPanel(ScreenFilter)
	for i, e := range a.panel.entries {
		if e.kind == "genre" && e.header {
			a.panel.cursor = i
		}
	}
	if !a.panel.sectionClosed["genre"] || !strings.Contains(a.filterHint(), "A "+gfx.ArrowLeft+" "+gfx.ArrowRight+" open/close") {
		t.Fatalf("unused section starts closed with the heading legend; hint %q", a.filterHint())
	}
	a.actPanel(platform.KeyEnter)
	if a.panel.sectionClosed["genre"] || a.filters.Active() {
		t.Fatal("A must open the heading without filtering")
	}
	values := 0
	for _, e := range a.panel.entries {
		if e.kind == "genre" && !e.header {
			values++
		}
	}
	if values != 2 {
		t.Fatalf("%d genre values listed after opening", values)
	}
	a.actPanel(platform.KeyDown)
	if strings.HasPrefix(a.filterHint(), "A "+gfx.ArrowLeft) {
		t.Fatal("value rows keep the toggle legend")
	}
	a.actPanel(platform.KeyUp)
	a.actPanel(platform.KeyEnter)
	if !a.panel.sectionClosed["genre"] || a.panel.entries[a.panel.cursor].kind != "genre" {
		t.Fatal("A must close the open heading and keep the cursor on it")
	}
}
