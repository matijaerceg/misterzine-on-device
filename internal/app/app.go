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
	ScreenViews   // Options -> Views: the checkbox page of the Y cycle
	ScreenCredits // Options -> Credits: who made it, what it builds on, the early adopters
	ScreenSaverOptions
)

func (s Screen) String() string {
	return [...]string{"list", "details", "screen", "filter", "options", "calibrate", "update", "troubleshooting", "scan", "views", "credits", "screensaver-options"}[s]
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
	// AltCore names the core of a version that may run on another core than
	// its row's ("" when unknown), for its label in the version picker.
	AltCore func(path string) string
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
	// ROMIssue reports a problem with the selected MRA's ROM files, and
	// block when MiSTer could not load the game, so Start refuses it;
	// otherwise the text is a warning. fresh bypasses cached Details
	// results when the user presses Start.
	ROMIssue func(rel string, fresh bool) (text string, block bool)
	// ROMKnown is the ROM check's stored answer for a file, if it has one,
	// without touching the card: the list asks it for every row. nil = no
	// background check, so the list never marks ROMs.
	ROMKnown func(rel string) (text string, block, known bool)
	// ROMProgress is how far the background ROM check has got (nil = none).
	ROMProgress func() (done, total int)
	// Launcher reports whether the main-menu launcher is enabled (nil = unsupported).
	Launcher func() bool
	// CanUpdateApp reports whether this card can fetch a new MisterZine on
	// its own, which needs Downloader (nil = unsupported).
	CanUpdateApp func() bool
	// Scroll is the held-scrolling speed in rows per second: 20, 30, 60.
	Scroll               string
	SmoothScrollDisabled bool
	// HoldDelay is the navigation repeat delay in milliseconds: 200, 300, 500.
	HoldDelay int
	// Screensaver is the idle timeout: "off", "1", "2", "5", "10" minutes.
	Screensaver         string
	SaverDisabled       bool
	TransitionsDisabled bool
	// LaunchTransition: "hold" (default) plays the launch animation only
	// when Start is held past cabHoldStart, a tap launching at once;
	// "always" plays it on every launch.
	LaunchTransition string
	// SaverStyle is what the saver shows: "word" (default: the lettering)
	// or "shots" (random arcade screenshots, Start held plays one); with
	// the shots, SaverBright is "half" (default) or "full" and SaverInfo is
	// what the caption says about a shot's game: "full" (default: the
	// pane's lines), "title" (the title alone) or "none".
	SaverStyle                               string
	SaverDim                                 string
	SaverBright                              string
	SaverInfo                                string
	SaverCard, SaverRotation, SaverFavorites bool
	SaverResolution                          string
	// RememberSort restores LastSort at startup; otherwise use DefaultSort.
	RememberSort   bool
	FollowRotation bool
	FilterRotation bool // strict filter on the current orientation
	// InstalledOnly is Options -> Sources: installed only. Sources whose Downloader
	// database the card lacks (SetHiddenSources) leave every view.
	InstalledOnly  bool
	ShowNonArcade  bool
	ArcadeIntro    bool // one-time explanation for upgraded installations
	ShowDeprecated bool // Include catalogue rows marked deprecated.
	// ViewsOff names the views Options -> Views left out of the Y cycle
	// (data.SortMode.Name); a fresh install lists only "recents".
	ViewsOff    []string
	LastSort    data.SortMode
	DefaultSort data.SortMode
	// OpenAtBoot and ReturnAfterGame are Options -> Operation switches the
	// host's resident launcher acts on; both need the launcher enabled.
	OpenAtBoot      bool
	ReturnAfterGame bool
	// ExitChord is Options -> Exit chord, the pad buttons that leave a
	// launched game for the menu: "" (off), "select-start" or
	// "lr-select-start"; the launcher watches for it.
	ExitChord string
	// TitleFont draws list titles in "tall" (default: the narrow font at the
	// body font's height), "narrow" or "normal" (the body font).
	TitleFont string
	// ListShot is the pane thumbnail preference: "gameplay" (default) or "title".
	ListShot string
	// DateFormat is the list date column: "mm-dd" (default), "dd-mm",
	// "mon-d", "d-mon" or "yymmdd".
	DateFormat string
	// ListLayout is the main view's arrangement: "list" (default), "split",
	// "picture" or "text" (see Layout.Style).
	ListLayout string
	// Canvas is Options -> Canvas, "full" (default), "fit" or "320x240"; the host
	// applies it live, the app shows and saves the selected choice.
	Canvas string
	// MenuButton is Options -> Menu button: what the pad's MiSTer menu (OSD)
	// button does here, "options" (default) or "leave".
	MenuButton string
	// ButtonLabels names the pad buttons in the legends: "mister" (default:
	// A B X Y), "xbox", "playstation" or "numbers" (see buttons.go).
	ButtonLabels string
	// OKButtons is Options -> OK button per pad, keyed by the pad's
	// vendor_product as in MiSTer's map file name: "a" or "b" overrides
	// the MENU OK choice read from that map; absent means auto (okbutton.go).
	OKButtons map[string]string
}

// App is the state machine.
type App struct {
	transition                    pageTransition
	layoutMotion                  layoutMotion
	paintedThumb, paintedPaneText image.Rectangle
	paintedThumbImage             *image.RGBA
	listMotion                    listMotion
	splash                        startupSplash
	optionSamples                 optionSampleClock
	cfg                           Config
	body                          *gfx.Font
	sm                            *gfx.Font
	narrow                        *gfx.Font // list titles, proportionally spaced
	tall                          *gfx.Font // the same at the body font's height
	lay                           Layout
	rot                           gfx.Rotation

	logical     *gfx.Canvas // what views paint into
	renderAhead bool        // the launch animation renders on its own goroutine
	physical    *image.RGBA // rotated frame handed to the display

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
	romMemo            map[int]data.ROMState // per row, cleared when the check or a choice changes (rom.go)
	seen               *data.Seen
	marker             bool                   // the since-visit status row heads the list (marks.go)
	sinceAdded         int                    // rows added since the last visit, over the enabled catalogue
	sinceUpdated       int                    // rows rebuilt since the last visit, over the enabled catalogue
	marks              []int                  // every marker line, by the view position it precedes (marks.go)
	viewsOff           map[data.SortMode]bool // views left out of the Y cycle (views.go)

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
	look       SaverLook                  // the saver ground's tuning (screensaver.go)
	released   map[platform.Key]time.Time // when each key last came up, for the bounce guard
	down       map[platform.Key]bool      // keys currently held, across all devices
	oks        okButtons                  // which pad pressed last and what its OK button is (okbutton.go)
	menuAt     time.Time                  // when the Menu button went down in Options mode; zero while up
	menuHinted bool                       // the hold hint is showing
	menuBar    int                        // hold progress since the hint, in pixels along the status bar's bottom line
	// where closing Options returns to (menu.go): the screen it was opened
	// over, the row key Details or the artwork showed, and the Filters
	// browsing state kept aside while Options uses the panel.
	optionsOpen    map[string]bool // session-only expansion, separate from the shared panel
	optionsFrom    Screen
	optionsKey     string
	filterHeld     *panelState
	notice         string
	until          time.Time
	net            string     // connection failures in the status bar
	catalogChecked time.Time  // last successful catalog check this session
	supporters     Supporters // Patreon supporters for the Credits page
	appUpdate      string
	scanReady      bool
	scanError      string
	scanCounts     bool // the finished scan delivered statuses worth showing
	all            bool // full repaint pending
	saver          screensaver
	marquee        marqueeState
	cab            launchCab
	holdLaunch     holdLaunch

	arcadeIntroAt  time.Time
	arcadeIntroBar int
}

type detailState struct {
	versionDirty bool
	versions     []launchEntry
	versionY     int

	dirty         bool
	clip          image.Rectangle
	text          []paneLine
	cols, entries int

	scroll int           // first information line wanted at the top
	pixel  int           // where the information sits now, in pixels, easing toward scroll
	next   time.Time     // the next animation frame
	last   time.Time     // previous scroll sample
	carry  time.Duration // fractional pixel travel between frames
	lines  int           // visible information lines, used for paging
	pick   int           // launch entry cursor
	// pickPath is the path of the chosen entry: a rescan can reorder the
	// versions, and the choice follows the file rather than the position.
	pickPath string
	from     Screen // where B returns to
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
	a := &App{cfg: cfg, body: fonts.Body(), sm: fonts.Small(), narrow: fonts.Narrow(), tall: fonts.NarrowTall(), rot: cfg.Rotation, down: map[platform.Key]bool{}, released: map[platform.Key]time.Time{}, look: DefaultSaverLook, supporters: defaultSupporters()}
	a.viewsOff = parseViewsOff(cfg.ViewsOff)
	a.mode = a.DefaultView()
	if cfg.RememberSort && a.viewOn(cfg.LastSort) {
		a.mode = cfg.LastSort
	}
	a.physical = image.NewRGBA(image.Rect(0, 0, cfg.PhysW, cfg.PhysH))
	a.setRotation(cfg.Rotation)
	a.SetData(ds, stored)
	a.saver.lastInput = cfg.TimerNow()
	return a
}

func (a *App) setRotation(rot gfx.Rotation) {
	a.layoutMotion = layoutMotion{}
	a.listMotion = listMotion{}
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

// SetCanvasSize rebuilds display-dependent surfaces without losing navigation.
func (a *App) SetCanvasSize(w, h int) {
	a.SaverSettle()
	a.saver = screensaver{lastInput: a.cfg.TimerNow()}
	a.cfg.Images.SetPaused(false)
	a.transition = pageTransition{enabled: a.transition.enabled}
	a.invalidateDetailScroll()
	a.splash = startupSplash{}
	a.cfg.PhysW, a.cfg.PhysH = w, h
	a.physical = image.NewRGBA(image.Rect(0, 0, w, h))
	a.setRotation(a.rot)
	switch a.screen {
	case ScreenOptions, ScreenSaverOptions, ScreenViews, ScreenCredits, ScreenFilter:
		a.buildPanel()
	}
}

// RestoreCanvasChoice reverts a failed host display change without requesting it again.
func (a *App) RestoreCanvasChoice(choice string) {
	a.cfg.Canvas = choice
	a.buildPanel()
	a.all = true
	a.settingsChanged()
}

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
	a.listMotion = listMotion{}
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
	if len(filters.SrcHidden) > 0 || filters.HideDeprecated || filters.ArcadeOnly {
		a.total = 0
		for i := range a.ds.Rows {
			if a.catalogueIncludes(&a.ds.Rows[i]) {
				a.total++
			}
		}
	}
	a.view = data.Apply(a.ds, a.order, &filters, a.cfg.Status, fav, unseen, a.romState)
	if q := searchText(a.query); q != "" {
		matched := a.view[:0]
		for _, i := range a.view {
			if strings.Contains(searchText(a.ds.Der[i].Title), q) {
				matched = append(matched, i)
			}
		}
		a.view = matched
	}
	a.marker = a.seen != nil
	a.sinceAdded, a.sinceUpdated = a.seen.Since(a.ds, a.catalogueIncludes)
	a.rebuildMarks()
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
	a.listMotion = listMotion{}
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

// MoveToKey puts the cursor on a key if it is in view, centred on the
// page the way a jump lands: the launcher reopens the app on the game
// that just ran, and a row pinned to the bottom edge looks like a
// half-restored list.
func (a *App) MoveToKey(k string) {
	a.moveToKey(k)
	if a.cursor < len(a.view) {
		a.top = centeredTop(a.screenLine(a.cursor), a.totalLines(), a.lay.Lines)
		a.shortPage = false
	}
	a.all = true
}

// SetSort switches the sort mode; a view turned off in Options -> Views
// is refused.
func (a *App) SetSort(m data.SortMode) {
	if !a.viewOn(m) || m == a.mode {
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

func (a *App) RememberSort() bool { return a.cfg.RememberSort }
func (a *App) OpenAtBoot() bool   { return a.cfg.OpenAtBoot }

// MenuButton is the Options -> Menu button choice: "options" or "leave".
func (a *App) MenuButton() string {
	if a.cfg.MenuButton == "leave" {
		return "leave"
	}
	return "options"
}

// Canvas is the Options -> Canvas choice: "fit", "full" or "320x240".
func (a *App) Canvas() string {
	if a.cfg.Canvas == "320x240" || a.cfg.Canvas == "fit" {
		return a.cfg.Canvas
	}
	return "full"
}
func (a *App) ReturnAfterGame() bool { return a.cfg.ReturnAfterGame }

// ExitChord is the Options -> Exit chord choice: "" (off), "select-start"
// or "lr-select-start"; anything else reads as off.
func (a *App) ExitChord() string {
	switch a.cfg.ExitChord {
	case "select-start", "lr-select-start":
		return a.cfg.ExitChord
	}
	return ""
}
func (a *App) FollowRotation() bool { return a.cfg.FollowRotation }
func (a *App) FilterRotation() bool { return a.cfg.FilterRotation }

// InstalledOnly reports Options -> Sources: installed only.
func (a *App) InstalledOnly() bool { return a.cfg.InstalledOnly }

func (a *App) ShowNonArcade() bool      { return a.cfg.ShowNonArcade }
func (a *App) ArcadeIntroPending() bool { return a.cfg.ArcadeIntro }

func (a *App) ShowDeprecated() bool { return a.cfg.ShowDeprecated }

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
	if a.cab.launched {
		a.cab.stopAhead()
		a.cab = launchCab{} // the launch failed: the page comes back with the notice
	}
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

// SetCatalogChecked records a successful check, even when the catalog is unchanged.
func (a *App) SetCatalogChecked(t time.Time) {
	a.catalogChecked = t
	if a.screen == ScreenOptions {
		a.buildPanel() // the Last checked row at the end of the list
	}
	a.all = true
}

// CatalogChecked is the last successful check in this session.
func (a *App) CatalogChecked() time.Time { return a.catalogChecked }

// Screen reports the current view.
func (a *App) Screen() Screen { return a.screen }

// screenLine maps a view position to its list line, counting marker lines.
func (a *App) screenLine(pos int) int {
	return pos + a.marksBefore(pos)
}

// totalLines is how many list lines the view occupies, markers included.
func (a *App) totalLines() int {
	return len(a.view) + len(a.marks)
}

func (a *App) ensureVisible() {
	if len(a.view) == 0 {
		a.top = 0
		return
	}
	line := a.screenLine(a.cursor)
	if a.markAt(a.cursor) && a.groupHeaders() {
		line-- // a group header comes into view with its first row
	}
	if line < a.top {
		a.top = line
	}
	last := a.screenLine(a.cursor)
	if a.markAt(a.cursor + 1) {
		last++ // keep the marker after the row reachable
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
	ev = a.padEvent(ev) // a pad whose OK button is B trades Enter and back
	if a.handleLaunchCab(ev) {
		return true
	}
	if a.handleSaverInput(ev) {
		return true
	}
	if a.cfg.ArcadeIntro {
		return a.handleArcadeIntro(ev)
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
	if a.bounced(ev) {
		return false
	}
	if ev.Key == platform.KeyMenu && a.MenuButton() != "leave" && !(a.screen == ScreenTroubleshooting && a.support.mode == "pad") {
		// Options mode: the tap acts on the release, a hold quits (menu.go);
		// the pad tester only logs the button
		if ev.Pressed {
			if a.down[ev.Key] {
				return false
			}
			a.down[ev.Key] = true
			a.menuPress(ev.At)
			return false
		}
		delete(a.down, ev.Key)
		return a.menuRelease(ev.At)
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
		if ev.Key == platform.KeyStart && a.releaseHoldLaunch() {
			return true
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
	if a.screen == ScreenList || a.screen == ScreenFilter || a.screen == ScreenOptions || a.screen == ScreenViews || a.screen == ScreenCredits || a.screen == ScreenSaverOptions {
		switch ev.Key {
		case platform.KeyUp, platform.KeyDown, platform.KeyLeft, platform.KeyRight:
			a.rep.next = ev.At.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
		case platform.KeyPageUp, platform.KeyPageDown, platform.KeyBackspace:
			if a.screen == ScreenList {
				a.rep.next = ev.At.Add(time.Duration(a.HoldDelay()) * time.Millisecond)
			}
		}
	}
	return a.act(ev.Key)
}

// debounce is the window after a key comes up in which a new press of the
// same key is taken for contact bounce and dropped. Worn arcade
// microswitches deliver a second press 2-8 ms after the release; the
// quickest deliberate double tap is several times longer.
const debounce = 25 * time.Millisecond

// bounced reports whether ev is a press that follows this key's release too
// closely to be a new tap; releases record their time on the way through.
// Only the press side is guarded: a bounce during a hold ends the hold,
// which costs one repeat delay, never a wrong action.
func (a *App) bounced(ev platform.Event) bool {
	if ev.Key == platform.KeyNone || ev.Key == platform.KeyOther {
		return false
	}
	at := ev.At
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	if ev.Pressed {
		// strictly after the release: scripted input (tests, the harness,
		// injected keys) may stamp a press and its release alike, a switch
		// never does
		up := a.released[ev.Key]
		return !a.down[ev.Key] && at.After(up) && at.Before(up.Add(debounce))
	}
	a.released[ev.Key] = at
	return false
}

// repeatStep says whether a held key repeats on the current screen and how
// fast; 0 means it does not.
func (a *App) repeatStep(k platform.Key, count int) time.Duration {
	switch a.screen {
	case ScreenList:
		switch k {
		case platform.KeyBackspace:
			return repeatErase
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
	case ScreenFilter, ScreenOptions, ScreenViews, ScreenCredits, ScreenSaverOptions:
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
	changed := a.tickPageTransition(now)
	changed = a.tickHoldLaunch(now) || changed
	changed = a.tickLaunchCab(now) || changed
	changed = a.tickLayoutMotion(now) || changed
	changed = a.tickSplash(now) || changed
	changed = a.tickMenu(now) || changed
	changed = a.tickArcadeIntro(now) || changed
	changed = a.tickOptionSamples() || changed
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
	changed := a.tickPageTransition(now)
	changed = a.tickHoldLaunch(now) || changed
	changed = a.tickLaunchCab(now) || changed
	changed = a.tickLayoutMotion(now) || changed
	changed = a.tickSplash(now) || changed
	changed = a.tickMenu(now) || changed
	changed = a.tickArcadeIntro(now) || changed
	changed = a.OptionSampleFrame() || changed
	if k := a.rep.frameDue(now, a.repeatStep); k != platform.KeyNone {
		oldTop, oldScreen := a.top, a.screen
		if a.act(k) {
			changed = true
		}
		if oldScreen == ScreenList && a.screen == ScreenList && (k == platform.KeyUp || k == platform.KeyDown) && a.rep.every > 1 && a.SmoothScrolling() {
			a.listMotion = listMotion{offset: (a.top - oldTop) * a.lay.Line, frames: a.rep.every}
		}
	}
	changed = a.ListScrollFrame(now) || changed
	changed = a.DetailScrollFrame(now) || changed // follows the display cadence under a held key
	if a.notice != "" && !now.Before(a.until) {
		a.notice = ""
		a.all = true
		changed = true
	}
	return changed
}

// NextTick reports when Tick next needs to run; zero when nothing is pending.
func (a *App) NextTick() time.Time {
	// Enter the preview frame loop immediately, including after releasing a key.
	if a.OptionSamplesRunning() || a.ListScrollRunning() || a.LayoutTransitionRunning() {
		return a.cfg.TimerNow()
	}
	t := a.rep.nextAt()
	if a.screen == ScreenTroubleshooting {
		t = a.support.next
		if v := &a.support; v.mode == "pad" && !v.backAt.IsZero() {
			t = a.nextHoldPixel(v.backAt, padTestLeave, v.holdBar)
		}
	}
	if next := a.nextUpdateTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
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
	if next := a.nextArcadeIntroTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.nextMenuTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.splash.next; !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.transition.next; !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.nextLaunchCabTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	if next := a.nextHoldLaunchTick(); !next.IsZero() && (t.IsZero() || next.Before(t)) {
		t = next
	}
	return t
}

// act performs a key on the current screen.
func (a *App) act(k platform.Key) bool {
	if a.listMotion.offset != 0 {
		a.all = true
	}
	a.listMotion = listMotion{}
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
	case ScreenFilter, ScreenOptions, ScreenViews, ScreenCredits, ScreenSaverOptions:
		return a.actPanel(k)
	case ScreenCalibrate:
		return a.actCalibrate(k)
	}
	return false
}

func (a *App) actList(k platform.Key) bool {
	n := len(a.view)
	if a.quickHeld() && (k == platform.KeyLeft || k == platform.KeyRight) {
		return a.rotateBy(map[platform.Key]int{platform.KeyLeft: -1, platform.KeyRight: 1}[k])
	}
	if a.quickHeld() && k != platform.KeySelect && k != platform.KeySpace && k != platform.KeyTab && k != platform.KeyEnter {
		return false // Select held: only the chords act (quick.go), nothing moves or launches
	}
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
	case platform.KeySelect: // held, Y, X and Left/Right become the quick toggles (quick.go)
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
		if a.quickHeld() {
			a.toggleFavorite()
			return true
		}
		if n > 0 {
			a.screen = ScreenDetails
			a.detail = detailState{from: ScreenList, pick: a.rememberedPick()}
			if row, _, _ := a.current(); row != nil && a.detail.pick > 0 {
				a.detail.pickPath = a.cfg.Versions[row.K]
			}
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
	if a.PageTransitionRunning() {
		a.transition.pending = true
		return
	}
	a.all = true
	if a.screen == ScreenOptions || a.screen == ScreenSaverOptions {
		a.buildPanel() // the prefetch tally and screenshot availability
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
	if box.Empty() {
		return // the text layout shows no pictures
	}
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
	if a.cab.active {
		a.all = true
		if a.cab.ahead != nil {
			if f := a.cab.ahead.frame(a.cab.elapsed); f != nil {
				a.logical.TakeDirty()
				return f, []image.Rectangle{f.Rect}
			}
		}
		a.paintLaunchCab(a.logical)
		return a.rotatePaint()
	}
	a.validateLayoutMotion()
	if !a.all && a.LayoutTransitionRunning() && a.layoutMotion.dirty {
		a.wants = a.wants[:0]
		a.logical.Fill(a.lay.Body, gen.Eva.Bg)
		a.paintLayoutMotion(a.logical)
		return a.rotatePaint()
	}
	if a.screen != ScreenList || a.saver.active {
		a.listMotion = listMotion{}
	}
	if !a.all && a.listMotion.dirty && !a.PageTransitionRunning() {
		a.paintRows(a.logical)
		a.listMotion.dirty = false
		return a.rotatePaint()
	}
	a.listMotion.dirty = false
	if !a.all {
		if (a.detail.dirty || a.detail.versionDirty) && a.screen == ScreenDetails && !a.saver.active && !a.PageTransitionRunning() {
			if a.detail.dirty {
				a.paintDetailInfo(a.logical)
			}
			if a.detail.versionDirty || (a.detail.dirty && a.marquee.key != "") {
				a.paintDetailVersion(a.logical)
			}
			a.detail.dirty, a.detail.versionDirty = false, false
			return a.rotatePaint()
		}
		if !a.PageTransitionRunning() {
			return a.physical, nil
		}
		copy(a.logical.Pix, a.transition.to)
		a.paintPageTransition(false)
		return a.rotatePaint()
	}
	a.preparePageTransition()
	a.detail.dirty, a.detail.versionDirty = false, false
	a.all = false
	a.wants = a.wants[:0]
	if a.screen != ScreenDetails {
		a.marquee = marqueeState{} // a return to Details starts its scroll afresh
	}
	c := a.logical
	if a.saver.active && a.saver.shots != nil {
		a.paintSaverShots(c) // its own pictures: the screen under it is not painted
	} else if a.saver.active && (a.saverCached(c) || (a.saver.style == "dim" && len(a.saver.dark) == len(c.Pix))) {
		// the saver shows the picture it froze on its first frame; the
		// screen under it is painted again when a key wakes the app
		a.paintSaver(c)
	} else {
		c.Fill(c.Rect, gen.Eva.Bg)
		switch a.screen {
		case ScreenList:
			a.paintList(c)
		case ScreenDetails:
			a.paintDetails(c)
		case ScreenShot:
			a.paintShot(c)
		case ScreenFilter, ScreenOptions, ScreenViews, ScreenCredits, ScreenSaverOptions:
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
		if a.cfg.ArcadeIntro {
			a.paintArcadeIntro(c)
		}
		if a.saver.active {
			a.paintSaver(c)
		}
	}
	a.paintSplash()
	if a.PageTransitionRunning() {
		a.transition.to = append(a.transition.to[:0], c.Pix...)
	}
	a.paintPageTransition(true)
	return a.rotatePaint()
}

func (a *App) rotatePaint() (*image.RGBA, []image.Rectangle) {
	c := a.logical
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
		out = append(out, data.GateStage(r))
	}
	if r.Deprecated {
		out = append(out, "deprecated")
	}
	if r.IsLocal() {
		out = append(out, "local")
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

func (a *App) SaverFilters() (bool, bool, bool, string) {
	return a.cfg.SaverCard, a.cfg.SaverRotation, a.cfg.SaverFavorites, a.cfg.SaverResolution
}
