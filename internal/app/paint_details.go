package app

import (
	"image"
	"path"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// The version line scrolls a label that does not fit: a short pause at
// each end, then a steady pixel-by-pixel pass to the other end and back.
const (
	marqueeFrame = time.Second / 30
	marqueeSpeed = 40 // pixels per second
	marqueePause = 500 * time.Millisecond
)

type marqueeState struct {
	key    string    // the row and version scrolling, "" when idle
	start  time.Time // when the scroll began
	next   time.Time // the next frame
	span   int       // pixels the label overflows the line by
	offset int       // offset in the last painted frame
}

// marqueeOffset is how far the label has scrolled left at elapsed.
func marqueeOffset(elapsed time.Duration, span int) int {
	if span <= 0 {
		return 0
	}
	travel := time.Duration(span) * time.Second / marqueeSpeed
	t := elapsed % (2 * (marqueePause + travel))
	switch {
	case t < marqueePause:
		return 0
	case t < marqueePause+travel:
		return int((t - marqueePause) * marqueeSpeed / time.Second)
	case t < 2*marqueePause+travel:
		return span
	}
	return span - int((t-2*marqueePause-travel)*marqueeSpeed/time.Second)
}

// runMarquee starts or continues the scroll of key, a label span pixels
// too wide, and returns its offset for this paint.
func (a *App) runMarquee(key string, span int) int {
	now := a.cfg.TimerNow()
	if a.marquee.key != key {
		a.marquee = marqueeState{key: key, start: now, next: now.Add(marqueeFrame)}
	}
	a.marquee.span = span
	a.marquee.offset = marqueeOffset(now.Sub(a.marquee.start), span)
	return a.marquee.offset
}

func (a *App) nextMarqueeTick() time.Time {
	if a.marquee.key == "" || a.screen != ScreenDetails || a.saver.active || a.PageTransitionRunning() {
		return time.Time{}
	}
	return a.marquee.next
}

func (a *App) tickMarquee(now time.Time) bool {
	next := a.nextMarqueeTick()
	if next.IsZero() || now.Before(next) {
		return false
	}
	a.marquee.next = now.Add(marqueeFrame)
	if marqueeOffset(now.Sub(a.marquee.start), a.marquee.span) == a.marquee.offset {
		return false
	}
	if a.detail.clip.Empty() {
		a.all = true
	} else {
		a.detail.versionDirty = true
	}
	return true
}

// The information slides at 360 pixels per second toward the wanted line.
// The timer fallback uses 60 Hz; the device follows its display refresh.
const (
	detailFrame = time.Second / 60
	detailStep  = 6
)

// detailLine is the height of one information line.
func (a *App) detailLine() int { return a.sm.H + 1 }

func (a *App) nextDetailTick() time.Time {
	if a.screen != ScreenDetails || a.saver.active || a.detail.pixel == a.detail.scroll*a.detailLine() {
		return time.Time{}
	}
	return a.detail.next
}

// DetailScrollRunning keeps the host in its display-paced loop after release.
func (a *App) DetailScrollRunning() bool { return !a.nextDetailTick().IsZero() }

// DetailScrollFrame advances at the display's cadence, without a second timer
// rejecting a refresh that arrives slightly before the nominal 60 Hz deadline.
func (a *App) DetailScrollFrame(now time.Time) bool {
	if !a.DetailScrollRunning() {
		return false
	}
	elapsed := now.Sub(a.detail.last)
	if elapsed <= 0 {
		return false
	}
	travel := elapsed*detailStep + a.detail.carry
	step := int(travel / detailFrame)
	a.detail.carry = travel % detailFrame
	a.detail.last = now
	a.detail.next = now.Add(detailFrame)
	d := a.detail.scroll*a.detailLine() - a.detail.pixel
	a.detail.pixel += max(-step, min(step, d))
	if step == 0 {
		return false
	}
	a.invalidateDetailScroll()
	return true
}

// Timer-driven callers (including the harness) use the nominal frame deadline.
func (a *App) tickDetailScroll(now time.Time) bool {
	next := a.nextDetailTick()
	if next.IsZero() || now.Before(next) {
		return false
	}
	return a.DetailScrollFrame(now)
}

// launchEntry is one thing the details view can launch.
type launchEntry struct {
	label string
	path  string // card-relative, or "core:NAME"
	ok    bool   // launchable: the file and its core are on the card
	why   string // the notice when not ok
}

// launchEntries lists the mainline file and installed alternatives, each
// marked with whether it can actually be launched. An MRA whose core is
// not installed is greyed like a missing file and is not launchable
// either: handed to MiSTer, it leaves the menu stuck on a load that never
// completes. Its alternatives share the core. Without an Exists hook a
// not-found status cannot tell the two apart and reads as the file missing.
func (a *App) launchEntries(row *data.Row, i int) []launchEntry {
	var out []launchEntry
	st := a.status(i)
	coreMissing := ""
	if row.MRA != "" && row.Core != "" && st == data.StatusNotFound && a.cfg.Status != nil && a.cfg.Exists != nil && a.cfg.Exists(row.MRA) {
		coreMissing = "the " + row.Core + " core is not on the card"
	}
	if row.MRA != "" {
		ok, why := true, ""
		if a.cfg.Exists != nil {
			ok = a.cfg.Exists(row.MRA)
		} else if a.cfg.Status != nil {
			ok = st != data.StatusNotFound
		}
		if !ok {
			why = "that file is not on the card"
		} else if coreMissing != "" {
			ok, why = false, coreMissing
		}
		out = append(out, launchEntry{path.Base(row.MRA), row.MRA, ok, why})
	} else if row.Core != "" {
		ok := a.cfg.Status == nil || st != data.StatusNotFound
		out = append(out, launchEntry{"load the " + row.Core + " core", "core:" + row.Core, ok, "the " + row.Core + " core is not on the card"})
	}
	if a.cfg.Alternatives != nil {
		for _, alt := range a.cfg.Alternatives(row) {
			out = append(out, launchEntry{"alt: " + path.Base(alt), alt, coreMissing == "", coreMissing})
		}
	}
	return out
}

// detailLines builds the text of the details view: the release decision
// first (when it shipped, is it on the card, how does it boot), then the
// rest of the site's panel fields.
func (a *App) detailLines(row *data.Row, d *data.Derived, i int) []paneLine {
	now := a.cfg.Now()
	rel := func(iso string) string {
		if !a.cfg.ClockTrusted {
			return ""
		}
		if s := data.RelAge(now, iso); s != "" {
			return "  (" + s + ")"
		}
		return ""
	}
	var L []paneLine
	add := func(label, val string, col rgb) {
		if val == "" {
			return
		}
		// labels padded so the values line up
		label += ":"
		for len(label) < 10 {
			label += " "
		}
		L = append(L, paneLine{label + data.ASCII(val), col})
	}
	fg, mu := gen.Eva.Fg, gen.Eva.Muted
	if a.cfg.ROMIssue != nil {
		entries := a.launchEntries(row, i)
		if len(entries) > 0 {
			e := entries[max(0, min(a.detail.pick, len(entries)-1))]
			if e.ok {
				if issue := a.cfg.ROMIssue(e.path, false); issue != "" {
					L = append(L, paneLine{issue, gen.Eva.Warn})
				}
			}
		}
	}
	prov := func(f string) rgb {
		if row.HasProv(f) {
			return mu
		}
		return fg
	}
	addSpec := func(label, field, val string) {
		if val != "" && row.HasProv(field) {
			val += " (provisional)"
		}
		add(label, val, prov(field))
	}
	add("Updated", row.Updated+rel(row.Updated), fg)
	kind := "Debut"
	if row.DateKind == "build" {
		kind = "Latest build"
	}
	add(kind, row.Date+rel(row.Date), fg)
	if a.cfg.Status != nil {
		st := a.status(i)
		_, sc := statusGlyph(st)
		L = append(L, paneLine{"Card:     " + statusText(st, ""), sc})
	}
	if row.IsLocal() {
		L = append(L, paneLine{"Not in the MisterZine catalogue; details read from the MRA file", mu})
		add("File", row.MRA, mu)
	}
	if row.Rot != "" {
		addSpec("Rotation", "rot", row.Rot)
		if row.Brot != "" {
			L = append(L, paneLine{"  boots " + row.Brot + ", no screen flip", gen.Eva.Warn})
		}
	}
	L = append(L, paneLine{"", fg})
	if !row.IsArcade() {
		add("Type", d.TypeLabel, typeHue(row.Base))
	}
	add("Genre", row.Genre, fg)
	add("Mfr", row.Manufacturer, fg)
	if row.Core != "" {
		add("Core", d.CoreLabel+" ("+row.Core+")", fg)
	}
	add("ROM", row.SN, fg)
	if row.Year != "" {
		y := row.Year
		if a.cfg.ClockTrusted {
			if age := data.YearAge(now, row.Year); age != "" {
				y += "  (" + age + ")"
			}
		}
		add("Year", y, fg)
	}
	add("Region", row.Reg, fg)
	if row.Scr == 2 {
		add("Screens", "dual screen", fg)
	} else if row.Scr == 3 {
		add("Screens", "triple screen", fg)
	}
	add("Video", row.Res, fg)
	addSpec("Players", "plr", row.Plr)
	addSpec("Controls", "ctl", d.Ctl)
	addSpec("Special", "spc", row.Spc)
	if row.HasProv("buttons") && !row.HasProv("ctl") && d.Buttons != "" {
		addSpec("Buttons", "buttons", d.Buttons)
	}
	add("Flip", row.Flip, fg)
	add("Commit", row.Act+rel(row.Act), mu)
	add("Source", data.SrcFull(row.Src), fg)
	if row.Src == "rmcores" {
		L = append(L, paneLine{"rm build: CRT Adjust and V-Size on the analog output and a pause overlay; the same game ships in the MiSTer Distribution without them", mu})
	}
	if row.Deprecated {
		add("Status", "deprecated", gen.Eva.Danger)
	}
	if row.Beta {
		L = append(L, paneLine{data.GateLine(row), mu})
	}
	add("Note", row.Note, fg)
	return L
}

func (a *App) paintDetails(c *gfx.Canvas) {
	l := &a.lay
	row, d, i := a.current()
	if row == nil {
		a.screen = ScreenList
		a.paintList(c)
		return
	}
	body := l.Body
	body.Min.Y = l.Root.Min.Y // no status bar here: the room goes to the specs
	if key, slot := cabShot(row); key != "" && a.transition.enabled {
		// the launch animation's monitor picture, decoded before Start
		a.want(ImageReq{Key: key, Slot: slot, W: cabTexW, H: cabTexH, Stretch: slot != "system" && row.ImgW > row.ImgH})
	}
	cols := a.body.Cols(body.Dx())
	y := body.Min.Y
	title := d.Title
	if a.cfg.Favorites[row.K] {
		title = "* " + title
	}
	for _, t := range gfx.Wrap(title, cols, 2) {
		c.Text(body.Min.X, y, a.body, t, gen.Eva.Accent)
		y += l.Line
	}
	if ch := chips(row, d); len(ch) > 0 {
		c.Text(body.Min.X, y, a.sm, gfx.Fit(strings.Join(ch, "  "), a.sm.Cols(body.Dx())), gen.Eva.Warn)
		y += a.sm.H + 2
	}
	y += 4 // breathing room before the pictures
	entries := a.launchEntries(row, i)
	lh := a.sm.H + 1
	launchH := lh
	// shots strip: in horizontal the pictures share the row; in tate they
	// are narrow, so each takes its own width and they pack from the left
	stripH := 54
	if !l.Portrait {
		stripH = 72
	}
	// Reserve the version selector and at least one information line.
	stripH = min(stripH, max(0, body.Max.Y-y-launchH-2-lh-4))
	slots := shotSlots(row)
	if len(slots) > 0 && stripH > 0 {
		bw := (body.Dx() - (len(slots)-1)*4) / len(slots)
		if bw > 96 {
			bw = 96
		}
		bh := stripH
		x := body.Min.X
		for _, s := range slots {
			// each box hugs its picture: horizontal games at 4:3 like the
			// site, vertical ones at their own shape; all packed from the left
			w, stretch := bw, false
			if s != "system" && row.ImgW > 0 && row.ImgH > 0 {
				if row.ImgW > row.ImgH {
					w, stretch = min(bw, bh*4/3), true
				} else if fw, _ := fitBox(row.ImgW, row.ImgH, bw, bh); fw > 0 {
					w = fw
				}
			}
			box := image.Rect(x, y, x+w, y+bh)
			key := row.Img
			if s == "system" {
				key = row.Core
			}
			a.paintImageBox(c, box, key, s, stretch)
			x += w + 4
		}
		y += bh + 4
	}
	// spec lines, scrollable
	sc := a.sm.Cols(body.Dx() - 4) // leave room for the information scrollbar
	lines := wrapDetailLines(a.detailLines(row, d, i), sc)
	avail := body.Max.Y - y
	specH := avail - launchH - 2
	maxLines := max(0, specH/lh)
	a.detail.lines = maxLines
	if a.detail.scroll > len(lines)-maxLines {
		a.detail.scroll = len(lines) - maxLines
	}
	if a.detail.scroll < 0 {
		a.detail.scroll = 0
	}
	a.detail.clip = image.Rect(body.Min.X, y, body.Max.X, y+maxLines*lh)
	a.detail.text, a.detail.cols, a.detail.entries = lines, sc, len(entries)
	a.paintDetailInfo(c)
	a.detail.versions, a.detail.versionY = entries, body.Max.Y-launchH
	a.paintDetailVersion(c)
	if a.notice != "" {
		c.Fill(l.Hint, gen.Eva.Surface)
		c.Text(l.Hint.Min.X+2, l.Hint.Min.Y+2, a.sm, gfx.Fit(a.notice, a.sm.Cols(l.Hint.Dx()-4)), gen.Eva.Fg)
		return
	}
	a.paintHint(c, a.detailsHint(len(lines) > maxLines, len(entries), max(0, len(lines)-maxLines)))
}

// The version marquee has its own small dirty region; it must not rebuild
// the information or query installed alternatives on every animation tick.
func (a *App) paintDetailVersion(c *gfx.Canvas) {
	body, y, entries := a.lay.Body, a.detail.versionY, a.detail.versions
	row, _, _ := a.current()
	if row == nil {
		return
	}
	c.Fill(image.Rect(body.Min.X, y-1, body.Max.X, y+a.detailLine()), gen.Eva.Bg)
	c.HLine(body.Min.X, body.Max.X-1, y-1, gen.Eva.Line)
	if a.detail.pick >= len(entries) {
		a.detail.pick = len(entries) - 1
	}
	if a.detail.pick < 0 {
		a.detail.pick = 0
	}
	if len(entries) == 0 {
		c.Text(body.Min.X, y, a.sm, "No version available", gen.Eva.Muted)
	} else {
		e := entries[a.detail.pick]
		count := itoa(a.detail.pick+1) + "/" + itoa(len(entries))
		c.TextRight(body.Max.X, y, a.sm, count, gen.Eva.Muted)
		prefix := "Version: "
		if len(entries) > 1 {
			prefix = gfx.ArrowLeft + gfx.ArrowRight + " "
		}
		col := gen.Eva.Accent
		label := e.label
		if !e.ok {
			prefix = "- " + prefix
			col = gen.Eva.Muted
		}
		labelW := body.Dx() - a.sm.Width(count) - a.sm.W
		text := prefix + label
		if over := a.sm.Width(text) - labelW; over > 0 {
			// too long for the line: scroll it back and forth by the pixel
			off := a.runMarquee(row.K+"#"+itoa(a.detail.pick), over)
			c.TextClip(body.Min.X-off, y, a.sm, text, col, image.Rect(body.Min.X, y, body.Min.X+labelW, y+a.sm.H))
		} else {
			a.marquee = marqueeState{}
			c.Text(body.Min.X, y, a.sm, text, col)
		}
	}
}

// Scrolling only changes the information viewport and its navigation hint.
// Full invalidations still rebuild the cache when images, metadata or layout change.
func (a *App) invalidateDetailScroll() {
	if a.detail.clip.Empty() || a.PageTransitionRunning() || a.saver.active {
		a.all = true
	} else {
		a.detail.dirty = true
	}
}

func (a *App) paintDetailInfo(c *gfx.Canvas) {
	clip, lines, sc, lh := a.detail.clip, a.detail.text, a.detail.cols, a.detailLine()
	maxLines := a.detail.lines
	off := max(0, min(a.detail.pixel, max(0, len(lines)-maxLines)*lh))
	a.detail.pixel = off
	c.Fill(clip, gen.Eva.Bg)
	for n := off / lh; n < len(lines) && n*lh-off < clip.Dy(); n++ {
		c.TextClip(clip.Min.X, clip.Min.Y+n*lh-off, a.sm, gfx.Fit(lines[n].text, sc), lines[n].col, clip)
	}
	if len(lines) > maxLines && maxLines > 0 {
		thumbH := max(2, clip.Dy()*maxLines/len(lines))
		thumbY := clip.Min.Y + (clip.Dy()-thumbH)*off/((len(lines)-maxLines)*lh)
		c.Fill(image.Rect(clip.Max.X-2, thumbY, clip.Max.X, thumbY+thumbH), gen.Eva.Accent)
	}
	if a.notice == "" {
		a.paintHint(c, a.detailsHint(len(lines) > maxLines, a.detail.entries, max(0, len(lines)-maxLines)))
	}
}

// detailsHint is the Details legend. The Left/Right hint appears only when
// the game has more than one version to choose from, and the Up/Down hint
// only while the information actually scrolls.
func (a *App) detailsHint(scrolls bool, versions int, scrollLimit int) string {
	hint := "Start Launch  A Shots  Y Fav"
	short := "Start Go  A Art  Y Fav"
	if versions > 1 {
		arrows := availableArrows(a.detail.pick > 0, a.detail.pick < versions-1, gfx.ArrowLeft, gfx.ArrowRight)
		hint += "  " + arrows + " Version"
		short += "  " + strings.ReplaceAll(arrows, " ", "") + " Alt"
	}
	if scrolls {
		down := a.detail.scroll < scrollLimit
		arrows := availableArrows(a.detail.scroll > 0, down, gfx.ArrowUp, gfx.ArrowDown)
		hint += "  " + arrows + " Info"
		short += "  " + strings.ReplaceAll(arrows, " ", "") + " Info"
	}
	if a.sm.Width(hint) > a.lay.Hint.Dx()-4 {
		return short
	}
	return hint
}

// Preserve aligned labels on the first line and wrap continuations underneath.
func wrapDetailLines(lines []paneLine, cols int) []paneLine {
	var out []paneLine
	for _, line := range lines {
		s := line.text
		for len(s) > cols && cols > 0 {
			cut := cols
			if space := strings.LastIndexByte(s[:cols], ' '); space > cols/2 {
				cut = space
			}
			out = append(out, paneLine{strings.TrimRight(s[:cut], " "), line.col})
			s = strings.TrimLeft(s[cut:], " ")
			if cols > 2 {
				s = "  " + s
			}
		}
		out = append(out, paneLine{s, line.col})
	}
	return out
}

// shotSlots is the list of picture slots to show for a row.
func shotSlots(row *data.Row) []string {
	if row.Img != "" && len(row.ImgSlots) > 0 {
		return row.ImgSlots
	}
	if row.Core != "" && !row.IsArcade() {
		return []string{"system"}
	}
	return nil
}

func (a *App) paintImageBox(c *gfx.Canvas, box image.Rectangle, key, slot string, stretch bool) {
	req := ImageReq{Key: key, Slot: slot, W: box.Dx(), H: box.Dy(), Stretch: stretch}
	img, st := a.cfg.Images.Get(req)
	if img == nil {
		a.want(req)
		text := "no shot"
		if st == ImageLoading {
			text = "loading"
		} else if st == ImageOffline {
			text = "no connection"
		}
		a.placeholder(c, box, text)
		return
	}
	p := image.Pt(box.Min.X, box.Min.Y+(box.Dy()-img.Rect.Dy())/2)
	if slot == "system" {
		c.BlitAlpha(p, img)
	} else {
		c.Blit(p, img)
	}
}

const FavoritesUnavailableNotice = "Favorites unreadable; file kept"

func (a *App) actDetails(k platform.Key) bool {
	row, _, i := a.current()
	if row == nil {
		a.screen = ScreenList
		a.all = true
		return true
	}
	switch k {
	case platform.KeyBack:
		a.screen = ScreenList
	case platform.KeyEnter:
		a.screen = ScreenShot
		a.slot = 0
	case platform.KeyLeft, platform.KeyRight:
		entries := a.launchEntries(row, i)
		prev := a.detail.pick
		if k == platform.KeyLeft {
			a.detail.pick = max(0, a.detail.pick-1)
		} else {
			a.detail.pick = min(a.detail.pick+1, max(0, len(entries)-1))
		}
		if a.detail.pick != prev { // only a change of choice is recorded
			a.rememberPick(row, entries, a.detail.pick)
		}
	case platform.KeyUp, platform.KeyDown:
		// a page with one line of overlap; the slide starts at the next tick
		page := max(1, a.detail.lines-1)
		if k == platform.KeyUp {
			page = -page
		}
		running := a.DetailScrollRunning()
		a.detail.scroll = max(0, a.detail.scroll+page)
		if !a.detail.clip.Empty() {
			a.detail.scroll = min(a.detail.scroll, max(0, len(a.detail.text)-a.detail.lines))
		}
		if !running {
			a.detail.next = a.cfg.TimerNow()
			a.detail.last = a.detail.next
			a.detail.carry = 0
		}
		a.invalidateDetailScroll()
		return true
	case platform.KeySpace:
		if a.toggleFavorite() {
			a.screen = ScreenList // the row left the filtered view
		}
	case platform.KeyStart:
		return a.launchPick(a.detail.pick)
	default:
		return false
	}
	a.all = true
	return true
}

// launchPick launches entry pick of the current row's versions (0 = the
// main one); a version that is not on the card only shows a notice.
func (a *App) launchPick(pick int) bool {
	row, _, i := a.current()
	if row == nil {
		return false
	}
	return a.launchRow(row, i, pick)
}

// launchRow launches entry pick of row i's versions; true when the
// version is not on the card and a notice says so instead.
func (a *App) launchRow(row *data.Row, i, pick int) bool {
	entries := a.launchEntries(row, i)
	if len(entries) > 0 && a.cfg.Launch != nil {
		pick = max(0, min(pick, len(entries)-1))
		e := entries[pick]
		if !e.ok {
			a.Notice(e.why, 3*time.Second)
			return true
		}
		if a.cfg.ROMIssue != nil {
			if issue := a.cfg.ROMIssue(e.path, true); issue != "" {
				a.Notice(issue, 6*time.Second)
				return true
			}
		}
		if pick > 0 {
			// An alternative launched is the choice from now on. Launching
			// the main version keeps any record: the alternatives may just
			// not be scanned yet.
			a.rememberPick(row, entries, pick)
		}
		a.recordLaunch(row)
		if a.LaunchTransition() == "hold" && a.transition.enabled {
			// the animation waits for Start to stay down; a tap launches
			// on the release
			a.holdLaunch = holdLaunch{row: row, path: e.path, at: a.cfg.TimerNow()}
			return false
		}
		a.startLaunchCab(row, e.path)
	}
	return false
}
