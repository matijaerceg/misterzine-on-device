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
	wide := a.lay.TitleW
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
		if a.lay.TitleW != wide-(a.dateCols()-5)*a.rowFont().W {
			t.Fatalf("%s: title width %d, want %d", want.format, a.lay.TitleW, wide-(a.dateCols()-5)*a.rowFont().W)
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
		for x := r.Min.X + a.body.W; x < r.Min.X+a.body.W+a.lay.TitleW; x++ {
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
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	if a.TitleFont() != "tall" || a.rowFont() != a.tall {
		t.Fatal("narrow tall is the default")
	}
	a.Paint()
	if titleInk(a, gen.Eva.Warn) == 0 {
		t.Fatal("beta sign missing after a tall title")
	}
	if a.narrow.PropWidth(long) != a.tall.PropWidth(long) || a.tall.Glyph('M')[2] == 0 || a.narrow.Glyph('M')[2] != 0 {
		t.Fatal("the tall font must keep the narrow widths and add a row to capitals")
	}
	w := a.lay.TitleW
	narrow, normal := gfx.FitProp(a.narrow, long, w), gfx.Fit(long, a.body.Cols(w))
	if !strings.HasSuffix(narrow, gfx.Ellipsis) || len(narrow)*100 < len(normal)*115 {
		t.Fatalf("narrow font fits %d characters, normal %d: want at least 15 percent more", len(narrow), len(normal))
	}
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "title-font" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyLeft)
	if a.TitleFont() != "narrow" || a.rowFont() != a.narrow {
		t.Fatal("Left did not choose the narrow font for titles and the row")
	}
	a.actPanel(platform.KeyBack)
	a.Paint()
	if titleInk(a, gen.Eva.Warn) == 0 {
		t.Fatal("beta sign missing after a narrow title")
	}
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "title-font" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyLeft)
	if a.TitleFont() != "normal" || a.rowFont() != a.tall {
		t.Fatal("normal titles keep the tall row font")
	}
	a.actPanel(platform.KeyBack)
	a.Paint()
	if titleInk(a, gen.Eva.Warn) == 0 {
		t.Fatal("beta sign missing after a normal title")
	}
	a.openPanel(ScreenOptions)
	for i, e := range a.panel.entries {
		if e.kind == "title-font" {
			a.panel.cursor = i
		}
	}
	a.actPanel(platform.KeyRight)
	a.actPanel(platform.KeyRight)
	if a.TitleFont() != "tall" || a.actPanel(platform.KeyRight) {
		t.Fatal("Right twice must reach narrow tall, the last choice")
	}
	a.actPanel(platform.KeyBack)
	a.Paint()
	if titleInk(a, gen.Eva.Warn) == 0 {
		t.Fatal("beta sign missing after a tall title")
	}
}

func TestClearAllFiltersRowIsAlwaysListed(t *testing.T) {
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest([]data.Row{
		{K: "a", Base: "Arcade", Title: "A", Genre: "Shooter"}, {K: "b", Base: "Arcade", Title: "B", Genre: "Puzzle"},
	}, "", time.Now()), nil)
	a.openPanel(ScreenFilter)
	e := a.panel.entries
	if e[0].kind != "clear" || !e[0].disabled || !(e[1].header && e[1].info && e[1].text == "") || e[2].kind != "install" || !e[2].header {
		t.Fatalf("want a greyed Clear row, a blank line, then the first heading; got %+v %+v %+v", e[0], e[1], e[2])
	}
	if a.panel.cursor != 2 {
		t.Fatalf("cursor starts on the first heading, not the greyed row: %d", a.panel.cursor)
	}
	a.panel.cursor = 0
	if a.actPanel(platform.KeyEnter) || a.filters.Active() {
		t.Fatal("a greyed Clear row does nothing")
	}
	a.SetFilters(data.Filters{GenreOff: map[string]bool{"Puzzle": true}})
	a.openPanel(ScreenFilter)
	if a.panel.entries[0].disabled || a.panel.cursor != 0 {
		t.Fatal("an active filter enables the row and opens on it")
	}
	a.actPanel(platform.KeyEnter)
	if a.filters.Active() || !a.panel.entries[0].disabled || len(a.view) != 2 || a.panel.cursor != 2 {
		t.Fatal("Clear must reset the filters, grey out and move to the first heading")
	}
}

func TestOptionsLayoutCuesAndColumn(t *testing.T) {
	rows := []data.Row{{K: "a", Title: "A", Updated: "2026-09-07"}}
	a := New(Config{PhysW: 320, PhysH: 240, Version: "v9.9.9-test"}, data.Ingest(rows, "", time.Now()), nil)
	a.openPanel(ScreenOptions)
	a.Paint()
	l := &a.lay
	font := a.sm
	count := func(r image.Rectangle, col rgb) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if a.logical.RGBA.RGBAAt(x, y) == col {
					n++
				}
			}
		}
		return n
	}
	// the build and data lines sit just above the hint bar, greyed
	versionRows := image.Rect(l.Body.Min.X, l.Body.Max.Y-2*font.H-2, l.Body.Max.X, l.Body.Max.Y)
	if count(versionRows, gen.Eva.Muted) == 0 || count(versionRows, gen.Eva.Line) != 0 {
		t.Fatal("version rows must be greyed and unframed above the hint bar")
	}
	// the help text is framed
	helpBox := image.Rect(l.Body.Min.X, versionRows.Min.Y-(3*(font.H+1)+5), l.Body.Max.X, versionRows.Min.Y)
	if count(image.Rect(helpBox.Min.X, helpBox.Min.Y, helpBox.Max.X, helpBox.Min.Y+1), gen.Eva.Line) != helpBox.Dx() {
		t.Fatal("help text must be framed")
	}
	if count(image.Rect(l.Body.Min.X, l.Body.Min.Y, l.Body.Max.X, l.Body.Min.Y+1), gen.Eva.Line) != 0 {
		t.Fatal("the entries must not be framed")
	}
	// more entries below than fit: a cue at the bottom, none at the top
	inner := image.Rect(l.Body.Min.X+2, l.Body.Min.Y+2, l.Body.Max.X-2, helpBox.Min.Y-2)
	lines := inner.Dy() / font.H
	if lines >= len(a.panel.entries) {
		t.Skip("every option fits")
	}
	edge := inner.Max.X - 2 - font.W
	top := image.Rect(edge, inner.Min.Y, edge+font.W, inner.Min.Y+font.H)
	below := image.Rect(edge, inner.Min.Y+font.H, edge+font.W, inner.Max.Y) // the gutter under the first row
	if count(top, gen.Eva.Muted) != 0 || count(below, gen.Eva.Muted) == 0 {
		t.Fatal("want a down cue only, before any scrolling")
	}
	a.actPanel(platform.KeyEnd)
	a.Paint()
	if count(top, gen.Eva.Muted) == 0 || count(below, gen.Eva.Muted) != 0 {
		t.Fatal("want an up cue only at the end")
	}
	// the selection sits near the middle while scrolling, as in the main list
	mid := len(a.panel.entries) / 2
	a.panel.cursor = mid
	a.all = true
	a.Paint()
	if a.panel.top == 0 || mid-a.panel.top < lines/3 || mid-a.panel.top > 2*lines/3 {
		t.Fatalf("cursor %d sits at row %d of %d visible, want it near the center", mid, mid-a.panel.top, lines)
	}
	if count(top, gen.Eva.Muted) == 0 || count(below, gen.Eva.Muted) == 0 {
		t.Fatal("rows out of view both ways must show both cues")
	}
	// values start in one column at the two-thirds mark in the horizontal layout
	vx := a.valueColumn(inner, edge)
	if vx != inner.Min.X+inner.Dx()/2 {
		t.Fatalf("value column %d, want the half mark %d", vx, inner.Min.X+inner.Dx()/2)
	}
	a.SetRotation(gfx.RotLeft)
	a.SetInset(40, 40)
	a.openPanel(ScreenOptions)
	a.Paint()
	l = &a.lay
	inner = image.Rect(l.Body.Min.X+2, l.Body.Min.Y+2, l.Body.Max.X-2, l.Body.Max.Y-2*font.H-2-(4*(font.H+1)+5)-2)
	if a.valueColumn(inner, inner.Max.X-2-font.W) != 0 {
		t.Fatal("the narrow tate layout right-aligns values")
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
	a.actDetails(platform.KeyRight)
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

func TestPageJumpsCenterTheRow(t *testing.T) {
	var rows []data.Row
	for i := 0; i < 80; i++ {
		rows = append(rows, data.Row{K: itoa(i), Title: "Game " + itoa(i), Updated: "2026-09-07"})
	}
	a := New(Config{PhysW: 320, PhysH: 240}, data.Ingest(rows, "", time.Now()), nil)
	a.actList(platform.KeyRight)
	if a.cursor != a.lay.Lines {
		t.Fatalf("Right moved to %d, want a screen of rows (%d)", a.cursor, a.lay.Lines)
	}
	if a.top != centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines) || a.top == a.screenLine(a.cursor) {
		t.Fatalf("Right left the row at line %d of top %d, want it centered", a.screenLine(a.cursor), a.top)
	}
	a.actList(platform.KeyLeft)
	if a.cursor != 0 || a.top != 0 {
		t.Fatal("Left must return to the first row at the top")
	}
	a.actList(platform.KeyEnd)
	a.actList(platform.KeyLeft)
	if a.cursor != len(a.view)-1-a.lay.Lines || a.top != centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines) {
		t.Fatal("Left from the end must step a screen up and center")
	}
}

func TestDetailsInformationSlides(t *testing.T) {
	clock := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	row := data.Row{K: "game", Title: "Game", Core: "Game", Note: strings.Repeat("Every word of this information must remain reachable. ", 20)}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock },
		Alternatives: func(*data.Row) []string {
			return []string{"_Arcade/_alternatives/A.mra", "_Arcade/_alternatives/B.mra"}
		}},
		data.Ingest([]data.Row{row}, "", clock), nil)
	a.actList(platform.KeyEnter)
	a.Paint()
	a.actDetails(platform.KeyRight)
	if a.detail.pick != 1 {
		t.Fatal("Right must choose the next version")
	}
	a.actDetails(platform.KeyLeft)
	if a.detail.pick != 0 {
		t.Fatal("Left must choose the previous version")
	}
	a.actDetails(platform.KeyDown)
	a.Paint()
	lh := a.detailLine()
	page := max(1, a.detail.lines-1)
	if a.detail.scroll <= 0 || a.detail.scroll > page || a.detail.pixel != 0 {
		t.Fatalf("Down wants a page (up to line %d) and starts sliding from 0: scroll=%d pixel=%d", page, a.detail.scroll, a.detail.pixel)
	}
	page = a.detail.scroll // the note may be a little shorter than a full page
	if next := a.NextTick(); next.IsZero() || next.After(clock) {
		t.Fatalf("the slide must be due now, got %v", next)
	}
	for i := 1; i <= 3; i++ {
		clock = clock.Add(detailFrame)
		if !a.Tick(clock) || a.detail.pixel != detailStep*i {
			t.Fatalf("frame %d: pixel %d, want %d", i, a.detail.pixel, detailStep*i)
		}
		a.Paint()
	}
	if next := a.NextTick(); next.Sub(clock) != detailFrame {
		t.Fatalf("next frame in %v, want %v", next.Sub(clock), detailFrame)
	}
	for a.detail.pixel != a.detail.scroll*lh {
		clock = clock.Add(detailFrame)
		a.Tick(clock)
		a.Paint()
	}
	if !a.NextTick().IsZero() && a.NextTick().Sub(clock) <= detailFrame {
		t.Fatal("a finished slide must stop ticking")
	}
	// held Up under the frame loop slides back the same way
	a.actDetails(platform.KeyUp)
	clock = clock.Add(detailFrame)
	if !a.Frame(clock) || a.detail.pixel != page*lh-detailStep {
		t.Fatalf("Frame did not step the slide: pixel %d", a.detail.pixel)
	}
}

func TestYearSortOrderAndJumps(t *testing.T) {
	clock := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	rows := []data.Row{
		{K: "u", Title: "Unknown", Base: "Arcade", Updated: "2026-09-01"},
		{K: "b", Title: "Bravo", Base: "Arcade", Year: "1985", Updated: "2026-09-02"},
		{K: "q", Title: "Quirk", Base: "Arcade", Year: "198?", Updated: "2026-09-03"},
		{K: "n", Title: "New", Base: "Arcade", Year: "1990", Updated: "2026-09-04"},
		{K: "a", Title: "Alpha", Base: "Arcade", Year: "1985", Updated: "2026-09-05"},
	}
	ds := data.Ingest(rows, "", clock)
	var got []string
	for _, i := range ds.Order(data.SortYear) {
		got = append(got, ds.Rows[i].K)
	}
	if strings.Join(got, "") != "nabqu" {
		t.Fatalf("year order %v, want newest year first, titles within a year, unknown years last", got)
	}
	if data.NextSort(data.SortDebut) != data.SortYear || data.NextSort(data.SortYear) != data.SortAlphabetical || data.NextSort(data.SortFavorites) != data.SortUpdated {
		t.Fatal("Y must walk Updated, Debut, Year, A-Z, Favorites")
	}
	a := New(Config{PhysW: 320, PhysH: 240, Now: func() time.Time { return clock }, TimerNow: func() time.Time { return clock }}, ds, nil)
	a.actList(platform.KeySpace)
	a.actList(platform.KeySpace)
	if a.Sort() != data.SortYear || a.CursorKey() != "a" {
		t.Fatalf("two presses reach the year order and keep the row: %v %q", a.Sort(), a.CursorKey())
	}
	a.actList(platform.KeyHome)
	if a.CursorKey() != "n" {
		t.Fatalf("the newest year leads: %q", a.CursorKey())
	}
	a.actList(platform.KeyPageDown)
	if a.CursorKey() != "a" || a.notice != "1985" {
		t.Fatalf("R must jump to the next year's first title: %q notice %q", a.CursorKey(), a.notice)
	}
	a.actList(platform.KeyPageDown)
	if a.CursorKey() != "q" || a.notice != "Year unknown" {
		t.Fatalf("R must jump to the unknown years: %q notice %q", a.CursorKey(), a.notice)
	}
	a.actList(platform.KeyPageUp)
	if a.CursorKey() != "a" {
		t.Fatal("L must jump back a year")
	}
	a.Paint()
	rf := a.rowFont()
	r := a.lay.lineRect(0)
	col := image.Rect(r.Max.X-a.dateCols()*rf.W, r.Min.Y-1, r.Max.X, r.Max.Y)
	lit := 0
	for y := col.Min.Y; y < col.Max.Y; y++ {
		for x := col.Min.X; x < col.Max.X; x++ {
			if a.logical.RGBA.RGBAAt(x, y) == gen.Eva.Muted {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatal("the date column must show the year")
	}
	b := New(Config{PhysW: 320, PhysH: 240, RememberSort: true, LastSort: data.SortYear}, ds, nil)
	if b.Sort() != data.SortYear {
		t.Fatal("the year order must be remembered")
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
