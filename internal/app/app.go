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
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// Screen is which view is showing.
type Screen int

const (
	ScreenList Screen = iota
	ScreenDetails
	ScreenShot
	ScreenFilter
	ScreenOptions
	ScreenCalibrate
	ScreenUpdate
	ScreenTroubleshooting
	ScreenScan
)

func (s Screen) String() string {
	return [...]string{"list", "details", "screen", "filter", "options", "calibrate", "update", "troubleshooting", "scan"}[s]
}

// Config is what the app needs from its host.
type Config struct {
	PhysW, PhysH int // physical frame: 320x240, or the fit-display size the host chose
	Rotation     gfx.Rotation
	SafeInsetX   int              // safe-zone margin at the left and right edges, as viewed
	SafeInsetY   int              // and at the top and bottom
	Now          func() time.Time // calendar dates, possibly corrected before NTP
	TimerNow     func() time.Time // same clock as input events and Tick/Frame
	ClockTrusted bool
	Images       Images
	Status       func(i int) data.Status // install status per row index; nil = unknown
	Favorites    map[string]bool
	Launch       func(path string) // called with a card-relative path; the host exits
	Quit         func()
	Version      string
	Support      *SupportHooks

	// FavoritesUnavailable prevents edits after the host could not read the file.
	FavoritesUnavailable bool

	// Alternatives lists installed alternative MRAs for a row (card-relative paths).
	Alternatives func(r *data.Row) []string
	// Versions remembers each row's last chosen version by row key (a
	// card-relative launch path): Details opens on it and Start launches it
	// from anywhere. The main version is not recorded.
	Versions map[string]string
	// VersionChanged fires when a remembered version changes so the host can persist.
	VersionChanged func()
	// RecentLaunches is the launch history, newest first: every launch from the
	// app adds to it (see Recents), whether or not the Recents view is on.
	RecentLaunches []data.Recent
	// RecentsChanged fires when the launch history changes so the host can persist.
	RecentsChanged func()
	// FavChanged fires after a favorite toggle so the host can persist.
	FavChanged func()
	// FiltersChanged fires when filter choices change so the host can persist them.
	FiltersChanged func()
	// SettingsChanged fires after a preference or sort order changes.
	SettingsChanged func()
	// Action asks the host for: prefetch (arg on/off), rescan, refresh, clearimg.
	Action func(kind, arg string)
	// Progress reports the picture prefetch state (files present, total).
	Progress func() (have, total int)
	// Exists reports whether a card-relative file is present (launch targets).
	Exists func(rel string) bool
	// Launcher reports whether the main-menu launcher is enabled (nil = unsupported).
	Launcher func() bool
	// Scroll is the held-scrolling speed in rows per second: 20, 30, 60.
	Scroll string
	// HoldDelay is the navigation repeat delay in milliseconds: 200, 300, 500.
	HoldDelay int
	// Screensaver is the idle timeout: "off", "1", "2", "5", "10" minutes.
	Screensaver string
	// RememberSort restores LastSort at startup; otherwise start with latest updates.
	RememberSort   bool
	FollowRotation bool
	FilterRotation bool // strict filter on the current orientation
	// InstalledOnly is Options -> Sources: installed only. Sources whose Downloader
	// database the card lacks (SetHiddenSources) leave every view.
	InstalledOnly bool
	// Recents is Options -> Recents view: the launch history joins the Y
	// cycle after Favorites.
	Recents  bool
	LastSort data.SortMode
	// OpenAtBoot and ReturnAfterGame are Options -> Operation switches the
	// host's resident launcher acts on; both need the launcher enabled.
	OpenAtBoot      bool
	ReturnAfterGame bool
	// TitleFont draws list titles in "tall" (default: the narrow font at the
	// body font's height), "narrow" or "normal" (the body font).
	TitleFont string
	// ListShot is the pane thumbnail preference: "gameplay" (default) or "title".
	ListShot string
	// DateFormat is the list date column: "mm-dd" (default), "dd-mm",
	// "mon-d", "d-mon" or "yymmdd".
	DateFormat string
	// ListLayout is the main view's arrangement: "list" (default), "split"
	// or "picture" (see Layout.Style).
	ListLayout string
	// Canvas is Options -> Canvas, "fit" (default) or "320x240"; the host
	// applies it at the next start, the app only shows and saves it.
	Canvas string
	// MenuButton is Options -> Menu button: what the pad's MiSTer menu (OSD)
	// button does here, "options" (default) or "leave".
	MenuButton string
	// ButtonLabels names the pad buttons in the legends: "mister" (default:
	// A B X Y), "xbox", "playstation" or "numbers" (see buttons.go).
	ButtonLabels string
}

// App is the state machine.
type App struct {
	cfg    Config
	body   *gfx.Font
	sm     *gfx.Font
	narrow *gfx.Font // list titles, proportionally spaced
	tall   *gfx.Font // the same at the body font's height
	lay    Layout
	rot    gfx.Rotation

	logical  *gfx.Canvas // what views paint into
	physical *image.RGBA // rotated frame handed to the display

	ds      *data.Dataset
	mode    data.SortMode
	query   string // keyboard title search, kept only for this session
	filters data.Filters
	order   []int // ds.Order(mode)
	view    []int // after filters
	total   int   // rows the sources rule leaves, the list's "N releases"
	// hiddenSrc are the sources without a Downloader database on the card,
	// nil until a card scan reports; iniKnown/iniFound say whether that
	// report found a downloader.ini at all.
	hiddenSrc          map[string]bool
	iniKnown, iniFound bool
	seen               *data.Seen
	split              int  // marker after view[split]; -1 none
	topMark            bool // "nothing new" marker on top

	screen     Screen
	cursor     int  // index into view
	top        int  // first visible screen line
	shortPage  bool // group jumps may leave space below the last group
	slot       int  // screen view slot index
	detail     detailState
	panel      panelState
	update     updater.State
	updateView updateView
	support    supportView

	wants      []ImageReq // pictures this frame asked for, in priority order
	rep        repeater
	down       map[platform.Key]bool // keys currently held, across all devices
	menuAt     time.Time             // when the Menu button went down in Options mode; zero while up
	menuHinted bool                  // the hold hint is showing
	// where closing Options returns to (menu.go): the screen it was opened
	// over, the row key Details or the artwork showed, and the Filters
	// browsing state kept aside while Options uses the panel.
	optionsFrom Screen
	optionsKey  string
	filterHeld  *panelState
	notice      string
	until       time.Time
	net         string // status bar right text: "offline", "updating", "data 2h ago"
	appUpdate   string
	scanReady   bool
	scanError   string
	scanCounts  bool // the finished scan delivered statuses worth showing
	all         bool // full repaint pending
	saver       screensaver
	marquee     marqueeState
}

type detailState struct {
	scroll int       // first information line wanted at the top
	pixel  int       // where the information sits now, in pixels, easing toward scroll
	next   time.Time // the next animation frame
	lines  int       // visible information lines, used for paging
	pick   int       // launch entry cursor
	from   Screen    // where B returns to
}

// New builds an app around a dataset. stored is the persisted last-look
// record from the previous run, nil on a first run.
func New(cfg Config, ds *data.Dataset, stored *data.SeenRecord) *App {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.TimerNow == nil {
		cfg.TimerNow = cfg.Now
	}
	if cfg.Images == nil {
		cfg.Images = noImages{}
	}
	if cfg.Favorites == nil {
		cfg.Favorites = map[string]bool{}
	}
	if cfg.Versions == nil {
		cfg.Versions = map[string]string{}
	}
	a := &App{cfg: cfg, body: fonts.Body(), sm: fonts.Small(), narrow: fonts.Narrow(), tall: fonts.NarrowTall(), rot: cfg.Rotation, split: -1, down: map[platform.Key]bool{}}
	if cfg.RememberSort && cfg.LastSort >= data.SortUpdated && cfg.LastSort <= data.SortRecents && (cfg.LastSort != data.SortRecents || cfg.Recents) {
		a.mode = cfg.LastSort
	}
	a.physical = image.NewRGBA(image.Rect(0, 0, cfg.PhysW, cfg.PhysH))
	a.setRotation(cfg.Rotation)
	a.SetData(ds, stored)
	a.saver.lastInput = cfg.TimerNow()
	return a
}

func (a *App) setRotation(rot gfx.Rotation) {
	orientationChanged := a.rot.Rotated() != rot.Rotated()
	a.rot = rot
	w, h := a.cfg.PhysW, a.cfg.PhysH
	if rot.Rotated() {
		w, h = h, w
	}
	a.logical = gfx.New(w, h)
	a.lay = NewLayout(w, h, a.cfg.SafeInsetX, a.cfg.SafeInsetY, a.body, a.rowFont().W, a.dateCols(), a.ListLayout())
	if a.ds != nil {
		if orientationChanged && a.cfg.FilterRotation {
			a.Refilter() // the strict filter follows the current orientation
		}
		a.ensureVisible()
	}
	a.all = true
}

// SetRotation switches orientation (settings / calibration).
func (a *App) SetRotation(rot gfx.Rotation) { a.setRotation(rot) }

// SetInset changes the safe-zone margins (physical horizontal, vertical).
func (a *App) SetInset(x, y int) {
	a.cfg.SafeInsetX, a.cfg.SafeInsetY = clampInset(x), clampInset(y)
	a.setRotation(a.rot)
}

func clampInset(px int) int {
	if px < 0 {
		return 0
	}
	if px > 40 {
		return 40
	}
	return px
}

// ScrollSpeed reports the held-scrolling speed setting.
func (a *App) ScrollSpeed() string { return a.scrollText() }

func (a *App) HoldDelay() int {
	switch a.cfg.HoldDelay {
	case 200, 500:
		return a.cfg.HoldDelay
	default:
		return 300
	}
}

// Rotation and Inset report the current display settings.
func (a *App) Rotation() gfx.Rotation { return a.rot }
func (a *App) Inset() (x, y int)      { return a.cfg.SafeInsetX, a.cfg.SafeInsetY }

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
	if a.screen == ScreenFilter || a.screen == ScreenOptions {
		a.buildPanel()
	}
	a.all = true
}

// Seen exposes the record to persist.
func (a *App) Seen() *data.Seen { return a.seen }

// FavoriteSet exposes the live favorites set.
func (a *App) FavoriteSet() map[string]bool { return a.cfg.Favorites }

// Versions is the remembered version per row key, for the host to persist.
func (a *App) Versions() map[string]string { return a.cfg.Versions }

// Data exposes the dataset.
func (a *App) Data() *data.Dataset { return a.ds }

// rebuild recomputes order, view and marker from the current state.
func (a *App) rebuild() {
	a.shortPage = false
	a.order = a.ds.Order(a.mode)
	if a.mode == data.SortRecents {
		a.order = a.ds.OrderRecents(a.cfg.RecentLaunches)
	}
	fav := func(k string) bool { return a.cfg.Favorites[k] }
	unseen := func(i int) bool { return a.seen != nil && a.seen.Unseen(&a.ds.Rows[i]) }
	filters := a.effectiveFilters()
	if a.mode == data.SortFavorites {
		filters.FavOnly = true
	}
	a.total = len(a.ds.Rows)
	if len(filters.SrcHidden) > 0 {
		a.total = 0
		for i := range a.ds.Rows {
			if !filters.SrcHidden[a.ds.Rows[i].Src] {
				a.total++
			}
		}
	}
	a.view = data.Apply(a.ds, a.order, &filters, a.cfg.Status, fav, unseen)
	if q := searchText(a.query); q != "" {
		matched := a.view[:0]
		for _, i := range a.view {
			if strings.Contains(searchText(a.ds.Der[i].Title), q) {
				matched = append(matched, i)
			}
		}
		a.view = matched
	}
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
	if m < data.SortUpdated || m > data.SortRecents || m == a.mode || (m == data.SortRecents && !a.cfg.Recents) {
		return
	}
	k := a.CursorKey()
	a.mode = m
	a.rebuild()
	if k != "" {
		a.moveToKey(k)
	}
	if a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
}

// Sort reports the mode.
func (a *App) Sort() data.SortMode { return a.mode }

// nextSort is the mode Y moves to: the cycle, with Recents after Favorites
// while the Recents view is on.
func (a *App) nextSort() data.SortMode {
	switch {
	case a.mode == data.SortRecents:
		return data.SortUpdated
	case a.mode == data.SortFavorites && a.cfg.Recents:
		return data.SortRecents
	}
	return data.NextSort(a.mode)
}

func (a *App) RememberSort() bool { return a.cfg.RememberSort }
func (a *App) OpenAtBoot() bool   { return a.cfg.OpenAtBoot }

// MenuButton is the Options -> Menu button choice: "options" or "leave".
func (a *App) MenuButton() string {
	if a.cfg.MenuButton == "leave" {
		return "leave"
	}
	return "options"
}

// Canvas is the Options -> Canvas choice: "fit" or "320x240".
func (a *App) Canvas() string {
	if a.cfg.Canvas == "320x240" {
		return "320x240"
	}
	return "fit"
}
func (a *App) ReturnAfterGame() bool { return a.cfg.ReturnAfterGame }
func (a *App) FollowRotation() bool  { return a.cfg.FollowRotation }
func (a *App) FilterRotation() bool  { return a.cfg.FilterRotation }

// InstalledOnly reports Options -> Sources: installed only.
func (a *App) InstalledOnly() bool { return a.cfg.InstalledOnly }

// SetInstalledOnly is the Sources option; the host calls Refilter after.
func (a *App) SetInstalledOnly(on bool) { a.cfg.InstalledOnly = on }

// SetHiddenSources takes a card scan's report of the sources without a
// Downloader database on the card (data.HiddenSources); found is whether
// a downloader.ini was read at all. The host calls Refilter after.
func (a *App) SetHiddenSources(hidden map[string]bool, found bool) {
	a.hiddenSrc = hidden
	a.iniKnown, a.iniFound = true, found
	if a.screen == ScreenOptions {
		a.buildPanel()
		a.all = true
	}
}

// Visible is the number of rows the list shows.
func (a *App) Visible() int { return len(a.view) }

// titleFonts are the list title choices, in Options order.
var titleFonts = []string{"normal", "narrow", "tall"}

// TitleFont is the list title font choice, one of titleFonts.
func (a *App) TitleFont() string {
	switch a.cfg.TitleFont {
	case "normal", "narrow":
		return a.cfg.TitleFont
	}
	return "tall"
}

// rowFont draws a list row's status glyph and date: the narrow font at
// the titles' height, so the tall one unless the titles are narrow.
func (a *App) rowFont() *gfx.Font {
	if a.TitleFont() == "narrow" {
		return a.narrow
	}
	return a.tall
}

// titleFont is the proportional font list titles draw in, nil for the
// body font.
func (a *App) titleFont() *gfx.Font {
	switch a.TitleFont() {
	case "narrow":
		return a.narrow
	case "tall":
		return a.tall
	}
	return nil
}

// ListShot is the pane thumbnail preference, "gameplay" or "title".
func (a *App) ListShot() string {
	if a.cfg.ListShot == "title" {
		return "title"
	}
	return "gameplay"
}

// ListLayout is the main view arrangement, one of listLayouts.
func (a *App) ListLayout() string {
	for _, s := range listLayouts {
		if s == a.cfg.ListLayout {
			return s
		}
	}
	return listLayouts[0]
}

// DateFormat is the list date column format, one of dateFormats.
func (a *App) DateFormat() string {
	for _, f := range dateFormats {
		if f == a.cfg.DateFormat {
			return f
		}
	}
	return dateFormats[0]
}

// Filters exposes the filters (copy).
func (a *App) Filters() data.Filters { return a.filters }

// SetFilters installs filters and re-applies them.
func (a *App) SetFilters(f data.Filters) {
	k := a.CursorKey()
	a.filters = f
	if a.cfg.FiltersChanged != nil {
		a.cfg.FiltersChanged()
	}
	a.cursor = 0
	a.rebuild()
	if k != "" {
		a.moveToKey(k)
	}
}

// Notice shows a short message in the status bar.
func (a *App) Notice(s string, d time.Duration) {
	a.notice = s
	a.until = a.cfg.TimerNow().Add(d)
	a.all = true
}

// SetClockTrusted flips the clock state once NTP has set it.
func (a *App) SetClockTrusted(t bool) {
	if t && !a.cfg.ClockTrusted && a.seen != nil {
		a.seen.State.T = a.cfg.Now().UTC().Format(time.RFC3339)
	}
	a.cfg.ClockTrusted = t
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
	maxTop := a.totalLines() - a.lay.Lines
	if a.shortPage {
		maxTop = a.totalLines() - 1
	}
	if a.top > maxTop {
		a.top = maxTop
	}
	if a.top < 0 {
		a.top = 0
	}
}

// Handle applies one key event and returns true when a repaint is needed.
// A press for a key that is already down is ignored: a keyboard encoder
// that Main also translates delivers every press twice (raw and through the
// virtual keyboard), and this folds the pair into one.
func (a *App) Handle(ev platform.Event) bool {
	if a.handleSaverInput(ev) {
		return true
	}
	if a.supportCapturing() {
		// Observe releases but do not navigate, search, launch or repeat. A
		// button held across the result boundary still needs a fresh press.
		if ev.Pressed {
			a.down[ev.Key] = true
		} else {
			delete(a.down, ev.Key)
		}
		return false
	}
	if ev.Key == platform.KeyMenu && !(a.screen == ScreenTroubleshooting && a.support.mode == "pad") {
		// a held Menu leaves the app (menu.go); the tester only logs it
		if ev.Pressed {
			a.menuPress(ev.At)
		} else {
			a.menuRelease()
		}
	}
	if a.screen == ScreenUpdate {
		return a.handleUpdate(ev)
	}
	if a.screen == ScreenTroubleshooting && a.support.mode == "pad" {
		return a.handlePadTest(ev)
	}
	if ev.Pressed && ev.Text != 0 && a.screen == ScreenList {
		// Space still sorts before a search begins; while searching it is text.
		if ev.Text != ' ' || a.query != "" {
			return a.typeSearch(ev.Text)
		}
	}
	if ev.Key == platform.KeyNone || ev.Key == platform.KeyOther {
		return false
	}
	if !ev.Pressed {
		if a.down[ev.Key] {
			delete(a.down, ev.Key)
			a.rep.release(ev.Key)
		}
		// letting go of Select puts the ordinary legend back
		if ev.Key == platform.KeySelect && a.screen == ScreenList {
			a.all = true
			return true
		}
		return false
	}
	if a.down[ev.Key] {
		return false
	}
	a.down[ev.Key] = true
	a.rep.press(ev.Key, ev.At)
	if ev.Key == platform.KeyMenu {
		return a.menuButton()
	}
	if a.screen == ScreenList || a.screen == ScreenFilter || a.screen == ScreenOptions {
		switch ev.Key {
		case platform.KeyUp, platform.KeyDown, platform.KeyLeft, platform.KeyRight:
			a.rep.next = ev.At.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
		case platform.KeyPageUp, platform.KeyPageDown:
			if a.screen == ScreenList {
				a.rep.next = ev.At.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
			}
		}
	}
	return a.act(ev.Key)
}

// repeatStep says whether a held key repeats on the current screen and how
// fast; 0 means it does not.
func (a *App) repeatStep(k platform.Key, count int) time.Duration {
	switch a.screen {
	case ScreenList:
		switch k {
		case platform.KeyBackspace:
			return repeatStep
		case platform.KeyUp, platform.KeyDown, platform.KeyLeft, platform.KeyRight, platform.KeyPageUp, platform.KeyPageDown:
			return scrollPace(a.cfg.Scroll) // rows, pages and groups share the same pace
		}
	case ScreenShot:
		switch k {
		case platform.KeyLeft, platform.KeyRight:
			return repeatStep
		}
	case ScreenDetails:
		switch k {
		case platform.KeyLeft, platform.KeyRight:
			return repeatStep
		case platform.KeyUp, platform.KeyDown:
			return repeatPage
		}
	case ScreenFilter, ScreenOptions:
		switch k {
		case platform.KeyUp, platform.KeyDown:
			return scrollPace(a.cfg.Scroll)
		case platform.KeyLeft, platform.KeyRight, platform.KeyPageUp, platform.KeyPageDown:
			return repeatStep
		}
	case ScreenCalibrate:
		switch k {
		case platform.KeyLeft, platform.KeyRight, platform.KeyUp, platform.KeyDown:
			return repeatCalib
		}
	case ScreenUpdate:
		switch k {
		case platform.KeyUp, platform.KeyDown, platform.KeyPageUp, platform.KeyPageDown:
			return scrollPace(a.cfg.Scroll)
		}
	}
	return 0
}

// Tick runs due repeats and expires notices; returns true to repaint.
func (a *App) Tick(now time.Time) bool {
	changed := a.tickMenu(now)
	// Expire notices on every screen so NextTick cannot keep returning a past
	// deadline while Update All handles its own animation and cancel input.
	if a.notice != "" && !now.Before(a.until) {
		a.notice = ""
		a.all = true
		changed = true
	}
	if a.screen == ScreenUpdate {
		// Stay in the host's normal event loop so update progress and cancellation
		// keep arriving while the log scrolls. NextTick schedules these repeats.
		if k := a.rep.due(now, a.repeatStep); k != platform.KeyNone {
			changed = a.scrollUpdate(k) || changed
		}
		changed = a.tickUpdate(now) || changed
		return a.tickSaver(now) || changed
	}
	if a.screen == ScreenTroubleshooting {
		return a.tickSupport(now) || changed
	}
	if k := a.rep.due(now, a.repeatStep); k != platform.KeyNone {
		if a.act(k) {
			changed = true
		}
	}
	changed = a.tickMarquee(now) || changed
	changed = a.tickDetailScroll(now) || changed
	return a.tickSaver(now) || changed
}

// Frame is Tick for the vsync-driven loop the host runs while a key is
// held: called once per frame, it moves at most one step.
func (a *App) Frame(now time.Time) bool {
	a.saver.lastInput = now // a held direction is still activity
	changed := a.tickMenu(now)
	if k := a.rep.frameDue(now, a.repeatStep); k != platform.KeyNone {
		if a.act(k) {
			changed = true
		}
	}
	changed = a.tickDetailScroll(now) || changed // keeps paging smooth under a held key
	if a.notice != "" && !now.Before(a.until) {
		a.notice = ""
		a.all = true
		changed = true
	}
	return changed
}

// NextTick reports when Tick next needs to run; zero when nothing is pending.
func (a *App) NextTick() time.Time {
	t := a.rep.nextAt()
	if a.screen == ScreenTroubleshooting {
		t = a.support.next
	}
	if a.notice != "" && (t.IsZero() || a.until.Before(t)) {
		t = a.until
	}
	if next := a.nextSaverTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.nextMarqueeTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.nextDetailTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.nextMenuTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	return t
}

// act performs a key on the current screen.
func (a *App) act(k platform.Key) bool {
	switch a.screen {
	case ScreenScan:
		if k == platform.KeyBack || (k == platform.KeyEnter && a.scanReady) {
			a.openOptions()
			return true
		}
		return false
	case ScreenList:
		return a.actList(k)
	case ScreenDetails:
		return a.actDetails(k)
	case ScreenTroubleshooting:
		return a.actSupport(k)
	case ScreenShot:
		return a.actShot(k)
	case ScreenFilter, ScreenOptions:
		return a.actPanel(k)
	case ScreenCalibrate:
		return a.actCalibrate(k)
	}
	return false
}

func (a *App) actList(k platform.Key) bool {
	n := len(a.view)
	switch k {
	case platform.KeyBackspace:
		if a.query == "" {
			return false
		}
		a.setSearch(a.query[:len(a.query)-1])
		return true
	case platform.KeyUp:
		if a.cursor > 0 {
			a.cursor--
		}
	case platform.KeyDown:
		if a.cursor < n-1 {
			a.cursor++
		}
	case platform.KeyPageUp:
		a.jumpGroup(-1)
	case platform.KeyPageDown:
		a.jumpGroup(1)
	case platform.KeyHome:
		a.shortPage = false
		a.cursor = 0
	case platform.KeyEnd:
		a.shortPage = false
		a.cursor = n - 1
	case platform.KeySelect: // held, Y and X become the quick toggles (quick.go)
		a.all = true // the legend names them
		return true
	case platform.KeySpace:
		if a.quickHeld() {
			return a.cycleListLayout()
		}
		a.SetSort(a.nextSort())
		return true
	case platform.KeyTab:
		if a.quickHeld() {
			return a.cycleListShot()
		}
		a.openPanel(ScreenFilter)
		return true
	case platform.KeyEnter:
		if n > 0 {
			a.screen = ScreenDetails
			a.detail = detailState{from: ScreenList, pick: a.rememberedPick()}
			a.all = true
		}
		return true
	case platform.KeyStart: // launch the remembered version straight from the list
		if n > 0 {
			a.launchPick(a.rememberedPick())
		}
		return true
	case platform.KeyLeft: // a screen of rows up, the row centered like a step
		a.pageBy(-1)
	case platform.KeyRight:
		a.pageBy(1)
	case platform.KeyBack:
		if a.query != "" {
			a.setSearch("")
			return true
		}
		a.openOptions()
		return true
	default:
		return false
	}
	if a.cursor < 0 {
		a.cursor = 0
	}
	if k == platform.KeyUp || k == platform.KeyDown {
		a.shortPage = false
		a.top = centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines)
	}
	a.ensureVisible()
	a.all = true
	return true
}

// pageBy moves the cursor a screenful of rows in direction dir and keeps
// it centered, the way a single step does.
func (a *App) pageBy(dir int) {
	a.shortPage = false
	n := len(a.view)
	if n == 0 {
		return
	}
	a.cursor = max(0, min(n-1, a.cursor+dir*a.lay.Lines))
	a.top = centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines)
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

// Refilter re-applies the filters after an input to them changed (install
// statuses arrived), keeping the cursor on its row.
func (a *App) Refilter() {
	k := a.CursorKey()
	a.rebuild()
	if k != "" {
		a.moveToKey(k)
	}
	if a.screen == ScreenFilter {
		a.buildPanel()
	}
	a.all = true
}

// Invalidate forces a full repaint on the next Paint (a picture landed,
// a scan finished).
func (a *App) Invalidate() {
	a.all = true
	if a.screen == ScreenOptions {
		a.buildPanel() // the prefetch tally
	}
}

// Repeating reports whether a held key is driving repeats right now; the
// host then runs its vsync-driven frame loop (see Frame).
func (a *App) Repeating() bool {
	return a.screen != ScreenUpdate && a.screen != ScreenTroubleshooting && a.rep.held
}

// RepeatActive reports whether a held key has started repeating (a tap
// released before the delay never does), which is when background work
// should step aside.
func (a *App) RepeatActive() bool { return a.rep.held && a.rep.count > 0 }

// want records a picture this frame needs; the list goes to the provider
// once per paint, in the order the views asked, so the cursor's pane image
// comes before neighbours and prefetch.
func (a *App) want(req ImageReq) {
	for _, w := range a.wants {
		if w == req {
			return
		}
	}
	a.wants = append(a.wants, req)
}

// neighbourhood asks for the pane thumbnails of the rows around the cursor
// so tapping along finds them downloaded and decoded already: 16 rows
// ahead, 6 behind, nearest first. The provider works through the list in
// order and only while nothing nearer is missing, and a held key pauses it,
// so this never competes with scrolling.
func (a *App) neighbourhood() {
	if a.screen != ScreenList {
		return
	}
	box := a.lay.Thumb
	for d := 1; d <= 16; d++ {
		for _, pos := range []int{a.cursor + d, a.cursor - d} {
			if pos < 0 || pos >= len(a.view) || (d > 6 && pos < a.cursor) {
				continue
			}
			row := &a.ds.Rows[a.view[pos]]
			key, slot := thumbSlot(row, a.ListShot())
			if key != "" {
				a.want(ImageReq{Key: key, Slot: slot, W: box.Dx(), H: box.Dy(), Stretch: slot != "system" && row.ImgW > row.ImgH})
			}
		}
	}
}

// Paint renders whatever changed and returns the physical frame with the
// rectangles that need presenting (nil when nothing changed).
func (a *App) Paint() (*image.RGBA, []image.Rectangle) {
	if !a.all {
		return a.physical, nil
	}
	a.all = false
	a.wants = a.wants[:0]
	if a.screen != ScreenDetails {
		a.marquee = marqueeState{} // a return to Details starts its scroll afresh
	}
	c := a.logical
	c.Fill(c.Rect, gen.Eva.Bg)
	switch a.screen {
	case ScreenList:
		a.paintList(c)
	case ScreenDetails:
		a.paintDetails(c)
	case ScreenShot:
		a.paintShot(c)
	case ScreenFilter, ScreenOptions:
		a.paintPanel(c)
	case ScreenCalibrate:
		a.paintCalibrate(c)
	case ScreenUpdate:
		a.paintUpdate(c)
	case ScreenTroubleshooting:
		a.paintSupport(c)
	case ScreenScan:
		a.paintScan(c)
	}
	a.neighbourhood()
	a.cfg.Images.Want(a.wants)
	if a.saver.active {
		a.paintSaver(c)
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

// dateFormats are the list date column choices, in Options order.
var dateFormats = []string{"mm-dd", "dd-mm", "mon-d", "d-mon", "yymmdd"}

// dateFormatLabels name them in Options.
var dateFormatLabels = []string{"MM-DD", "DD-MM", "Mon D", "D Mon", "YYMMDD"}

// dateCols is the width of the list date column for the chosen format.
func (a *App) dateCols() int {
	switch a.DateFormat() {
	case "mm-dd", "dd-mm":
		return 5
	}
	return 6
}

// listYear is the year list dates are shown within: the catalogue's, or
// the clock's before any catalogue has loaded.
func (a *App) listYear() int {
	if a.ds != nil && !a.ds.Updated.IsZero() {
		return a.ds.Updated.Year()
	}
	return a.cfg.Now().Year()
}

// dateCol formats an ISO date for the list column, right-aligned in
// dateCols cells: the day within the list year, the year alone otherwise.
func (a *App) dateCol(iso string) string {
	cols := a.dateCols()
	s := formatListDate(a.DateFormat(), iso, a.listYear())
	for len(s) < cols {
		s = " " + s
	}
	return s
}

var monthShort = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// formatListDate renders "YYYY-MM-DD" in a list date format: YYMMDD always
// carries the full date; the others show the day only within year and the
// year alone for earlier ones. Anything unparsable is blank.
func formatListDate(format, iso string, year int) string {
	if len(iso) < 10 {
		return ""
	}
	yy, mm, dd := iso[2:4], iso[5:7], iso[8:10]
	if format == "yymmdd" {
		return yy + mm + dd
	}
	if iso[:4] != itoa(year) {
		return iso[:4]
	}
	day := strings.TrimLeft(dd, "0")
	mon := "???"
	if m := (int(mm[0])-'0')*10 + int(mm[1]) - '0'; m >= 1 && m <= 12 {
		mon = monthShort[m-1]
	}
	switch format {
	case "dd-mm":
		return dd + "-" + mm
	case "mon-d":
		return mon + " " + day
	case "d-mon":
		return day + " " + mon
	}
	return mm + "-" + dd
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
	case data.StatusOutdated, data.StatusLikelyOutdated:
		return "^", gen.Eva.Warn
	case data.StatusFoundUndated:
		return "~", gen.Eva.Muted
	case data.StatusNotFound:
		return "-", gen.Eva.Muted
	}
	return " ", gen.Eva.Muted
}

// statusText fits the pane's 20 columns.
func statusText(st data.Status, cardDate string) string {
	switch st {
	case data.StatusCurrent:
		return "current build"
	case data.StatusOutdated:
		if cardDate != "" {
			return "older: " + cardDate
		}
		return "older build"
	case data.StatusLikelyOutdated:
		return "older build likely"
	case data.StatusFoundUndated:
		return "on card, date unknown"
	case data.StatusNotFound:
		return "not on card"
	}
	return "checking card" + gfx.Ellipsis
}

// chips lists the row's badges in the site's title-cell order.
func chips(r *data.Row, d *data.Derived) []string {
	var out []string
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
