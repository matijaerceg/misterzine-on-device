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

// launchEntry is one thing the details view can launch.
type launchEntry struct {
	label string
	path  string // card-relative, or "core:NAME"
	ok    bool   // present on the card
}

// launchEntries lists the mainline file and installed alternatives, each
// marked with whether it is actually on the card.
func (a *App) launchEntries(row *data.Row, i int) []launchEntry {
	var out []launchEntry
	st := a.status(i)
	if row.MRA != "" {
		ok := true
		if a.cfg.Exists != nil {
			ok = a.cfg.Exists(row.MRA)
		} else if a.cfg.Status != nil {
			ok = st != data.StatusNotFound
		}
		out = append(out, launchEntry{path.Base(row.MRA), row.MRA, ok})
	} else if row.Core != "" {
		ok := a.cfg.Status == nil || st != data.StatusNotFound
		out = append(out, launchEntry{"load the " + row.Core + " core", "core:" + row.Core, ok})
	}
	if a.cfg.Alternatives != nil {
		for _, alt := range a.cfg.Alternatives(row) {
			out = append(out, launchEntry{"alt: " + path.Base(alt), alt, true})
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
		L = append(L, paneLine{label + ": " + data.ASCII(val), col})
	}
	fg, mu := gen.Eva.Fg, gen.Eva.Muted
	prov := func(f string) rgb {
		if row.HasProv(f) {
			return mu
		}
		return fg
	}
	add("Updated", row.Updated+rel(row.Updated), fg)
	if d.BatchN >= 2 {
		L = append(L, paneLine{"  shipped together with " + itoa(d.BatchN-1) + " other game(s) on this core", mu})
	}
	kind := "Debut"
	if row.DateKind == "build" {
		kind = "Latest build"
	}
	add(kind, row.Date+rel(row.Date), fg)
	if a.cfg.Status != nil {
		st := a.status(i)
		_, sc := statusGlyph(st)
		L = append(L, paneLine{"Card: " + statusText(st, ""), sc})
	}
	if row.Rot != "" {
		add("Rotation", row.Rot, prov("rot"))
		if row.Brot != "" {
			L = append(L, paneLine{"  boots " + row.Brot + ", no screen flip", gen.Eva.Warn})
		}
	}
	L = append(L, paneLine{"", fg})
	add("Type", d.TypeLabel, typeHue(row.Base))
	add("Genre", row.Genre, fg)
	add("Maker", row.Manufacturer, fg)
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
	add("Players", row.Plr, prov("plr"))
	add("Controls", d.Ctl, prov("ctl"))
	add("Special", row.Spc, prov("spc"))
	add("Flip", row.Flip, fg)
	add("Commit", row.Act+rel(row.Act), mu)
	add("Source", data.SrcFull(row.Src), fg)
	add("Repo", row.Repo, mu)
	if row.Deprecated {
		add("Status", "deprecated", gen.Eva.Danger)
	}
	if row.Beta {
		L = append(L, paneLine{"Patreon beta: needs Jotego's jtbeta.zip", mu})
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
	a.paintStatus(c)
	body := l.Body
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
	// shots strip
	stripH := 54
	if !l.Portrait {
		stripH = 72
	}
	slots := shotSlots(row)
	if len(slots) > 0 {
		bw := (body.Dx() - (len(slots)-1)*3) / len(slots)
		if bw > 96 {
			bw = 96
		}
		bh := stripH
		x := body.Min.X
		for _, s := range slots {
			box := image.Rect(x, y, x+bw, y+bh)
			key := row.Img
			if s == "system" {
				key = row.Core
			}
			a.paintImageBox(c, box, key, s)
			x += bw + 3
		}
		y += bh + 3
	}
	// spec lines, scrollable
	lines := a.detailLines(row, d, i)
	entries := a.launchEntries(row, i)
	avail := body.Max.Y - y
	lh := a.sm.H + 1
	const launchMax = 5 // visible launch entries; more scroll under the pick
	shown := len(entries)
	if shown > launchMax {
		shown = launchMax
	}
	launchH := (shown + 1) * lh
	specH := avail - launchH - 2
	if specH < lh*4 {
		specH = lh * 4
	}
	maxLines := specH / lh
	if a.detail.scroll > len(lines)-maxLines {
		a.detail.scroll = len(lines) - maxLines
	}
	if a.detail.scroll < 0 {
		a.detail.scroll = 0
	}
	sc := a.sm.Cols(body.Dx())
	for n := a.detail.scroll; n < len(lines) && n < a.detail.scroll+maxLines; n++ {
		c.Text(body.Min.X, y, a.sm, gfx.Fit(lines[n].text, sc), lines[n].col)
		y += lh
	}
	// launch section
	y = body.Max.Y - launchH
	c.HLine(body.Min.X, body.Max.X-1, y-1, gen.Eva.Line)
	c.Text(body.Min.X, y, a.sm, "Launch (A):", gen.Eva.Muted)
	y += lh
	if a.detail.pick >= len(entries) {
		a.detail.pick = len(entries) - 1
	}
	if a.detail.pick < 0 {
		a.detail.pick = 0
	}
	first := 0
	if a.detail.pick >= shown {
		first = a.detail.pick - shown + 1
	}
	for n := first; n < len(entries) && n < first+shown; n++ {
		e := entries[n]
		col := gen.Eva.Fg
		prefix := "  "
		if n == a.detail.pick {
			col = gen.Eva.Accent
			prefix = "> "
		}
		label := e.label
		if !e.ok {
			label += " (not on card)"
			col = gen.Eva.Muted
		}
		if n == first+shown-1 && n < len(entries)-1 {
			label += " (+" + itoa(len(entries)-1-n) + " more)"
		}
		c.Text(body.Min.X, y, a.sm, gfx.Fit(prefix+label, sc), col)
		y += lh
	}
	if l.Portrait {
		a.paintHint(c, "A launch  Y fav  </> next  L/R info  B back")
	} else {
		a.paintHint(c, "A launch  Y fav  </> next row  U/D pick  L/R info  B back")
	}
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

func (a *App) paintImageBox(c *gfx.Canvas, box image.Rectangle, key, slot string) {
	c.Fill(box, gen.Eva.Surface)
	c.Box(box, gen.Eva.Line)
	req := ImageReq{Key: key, Slot: slot, W: box.Dx() - 2, H: box.Dy() - 2}
	img, st := a.cfg.Images.Get(req)
	if img == nil {
		a.want(req)
		text := "no shot"
		if st == ImageLoading {
			text = "loading"
		} else if st == ImageOffline {
			text = "offline"
		}
		a.placeholder(c, box, text)
		return
	}
	p := image.Pt(box.Min.X+(box.Dx()-img.Rect.Dx())/2, box.Min.Y+(box.Dy()-img.Rect.Dy())/2)
	if slot == "system" {
		c.BlitAlpha(p, img)
	} else {
		c.Blit(p, img)
	}
}

func (a *App) actDetails(k platform.Key) bool {
	row, _, _ := a.current()
	if row == nil {
		a.screen = ScreenList
		a.all = true
		return true
	}
	switch k {
	case platform.KeyBack:
		if a.detail.from == ScreenShot {
			a.screen = ScreenShot
		} else {
			a.screen = ScreenList
		}
	case platform.KeyUp:
		if a.detail.pick > 0 {
			a.detail.pick--
		}
	case platform.KeyDown:
		a.detail.pick++
	case platform.KeyPageUp:
		a.detail.scroll -= 4
	case platform.KeyPageDown:
		a.detail.scroll += 4
	case platform.KeyLeft:
		if a.cursor > 0 {
			a.cursor--
			a.detail = detailState{from: a.detail.from}
			a.ensureVisible()
		}
	case platform.KeyRight:
		if a.cursor < len(a.view)-1 {
			a.cursor++
			a.detail = detailState{from: a.detail.from}
			a.ensureVisible()
		}
	case platform.KeySpace:
		a.cfg.Favorites[row.K] = !a.cfg.Favorites[row.K]
		if !a.cfg.Favorites[row.K] {
			delete(a.cfg.Favorites, row.K)
			a.Notice("favorite removed", 2*time.Second)
		} else {
			a.Notice("favorite added", 2*time.Second)
		}
		if a.cfg.FavChanged != nil {
			a.cfg.FavChanged()
		}
		k := row.K
		a.rebuild()
		a.moveToKey(k)
		if a.cursor >= len(a.view) || a.ds.Rows[a.view[a.cursor]].K != k {
			a.screen = ScreenList // the row left the filtered view
		}
	case platform.KeyEnter:
		_, _, i := a.current()
		entries := a.launchEntries(row, i)
		if len(entries) > 0 && a.cfg.Launch != nil {
			e := entries[min(a.detail.pick, len(entries)-1)]
			if !e.ok {
				a.Notice("that file is not on the card", 3*time.Second)
				return true
			}
			a.cfg.Launch(e.path)
		}
		return false
	default:
		return false
	}
	a.all = true
	return true
}
