// Package app is the whole user interface as a state machine: it owns the
// dataset, the sort, filters, cursor and view, turns key events into state
// changes, and paints the logical canvas. It knows nothing about the
// framebuffer, evdev or files; the device loop and the PC harness drive it
// through Handle, Tick and Paint.
package app

import (
	"image"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/fonts"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Screen is which view is showing.
type Screen int

const (
	ScreenList Screen = iota
	ScreenDetails
	ScreenShot
	ScreenFilter
	ScreenSettings
	ScreenCalibrate
)

func (s Screen) String() string {
	return [...]string{"list", "details", "screen", "filter", "settings", "calibrate"}[s]
}

// Config is what the app needs from its host.
type Config struct {
	PhysW, PhysH int // physical frame, 320x240
	Rotation     gfx.Rotation
	SafeInset    int
	Now          func() time.Time
	ClockTrusted bool
	Images       Images
	Status       func(i int) data.Status // install status per row index; nil = unknown
	Favorites    map[string]bool
	Launch       func(path string) // called with a card-relative path; the host exits
	Quit         func()
	Version      string

	// Alternatives lists installed alternative MRAs for a row (card-relative paths).
	Alternatives func(r *data.Row) []string
	// FavChanged fires after a favorite toggle so the host can persist.
	FavChanged func()
	// SettingsChanged fires after rotation or safe-zone changes.
	SettingsChanged func()
	// Action asks the host for: prefetch (arg on/off), rescan, refresh, clearimg.
	Action func(kind, arg string)
}

// App is the state machine.
type App struct {
	cfg  Config
	body *gfx.Font
	sm   *gfx.Font
	lay  Layout
	rot  gfx.Rotation

	logical  *gfx.Canvas // what views paint into
	physical *image.RGBA // rotated frame handed to the display

	ds      *data.Dataset
	mode    data.SortMode
	filters data.Filters
	order   []int // ds.Order(mode)
	view    []int // after filters
	seen    *data.Seen
	split   int  // marker after view[split]; -1 none
	topMark bool // "nothing new" marker on top

	screen Screen
	cursor int // index into view
	top    int // first visible screen line
	slot   int // screen view slot index
	detail detailState
	panel  panelState

	rep    repeater
	notice string
	until  time.Time
	net    string // status bar right text: "offline", "updating", "data 2h ago"
	all    bool   // full repaint pending
}

type detailState struct {
	scroll int
	pick   int // launch entry cursor
}

// New builds an app around a dataset. stored is the persisted last-look
// record from the previous run, nil on a first run.
func New(cfg Config, ds *data.Dataset, stored *data.SeenRecord) *App {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Images == nil {
		cfg.Images = noImages{}
	}
	if cfg.Favorites == nil {
		cfg.Favorites = map[string]bool{}
	}
	a := &App{cfg: cfg, body: fonts.Body(), sm: fonts.Small(), rot: cfg.Rotation, split: -1}
	a.physical = image.NewRGBA(image.Rect(0, 0, cfg.PhysW, cfg.PhysH))
	a.setRotation(cfg.Rotation)
	a.SetData(ds, stored)
	return a
}

func (a *App) setRotation(rot gfx.Rotation) {
	a.rot = rot
	w, h := a.cfg.PhysW, a.cfg.PhysH
	if rot.Rotated() {
		w, h = h, w
	}
	a.logical = gfx.New(w, h)
	a.lay = NewLayout(w, h, a.cfg.SafeInset, a.body)
	if a.ds != nil {
		a.ensureVisible()
	}
	a.all = true
}

// SetRotation switches orientation (settings / calibration).
func (a *App) SetRotation(rot gfx.Rotation) { a.setRotation(rot) }

// SetInset changes the safe-zone inset.
func (a *App) SetInset(px int) {
	if px < 0 {
		px = 0
	}
	if px > 24 {
		px = 24
	}
	a.cfg.SafeInset = px
	a.setRotation(a.rot)
}

// Rotation and Inset report the current display settings.
func (a *App) Rotation() gfx.Rotation { return a.rot }
func (a *App) Inset() int             { return a.cfg.SafeInset }

// SetData installs a dataset (first load or a live swap), keeping the cursor
// on the same key. stored is the persisted seen record, nil on first ever run.
func (a *App) SetData(ds *data.Dataset, stored *data.SeenRecord) {
	var curK string
	if a.ds != nil && a.cursor < len(a.view) {
		curK = a.ds.Rows[a.view[a.cursor]].K
	}
	first := a.ds == nil
	a.ds = ds
	if first {
		a.seen = data.InitSeen(stored, ds.Rows, a.cfg.Now(), a.cfg.ClockTrusted)
	} else if a.seen != nil {
		a.seen.Bank(ds.Rows)
	}
	a.rebuild()
	if curK != "" {
		a.moveToKey(curK)
	}
	a.all = true
}

// Seen exposes the record to persist.
func (a *App) Seen() *data.Seen { return a.seen }

// FavoriteSet exposes the live favorites set.
func (a *App) FavoriteSet() map[string]bool { return a.cfg.Favorites }

// Data exposes the dataset.
func (a *App) Data() *data.Dataset { return a.ds }

// rebuild recomputes order, view and marker from the current state.
func (a *App) rebuild() {
	a.order = a.ds.Order(a.mode)
	fav := func(k string) bool { return a.cfg.Favorites[k] }
	a.view = data.Apply(a.ds, a.order, &a.filters, a.cfg.Status, fav)
	a.split = -1
	a.topMark = false
	if a.seen != nil && a.seen.MarkerOn(a.mode) {
		a.split = a.seen.SplitAt(a.ds, a.view, a.mode)
		a.topMark = a.split < 0
	}
	if a.cursor >= len(a.view) {
		a.cursor = len(a.view) - 1
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
	a.ensureVisible()
	a.all = true
}

func (a *App) moveToKey(k string) {
	for i, idx := range a.view {
		if a.ds.Rows[idx].K == k {
			a.cursor = i
			a.ensureVisible()
			return
		}
	}
}

// CursorKey is the key under the cursor, "" when the view is empty.
func (a *App) CursorKey() string {
	if a.cursor < len(a.view) {
		return a.ds.Rows[a.view[a.cursor]].K
	}
	return ""
}

// MoveToKey puts the cursor on a key if it is in view.
func (a *App) MoveToKey(k string) { a.moveToKey(k); a.all = true }

// SetSort switches the sort mode.
func (a *App) SetSort(m data.SortMode) {
	k := a.CursorKey()
	a.mode = m
	a.rebuild()
	if k != "" {
		a.moveToKey(k)
	}
}

// Sort reports the mode.
func (a *App) Sort() data.SortMode { return a.mode }

// Filters exposes the filters (copy).
func (a *App) Filters() data.Filters { return a.filters }

// SetFilters installs filters and re-applies them.
func (a *App) SetFilters(f data.Filters) {
	k := a.CursorKey()
	a.filters = f
	a.cursor = 0
	a.rebuild()
	if k != "" {
		a.moveToKey(k)
	}
}

// Notice shows a short message in the status bar.
func (a *App) Notice(s string, d time.Duration) {
	a.notice = s
	a.until = a.cfg.Now().Add(d)
	a.all = true
}

// SetNet sets the status bar's right-hand text.
func (a *App) SetNet(s string) {
	if a.net != s {
		a.net = s
		a.all = true
	}
}

// Screen reports the current view.
func (a *App) Screen() Screen { return a.screen }

// screenLine maps a view position to its list line, counting marker lines.
func (a *App) screenLine(pos int) int {
	n := pos
	if a.topMark {
		n++
	}
	if a.split >= 0 && pos > a.split {
		n++
	}
	return n
}

// totalLines is how many list lines the view occupies, markers included.
func (a *App) totalLines() int {
	n := len(a.view)
	if a.topMark || a.split >= 0 {
		n++
	}
	return n
}

func (a *App) ensureVisible() {
	if len(a.view) == 0 {
		a.top = 0
		return
	}
	line := a.screenLine(a.cursor)
	if line < a.top {
		a.top = line
	}
	last := line
	if a.split == a.cursor {
		last++ // keep the marker after the last unseen row reachable
	}
	if last >= a.top+a.lay.Lines {
		a.top = last - a.lay.Lines + 1
	}
	if a.top < 0 {
		a.top = 0
	}
}

// Handle applies one key event and returns true when a repaint is needed.
func (a *App) Handle(ev platform.Event) bool {
	if !ev.Pressed {
		a.rep.release(ev.Key)
		return false
	}
	pace := time.Duration(0)
	if a.screen == ScreenShot && (ev.Key == platform.KeyLeft || ev.Key == platform.KeyRight) {
		pace = repeatShot
	}
	a.rep.press(ev.Key, ev.At, pace)
	return a.act(ev.Key)
}

// Tick runs due repeats and expires notices; returns true to repaint.
func (a *App) Tick(now time.Time) bool {
	changed := false
	for k := a.rep.due(now); k != platform.KeyNone; k = a.rep.due(now) {
		if a.act(k) {
			changed = true
		}
		break // one repeat per tick keeps input responsive
	}
	if a.notice != "" && now.After(a.until) {
		a.notice = ""
		a.all = true
		changed = true
	}
	return changed
}

// NextTick reports when Tick next needs to run; zero when nothing is pending.
func (a *App) NextTick() time.Time {
	t := a.rep.nextAt()
	if a.notice != "" && (t.IsZero() || a.until.Before(t)) {
		t = a.until
	}
	return t
}

// act performs a key on the current screen.
func (a *App) act(k platform.Key) bool {
	switch a.screen {
	case ScreenList:
		return a.actList(k)
	case ScreenDetails:
		return a.actDetails(k)
	case ScreenShot:
		return a.actShot(k)
	case ScreenFilter, ScreenSettings:
		return a.actPanel(k)
	case ScreenCalibrate:
		return a.actCalibrate(k)
	}
	return false
}

func (a *App) actList(k platform.Key) bool {
	n := len(a.view)
	switch k {
	case platform.KeyUp:
		if a.cursor > 0 {
			a.cursor--
		}
	case platform.KeyDown:
		if a.cursor < n-1 {
			a.cursor++
		}
	case platform.KeyPageUp:
		a.cursor -= a.lay.Lines
		if a.cursor < 0 {
			a.cursor = 0
		}
	case platform.KeyPageDown:
		a.cursor += a.lay.Lines
		if a.cursor > n-1 {
			a.cursor = n - 1
		}
	case platform.KeyHome:
		a.cursor = 0
	case platform.KeyEnd:
		a.cursor = n - 1
	case platform.KeySpace:
		if a.mode == data.SortUpdated {
			a.SetSort(data.SortDebut)
		} else {
			a.SetSort(data.SortUpdated)
		}
		return true
	case platform.KeyTab:
		a.openPanel(ScreenFilter)
		return true
	case platform.KeyEnter:
		if n > 0 {
			a.screen = ScreenDetails
			a.detail = detailState{}
			a.all = true
		}
		return true
	case platform.KeyRight:
		if n > 0 {
			a.screen = ScreenShot
			a.slot = 0
			a.all = true
		}
		return true
	case platform.KeyBack:
		if a.cfg.Quit != nil {
			a.cfg.Quit()
		}
		return false
	default:
		return false
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
	a.ensureVisible()
	a.all = true
	return true
}

// current returns the row under the cursor, nil when the view is empty.
func (a *App) current() (*data.Row, *data.Derived, int) {
	if a.cursor >= len(a.view) {
		return nil, nil, -1
	}
	i := a.view[a.cursor]
	return &a.ds.Rows[i], &a.ds.Der[i], i
}

func (a *App) status(i int) data.Status {
	if a.cfg.Status == nil || i < 0 {
		return data.StatusUnknown
	}
	return a.cfg.Status(i)
}

// Paint renders whatever changed and returns the physical frame with the
// rectangles that need presenting (nil when nothing changed).
func (a *App) Paint() (*image.RGBA, []image.Rectangle) {
	if !a.all {
		return a.physical, nil
	}
	a.all = false
	c := a.logical
	c.Fill(c.Rect, gen.Eva.Bg)
	switch a.screen {
	case ScreenList:
		a.paintList(c)
	case ScreenDetails:
		a.paintDetails(c)
	case ScreenShot:
		a.paintShot(c)
	case ScreenFilter, ScreenSettings:
		a.paintList(c)
		a.paintPanel(c)
	case ScreenCalibrate:
		a.paintCalibrate(c)
	}
	dirty := c.TakeDirty()
	var out []image.Rectangle
	for _, r := range dirty {
		out = append(out, gfx.RotateRect(a.physical, c.RGBA, r, a.rot))
	}
	return a.physical, out
}

// Logical exposes the unrotated canvas (harness and tests).
func (a *App) Logical() *image.RGBA { return a.logical.RGBA }

// helpers shared by the views

func (a *App) dateCol(iso string) string {
	if len(iso) < 10 {
		return "     "
	}
	year := a.cfg.Now().Year()
	if !a.ds.Updated.IsZero() {
		year = a.ds.Updated.Year()
	}
	if iso[:4] == itoa(year) {
		return iso[5:10]
	}
	return " " + iso[:4]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func statusGlyph(st data.Status) (string, rgb) {
	switch st {
	case data.StatusCurrent:
		return "+", gen.Eva.Ok
	case data.StatusOutdated:
		return "^", gen.Eva.Warn
	case data.StatusFoundUndated:
		return "~", gen.Eva.Muted
	case data.StatusNotFound:
		return "-", gen.Eva.Muted
	}
	return " ", gen.Eva.Muted
}

func statusText(st data.Status, cardDate string) string {
	switch st {
	case data.StatusCurrent:
		return "on card, current"
	case data.StatusOutdated:
		if cardDate != "" {
			return "on card, older: " + cardDate
		}
		return "on card, older build"
	case data.StatusFoundUndated:
		return "on card, build date unknown"
	case data.StatusNotFound:
		return "not found on card"
	}
	return "checking card.."
}

// chips lists the row's badges in the site's title-cell order.
func chips(r *data.Row, d *data.Derived) []string {
	var out []string
	if d.BatchN >= 2 {
		out = append(out, "batch of "+itoa(d.BatchN))
	}
	if r.Beta {
		out = append(out, "beta")
	}
	if r.Deprecated {
		out = append(out, "deprecated")
	}
	if r.Brot != "" {
		dir := "CW"
		if strings.Contains(r.Brot, "CCW") {
			dir = "CCW"
		}
		out = append(out, "boots "+dir)
	}
	return out
}

func typeHue(base string) rgb {
	switch base {
	case "Arcade":
		return gen.Eva.TypeArcade
	case "Console":
		return gen.Eva.TypeConsole
	case "Computer":
		return gen.Eva.TypeComputer
	}
	return gen.Eva.TypeOther
}

// rotShort compresses the MAD rotation string for narrow pane lines.
func rotShort(rot string) string {
	switch rot {
	case "Horizontal":
		return "Horiz"
	case "Horizontal (180)":
		return "Horiz 180"
	case "Vertical (CW)":
		return "Vert CW"
	case "Vertical (CCW)":
		return "Vert CCW"
	case "Vertical":
		return "Vert"
	}
	return rot
}

// plrShort compresses "2 (alternating)" to "2P alt".
func plrShort(plr string) string {
	if plr == "" {
		return ""
	}
	s := strings.Replace(plr, " (simultaneous)", "P sim", 1)
	s = strings.Replace(s, " (alternating)", "P alt", 1)
	if !strings.Contains(s, "P") {
		s += "P"
	}
	return s
}
