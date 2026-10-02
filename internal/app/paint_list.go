package app

import (
	"image"
	"image/color"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

type rgb = color.RGBA

// pal is the palette everything is drawn in: Unit-01, the site's "eva"
// theme, unless the members' build picks another (members.go).
var pal = gen.Eva

// paintList draws the status bar, the list, the pane and the hint bar.
func (a *App) paintList(c *gfx.Canvas) {
	a.paintStatus(c)
	if a.LayoutTransitionRunning() {
		a.paintLayoutMotion(c)
	} else {
		a.paintRows(c)
		a.paintScrollbar(c)
		a.paintPane(c)
	}
	a.paintHint(c, a.listHint())
}

// listHint is the list legend: the buttons as they act right now. The last
// chunk reminds of the Select chords (quick.go) and the bar drops it first
// when the safe zone leaves no room; while Select is down the chords take
// the bar over (Left/Right turning the display last), and X is left out
// of them in the text layout, which has no picture for Art type to change.
func (a *App) listHint() string {
	var parts []string
	if len(a.view) > 0 {
		parts = append(parts, "A Details")
	}
	back := "B Options"
	if a.query != "" {
		back = "B Clear find"
	}
	parts = append(parts, back, "X Filters")
	if a.viewsOnCount() > 1 {
		parts = append(parts, "Y View")
	}
	parts = append(parts, "Select +")
	if a.quickHeld() {
		parts = nil
		if len(a.view) > 0 {
			parts = append(parts, "A Favorite")
		}
		if a.listShowsArt() {
			parts = append(parts, "X Shots")
		}
		parts = append(parts, "Y Layout", gfx.ArrowLeft+gfx.ArrowRight+" Rotate")
	}
	hint := strings.Join(parts, "  ")
	if !a.hintFits(hint) {
		hint = strings.NewReplacer("A Details", "A Open", "B Options", "B Opt.", "X Filters", "X Filt.", "A Favorite", "A Fav", "Y Layout", "Y Lay.", " Rotate", " Rot.").Replace(hint)
	}
	return hint
}

func (a *App) paintStatus(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Status, pal.Surface)
	c.HLine(l.Status.Min.X, l.Status.Max.X-1, l.Status.Max.Y-1, pal.Muted)
	if !a.cfg.ArcadeIntro && !a.cfg.ArcadeBack {
		a.paintHoldBar(c)
	}
	// the text's room: the bar, less the beta's BETA mark on the list
	st := l.Status
	if a.screen == ScreenList {
		st.Max.X = a.paintBetaMark(c)
	}
	y := st.Min.Y + 2
	if a.notice != "" {
		// A notice covers the bar until it expires or a press changes what
		// the bar says under it (notice.go): typing a search brings the
		// query and its count back at once, even over a jump's month.
		c.Text(st.Min.X+2, y, a.sm, gfx.Fit(a.notice, a.sm.Cols(st.Dx()-4)), pal.Fg)
		return
	}
	s := a.statusBar()
	if s.query != "" {
		count := itoa(s.shown) + " matches"
		c.TextRight(st.Max.X-2, y, a.sm, count, pal.Muted)
		cols := a.sm.Cols(st.Dx()-8-a.sm.Width(count)) - len("Find: ") - 1
		query := s.query
		if len(query) > cols {
			query = query[len(query)-max(0, cols):]
		}
		c.Text(st.Min.X+2, y, a.sm, "Find: "+query+"_", pal.Accent)
		return
	}
	left := s.view
	if s.update {
		c.Text(st.Min.X+2, y, a.sm, left, pal.Accent)
		c.TextRight(st.Max.X-2, y, a.sm, "App update", pal.Accent)
		return
	}
	count := itoa(s.total) + " releases"
	if s.narrow {
		count = itoa(s.shown) + " of " + itoa(s.total) + " releases"
	}
	c.Text(st.Min.X+2, y, a.sm, left, pal.Accent)
	x := st.Min.X + 2 + a.sm.Width(left) + a.sm.W*2
	if a.sm.Width(count) > st.Max.X-2-x {
		count = itoa(s.total)
		if s.narrow {
			count = itoa(s.shown) + "/" + count
		}
	}
	count = gfx.Fit(count, a.sm.Cols(st.Max.X-2-x))
	c.Text(x, y, a.sm, count, pal.Muted)
	left += "  " + count
	if s.net != "" {
		c.TextRight(st.Max.X-2, y, a.sm, gfx.Fit(s.net, a.sm.Cols(st.Dx()-a.sm.Width(left)-8)), pal.Fg)
	}
}

// statusBar is what the list's status bar says when no notice covers it:
// a search and its match count; or the view's name with the release count
// and the connection text; or, with an app update waiting, the view's
// short name and App update. It holds only what shows, fitted to the bar
// by paintStatus, so a notice can tell when that changes (notice.go).
type statusBar struct {
	query  string // the search, "" when none
	view   string // the view's name
	shown  int    // a search's matches, or the rows a narrowed view shows
	total  int    // the releases
	narrow bool   // the count reads shown of total
	update bool   // App update on the right
	net    string // the connection text on the right
}

// statusViews name the views in the status bar; the build date view is
// the one missing. statusViewsShort are the names beside App update:
// they leave it room even on narrow tate screens.
var (
	statusViews      = map[data.SortMode]string{data.SortDebut: "by: MiSTer debut", data.SortYear: "by: original year", data.SortAlphabetical: "by: A-Z", data.SortMaker: "by: manufacturer", data.SortCore: "by: core", data.SortFavorites: "Favorites A-Z", data.SortRecents: "Recents"}
	statusViewsShort = map[data.SortMode]string{data.SortUpdated: "Build date", data.SortDebut: "MiSTer debut", data.SortYear: "Original year", data.SortAlphabetical: "A-Z", data.SortMaker: "Manufacturer A-Z", data.SortCore: "Core A-Z", data.SortFavorites: "Favorites A-Z", data.SortRecents: "Recents"}
)

func (a *App) statusBar() statusBar {
	if a.query != "" {
		return statusBar{query: a.query, shown: len(a.view)}
	}
	if a.appUpdate != "" {
		return statusBar{view: statusViewsShort[a.mode], update: true}
	}
	s := statusBar{view: "by: build date", total: a.total, net: a.net}
	if v, ok := statusViews[a.mode]; ok {
		s.view = v
	}
	if a.filtersActive() || a.mode == data.SortFavorites || a.mode == data.SortRecents {
		s.shown, s.narrow = len(a.view), true
	}
	return s
}

// paintHint draws the bottom hint bar. Chunks are separated by two spaces;
// the first word of each chunk is the button and paints in the accent.
func (a *App) paintHint(c *gfx.Canvas, s string) {
	l := &a.lay
	c.Fill(l.Hint, pal.Surface)
	a.hintLine(c, l.Hint.Min.X+2, l.Hint.Min.Y+2, l.Hint.Dx()-4, s)
}

// hintChunk is one legend entry: the control, painted in the accent, and
// what it does.
type hintChunk struct{ btn, rest string }

// hintChunks splits a legend line at its double spaces. "Hold" belongs to
// the button after it, and arrows right after the button belong to it: a
// pair of arrows is one button, and "A < > open/close" names two controls
// for one action.
func hintChunks(s string) []hintChunk {
	var out []hintChunk
	for _, chunk := range strings.Split(s, "  ") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		btn, rest, _ := strings.Cut(chunk, " ")
		if btn == "Hold" {
			button, tail, _ := strings.Cut(rest, " ")
			btn, rest = btn+" "+button, tail
		}
		for {
			b2, r2, ok := strings.Cut(rest, " ")
			if !ok || !isArrow(b2) || !(isArrow(btn[len(btn)-1:]) || btn == "A") {
				break
			}
			btn, rest = btn+" "+b2, r2
		}
		out = append(out, hintChunk{btn, rest})
	}
	return out
}

// hintWidth is the room a legend line takes with the given gap between
// its chunks.
func (a *App) hintWidth(chunks []hintChunk, gap string) int {
	w := 0
	for i, ch := range chunks {
		if i > 0 {
			w += a.sm.Width(gap)
		}
		w += a.sm.Width(strings.TrimSpace(ch.btn + " " + ch.rest))
	}
	return w
}

// hintFits reports whether the legend line fits the hint bar, with its
// chunks two spaces apart or, failing that, one.
func (a *App) hintFits(s string) bool {
	return a.hintWidth(hintChunks(s), " ") <= a.lay.Hint.Dx()-4
}

// hintLine paints a legend line. Chunks sit two spaces apart, one when
// the line would not fit otherwise; a chunk that still does not fit, and
// every chunk after it, is left off.
func (a *App) hintLine(c *gfx.Canvas, x, y, w int, s string) {
	chunks := hintChunks(s)
	gap := "  "
	if a.hintWidth(chunks, gap) > w {
		gap = " "
	}
	maxX := x + w
	for i, ch := range chunks {
		text := strings.TrimSpace(ch.btn + " " + ch.rest)
		if i > 0 {
			x += a.sm.Width(gap)
		}
		if x+a.sm.Width(text) > maxX {
			break
		}
		x += c.Text(x, y, a.sm, a.btn(ch.btn), pal.Accent)
		if ch.rest != "" {
			x += c.Text(x, y, a.sm, " "+ch.rest, pal.Muted)
		}
	}
}

func isArrow(s string) bool {
	return s == gfx.ArrowUp || s == gfx.ArrowDown || s == gfx.ArrowLeft || s == gfx.ArrowRight
}

func (a *App) emptyListMessage() string {
	if a.mode == data.SortFavorites && len(a.cfg.Favorites) == 0 {
		return "No favorites yet"
	}
	if a.mode == data.SortRecents && len(a.cfg.RecentLaunches) == 0 {
		return "No launches yet"
	}
	if a.query != "" {
		return "no matches, " + a.btn("B") + ": clear find"
	}
	if a.filters.Since {
		if a.seen == nil || a.seen.BaseRows == nil {
			return "no previous visit yet"
		}
		other := a.filters
		other.Since = false
		if !other.Active() {
			return "No changed matches"
		}
	}
	return "no rows match, " + a.btn("X") + ": filters"
}

// paintRows draws the visible list lines: rows and the marker lines
// between them (the since-visit status row, the maker headers).
func (a *App) paintRows(c *gfx.Canvas) {
	if a.listMotion.offset == 0 {
		a.paintRowsStill(c)
		return
	}
	l := &a.lay
	c.Fill(l.List, pal.Bg)
	if len(a.view) == 0 {
		c.Text(l.List.Min.X+a.body.W, l.List.Min.Y+l.Line, a.body, gfx.Fit(a.emptyListMessage(), l.Cols-1), pal.Muted)
		return
	}
	// Clip partial rows to the list, keeping the snapped selection background
	// independent of the travelling text.
	parent := c
	c = &gfx.Canvas{RGBA: c.Sub(l.List)}
	defer parent.Dirty(l.List)
	c.Fill(l.lineRect(a.screenLine(a.cursor)-a.top), pal.Surface)
	offset := a.listMotion.offset
	first := a.top - (offset+l.Line-1)/l.Line
	if offset < 0 {
		first = a.top + (-offset)/l.Line
	}
	first = max(0, first)
	pos, mk := 0, 0
	for pos < len(a.view) && a.screenLine(pos) < first {
		pos++
	}
	for mk < len(a.marks) && (a.marks[mk] < pos || a.markLine(mk) < first) {
		mk++
	}
	for line := first; ; line++ {
		r := l.lineRect(line - a.top).Add(image.Pt(0, offset))
		if r.Min.Y >= l.List.Max.Y {
			break
		}
		if mk < len(a.marks) && a.marks[mk] == pos {
			a.paintMarker(c, r, a.markText(mk), a.markNote(mk))
			mk++
			continue
		}
		if pos >= len(a.view) {
			break
		}
		a.paintRow(c, r, pos)
		pos++
	}
	// a maker whose header scrolled off keeps its name on the top line
	if pos := a.pinnedPos(); pos >= 0 {
		r := l.lineRect(0)
		c.Fill(r, pal.Bg)
		a.paintMarker(c, r, a.groupLabel(a.view[pos]), a.headerNote(pos))
	}
}

func (a *App) paintRowsStill(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.List, pal.Bg)
	if len(a.view) == 0 {
		c.Text(l.List.Min.X+a.body.W, l.List.Min.Y+l.Line, a.body, gfx.Fit(a.emptyListMessage(), l.Cols-1), pal.Muted)
		return
	}
	// the first row and the first marker at or below the top line
	pos, mk := 0, 0
	for pos < len(a.view) && a.screenLine(pos) < a.top {
		pos++
	}
	for mk < len(a.marks) && (a.marks[mk] < pos || a.markLine(mk) < a.top) {
		mk++
	}
	for n := 0; n < l.Lines; n++ {
		r := l.lineRect(n)
		if mk < len(a.marks) && a.marks[mk] == pos {
			a.paintMarker(c, r, a.markText(mk), a.markNote(mk))
			mk++
			continue
		}
		if pos >= len(a.view) {
			break
		}
		a.paintRow(c, r, pos)
		pos++
	}
	// a maker whose header scrolled off keeps its name on the top line
	if pos := a.pinnedPos(); pos >= 0 {
		r := l.lineRect(0)
		c.Fill(r, pal.Bg)
		a.paintMarker(c, r, a.groupLabel(a.view[pos]), a.headerNote(pos))
	}
}

// paintScrollbar draws the track beside the list with a thumb sized to the
// visible share of the lines and placed at the scroll position.
func (a *App) paintScrollbar(c *gfx.Canvas) {
	l := &a.lay
	t := l.Scroll
	if t.Empty() {
		return
	}
	total := a.totalLines()
	if total <= l.Lines {
		return
	}
	h := t.Dy() * l.Lines / total
	if h < 4 {
		h = 4
	}
	// A group jump can put the last group at the top with blank rows below.
	// Its thumb still stops at the end of the track.
	y := t.Min.Y + (t.Dy()-h)*min(a.top, total-l.Lines)/(total-l.Lines)
	c.Fill(image.Rect(t.Min.X, y, t.Max.X, y+h), pal.Accent)
}

// paintMarker draws a marker line: text on the rule near its left end
// and, when there is one, note flush with its right end, where the rows
// keep their dates.
func (a *App) paintMarker(c *gfx.Canvas, r image.Rectangle, text, note string) {
	mid := r.Min.Y + r.Dy()/2
	c.HLine(r.Min.X, r.Max.X-1, mid, pal.Line)
	// fitted to the line at rest (restLayout); the list's edge clips it
	w := r.Dx() + a.restLayout().List.Dx() - a.lay.List.Dx()
	cols := a.sm.Cols(w) - 2
	if note != "" {
		note = " (" + note + ")"
		// Keep the note beside the name, shortening only the name to fit.
		cols = min(cols, (w-a.body.W)/a.sm.W-3-len(note))
	}
	s := " " + gfx.Fit(text, cols) + note + " "
	x := r.Min.X + a.body.W
	c.Fill(image.Rect(x, r.Min.Y, x+a.sm.Width(s), r.Max.Y), pal.Bg)
	c.Text(x, r.Min.Y+2, a.sm, s, pal.Muted)
}

func (a *App) paintRow(c *gfx.Canvas, r image.Rectangle, pos int) {
	l := &a.lay
	i := a.view[pos]
	row := &a.ds.Rows[i]
	d := &a.ds.Der[i]
	selected := pos == a.cursor
	if selected && a.listMotion.offset == 0 {
		c.Fill(r, pal.Surface)
	}
	x := r.Min.X
	y := r.Min.Y
	fw := a.body.W
	// fav
	if a.cfg.Favorites[row.K] {
		c.Text(x, y, a.body, "*", pal.Accent)
	}
	x += fw
	// title
	st := a.status(i)
	titleCol := pal.Fg
	if selected {
		titleCol = pal.Accent
	} else if st == data.StatusNotFound {
		titleCol = pal.Muted
	}
	// the status glyph and the date use the narrow font at the titles'
	// height, whose baseline sits one pixel below the body font's
	rf := a.rowFont()
	tw := l.TitleW
	if a.mode == data.SortYear && !l.PictureColumns {
		// the year is on the header line, so the title takes the date column
		tw += a.dateCols() * rf.W
	}
	if fit := tw + a.restLayout().TitleW - l.TitleW; fit != tw {
		// a framed layout motion: the title as it reads at rest, cut by
		// its column's moving edge
		col := &gfx.Canvas{RGBA: c.Sub(image.Rect(x, c.Rect.Min.Y, x+tw, c.Rect.Max.Y))}
		a.paintTitle(col, x, y, fit, d.Title, row.Beta, titleCol)
	} else {
		a.paintTitle(c, x, y, tw, d.Title, row.Beta, titleCol)
	}
	if l.PictureColumns {
		return
	}
	x += tw + rf.W
	g, gc := statusGlyph(st)
	// a ROM problem with the version Start launches outranks the build
	// status: the game will not start, whatever the core's date
	if m, mc, ok := a.romMark(i); ok {
		g, gc = m, mc
	}
	c.Text(x, y-1, rf, g, gc)
	if a.mode == data.SortYear {
		return
	}
	// date, right-aligned in its column
	date := row.Updated
	if a.mode == data.SortDebut {
		date = row.Date
	} else if a.mode == data.SortRecents {
		date = a.launchedAt(row.K) // when it was launched last
	}
	// a row added or rebuilt since the last visit shows its date in the
	// accent under every order: the app's form of the site's per-row dot
	dateCol := pal.Muted
	if a.seen != nil && a.seen.Unseen(row) {
		dateCol = pal.Accent
	}
	text := a.dateCol(date)
	if a.mode == data.SortMaker || a.mode == data.SortCore {
		// the original release year as the catalogue has it, "198?" included
		// (a core's games mostly share one build date, which would only
		// repeat down its group)
		text = gfx.Fit(strings.TrimSpace(row.Year), a.dateCols())
		for len(text) < a.dateCols() {
			text = " " + text
		}
	}
	c.Text(r.Max.X-a.dateCols()*rf.W, y-1, rf, text, dateCol)
}

// paintTitle draws a list title into w pixels, in the narrow font when
// chosen, with a beta sign after a Patreon beta core's name.
func (a *App) paintTitle(c *gfx.Canvas, x, y, w int, title string, beta bool, col rgb) {
	if f := a.titleFont(); f != nil {
		y-- // scientifica's baseline is one pixel lower than the body font's
		if beta {
			w -= f.Advance(' ') + f.Advance(gfx.Beta[0])
		}
		tw := c.TextProp(x, y, f, gfx.FitProp(f, title, w), col)
		if beta {
			c.TextProp(x+tw+f.Advance(' '), y, f, gfx.Beta, pal.Warn)
		}
		return
	}
	cols := a.body.Cols(w)
	if beta {
		cols -= 2
	}
	s := gfx.Fit(title, cols)
	c.Text(x, y, a.body, s, col)
	if beta {
		c.Text(x+a.body.Width(s)+a.body.W, y, a.body, gfx.Beta, pal.Warn)
	}
}

// paintPane draws the highlighted row's thumbnail and specs.
func (a *App) paintPane(c *gfx.Canvas) {
	l := &a.lay
	c.Fill(l.Pane, pal.Bg)
	dividerForLayout(*l).paint(c, l.Body)
	a.paintedThumb, a.paintedPaneText = image.Rectangle{}, image.Rectangle{}
	if l.Pane.Empty() {
		a.paintedThumbImage = nil // the text layout: rows only
		return
	}
	row, d, i := a.current()
	if row == nil {
		return
	}
	drawn := a.paintThumb(c, l.Thumb, row)
	text := l.PaneText
	if l.TextBeside {
		// beside the picture when at least 60 px remain there, else below
		if beside := l.Pane.Max.X - (drawn.Max.X + 4); beside >= 60 {
			text = image.Rect(drawn.Max.X+4, l.Pane.Min.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		} else {
			text = image.Rect(l.Thumb.Min.X, l.Thumb.Max.Y+3, l.Pane.Max.X, l.Pane.Max.Y)
		}
	}
	a.paintedThumb, a.paintedPaneText = drawn, text
	lines := a.paneLines(row, d, i, a.sm.Cols(text.Dx()))
	if l.PictureColumns {
		// The fixed right column is narrow but tall: wrap information rather
		// than losing it to the ellipsis used by the smaller classic panes.
		var wrapped []paneLine
		for _, line := range lines {
			for _, part := range gfx.Wrap(line.text, a.sm.Cols(text.Dx()), 99) {
				wrapped = append(wrapped, paneLine{part, line.col})
			}
		}
		lines = wrapped
	}
	a.paintPaneText(c, text, lines)
}

// paneLines is what the pane says about a row, in lines of at most cols
// characters: the title over up to two lines, the card answer, the kind,
// the core, the year and maker, the arcade rotation, players and
// controls, and the badges.
func (a *App) paneLines(row *data.Row, d *data.Derived, i, cols int) []paneLine {
	var lines []paneLine
	hue := typeHue(row.Base)
	for _, t := range gfx.Wrap(d.Title, cols, 2) {
		lines = append(lines, paneLine{t, pal.Fg})
	}
	// the card answer comes right after the title: it is the point of the app
	if a.cfg.Status != nil {
		st := a.status(i)
		_, sc := statusGlyph(st)
		lines = append(lines, paneLine{statusText(st, ""), sc})
	}
	kind := d.TypeLabel
	if row.IsArcade() {
		kind = "Arcade / " + d.SrcShort
	}
	lines = append(lines, paneLine{kind, hue})
	if row.Core != "" && d.CoreLabel != row.Title {
		lines = append(lines, paneLine{d.CoreLabel, pal.Muted})
	}
	if row.Year != "" || row.Manufacturer != "" {
		lines = append(lines, paneLine{strings.TrimSpace(row.Year + " " + data.ASCII(row.Manufacturer)), pal.Fg})
	}
	if row.IsArcade() {
		spec := strings.TrimSpace(strings.Join(nonEmpty(rotShort(row.Rot), plrShort(row.Plr)), " / "))
		if spec != "" {
			lines = append(lines, paneLine{spec, pal.Fg})
		}
		if d.Ctl != "" {
			lines = append(lines, paneLine{d.Ctl, pal.Fg})
		}
	}
	if ch := chips(row, d); len(ch) > 0 {
		col := pal.Muted
		for _, x := range ch {
			if x == "beta" || strings.HasPrefix(x, "boots") {
				col = pal.Warn
			}
		}
		lines = append(lines, paneLine{strings.Join(ch, ", "), col})
	}
	return lines
}

type paneLine struct {
	text string
	col  rgb
}

func (a *App) paintPaneText(c *gfx.Canvas, r image.Rectangle, lines []paneLine) {
	lh := a.sm.H + 1
	cols := a.sm.Cols(r.Dx())
	y := r.Min.Y
	for _, ln := range lines {
		if y+a.sm.H > r.Max.Y {
			return
		}
		c.Text(r.Min.X, y, a.sm, gfx.Fit(ln.text, cols), ln.col)
		y += lh
	}
}

// paintThumb draws a row's list thumbnail or a placeholder into box and
// returns the rectangle it covered.
func (a *App) paintThumb(c *gfx.Canvas, box image.Rectangle, row *data.Row) image.Rectangle {
	a.paintedThumbImage = nil
	key, slot := thumbSlot(row, a.ListShot())
	if key == "" {
		a.placeholder(c, box, "no shot")
		return box
	}
	// a horizontal game fills the 4:3 box like on the site; a vertical one
	// keeps its shape and sits on the left edge with the text
	req := ImageReq{Key: key, Slot: slot, W: box.Dx(), H: box.Dy(), Stretch: slot != "system" && row.ImgW > row.ImgH}
	img, st := a.cfg.Images.Get(req)
	if img == nil {
		a.want(req)
		switch st {
		case ImageLoading:
			a.placeholder(c, box, "loading")
		case ImageOffline:
			a.placeholder(c, box, "no connection")
		default:
			a.placeholder(c, box, "no shot")
		}
		return box
	}
	a.paintedThumbImage = img
	// horizontal: on the left edge, in line with the text below; the
	// classic tate pane centres it in the box beside the text; the wider
	// layouts keep it left so the text can sit beside it, unless the text
	// keeps a column of its own
	p := image.Pt(box.Min.X, box.Min.Y+(box.Dy()-img.Rect.Dy())/2)
	if (a.lay.Portrait && !a.lay.TextBeside) || a.lay.PictureColumns || a.lay.ArtCentered {
		p.X = box.Min.X + (box.Dx()-img.Rect.Dx())/2
	}
	if slot == "system" {
		c.BlitAlpha(p, img)
	} else {
		c.Blit(p, img)
	}
	return image.Rectangle{Min: p, Max: p.Add(img.Rect.Size())}
}

// placeholder stands in for a picture: a solid black shape with a word on
// it (pictures themselves draw bare, no frame).
func (a *App) placeholder(c *gfx.Canvas, box image.Rectangle, text string) {
	c.Fill(box, rgb{R: 0, G: 0, B: 0, A: 255})
	w := a.sm.Width(text)
	c.Text(box.Min.X+(box.Dx()-w)/2, box.Min.Y+(box.Dy()-a.sm.H)/2, a.sm, text, pal.Muted)
}

func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// availableArrows names only directions that can change the current position.
func availableArrows(back, forward bool, backLabel, forwardLabel string) string {
	var arrows []string
	if back {
		arrows = append(arrows, backLabel)
	}
	if forward {
		arrows = append(arrows, forwardLabel)
	}
	return strings.Join(arrows, " ")
}
