//go:build linux

// misterzine is the device binary: it runs the app on the MiSTer's
// framebuffer from the main-menu launcher. See deploy/launch.sh.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/debugsrv"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/images"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/mister"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"github.com/matijaerceg/misterzine-on-device/internal/snapshot"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

const (
	canvasW, canvasH = 320, 240
	logMax           = 1 << 20
)

type host struct {
	root, card      string
	lg              *log.Logger
	console         *mister.Console
	cmd             *mister.Cmd
	fb              *mister.FB
	input           *mister.Input
	a               *app.App
	settings        store.Settings
	state           store.State
	favs            store.Favorites
	events          chan platform.Event
	uiRun           chan func()
	quit            chan struct{}
	launch          string
	dirty           bool // state needs saving
	stopRequested   atomic.Bool
	lostCh          chan struct{} // the screen-lost probe fired
	debugEnabled    bool          // input logging and frame performance measurements
	lastEv          time.Time
	checkFailed     atomic.Bool
	dataUpdated     time.Time // the data's build time, for the clock check
	slowLog         time.Time
	stats           frameStats
	favDirty        bool
	favLoadFailed   bool // preserve a favorites file we could not read
	saveAt          time.Time
	saveRetry       bool
	setDirty        bool
	clock           platform.Clock
	client          *fetch.Client
	img             *images.Service
	index           *scan.Index
	status          []data.Status
	alts            []scan.Alt
	scanCh          chan scanResult
	timeSample      atomic.Pointer[serverClockSample]
	checkRunning    bool // UI-owned; held until the result is installed
	checkPending    bool
	nextCheck       time.Time
	scanRunning     bool
	scanPending     bool
	netCh           chan string
	updates         chan updateResult
	updatePending   bool
	updateReadError string // UI-owned; suppress repeated status-read diagnostics
	updateRunning   bool
	troubleshooting supportHost
}

func main() {
	// fewer collections (the decoder allocates a lot), with a hard cap
	debug.SetGCPercent(400)
	debug.SetMemoryLimit(96 << 20)
	// ahead of Main and the resident services when a frame is due
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, -10)
	if len(os.Args) > 1 && os.Args[1] == "console-restore" {
		mister.RestoreAll()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "launcher" {
		os.Exit(launcherCmd(os.Args[2:]))
	}
	if len(os.Args) == 5 && os.Args[1] == "update-worker" {
		os.Exit(updater.Worker(os.Args[2], os.Args[3], os.Args[4]))
	}
	root := flag.String("root", "/media/fat/misterzine", "config directory")
	card := flag.String("card", "/media/fat", "card root")
	ini := flag.String("ini", "/media/fat/MiSTer.ini", "MiSTer.ini path")
	debugAddr := flag.String("debug-http", "", "LAN debug server address, e.g. :8195")
	version := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *version {
		fmt.Println("misterzine " + buildinfo.String())
		return
	}
	os.Exit(run(*root, *card, *ini, *debugAddr))
}

func run(root, card, iniPath, debugAddr string) (code int) {
	os.MkdirAll(filepath.Join(root, "cache"), 0755)
	lg := openLog(filepath.Join(root, "log.txt"))
	lg.Printf("==== misterzine %s pid %d args %v", buildinfo.String(), os.Getpid(), os.Args[1:])
	t0 := time.Now()
	h := &host{root: root, card: card, lg: lg, events: make(chan platform.Event, 256), uiRun: make(chan func(), 8), quit: make(chan struct{}), netCh: make(chan string, 4), scanCh: make(chan scanResult, 1)}

	// settings, state, favorites
	var serr error
	if h.settings, serr = store.LoadSettings(filepath.Join(root, "settings.json")); serr != nil && !errors.Is(serr, os.ErrNotExist) {
		lg.Printf("settings: %v", serr)
	}
	hasState := true
	if err := store.Load(filepath.Join(root, "state.json"), &h.state); err != nil {
		hasState = false
		if !errors.Is(err, os.ErrNotExist) {
			lg.Printf("state: %v", err)
		}
	}
	h.loadFavorites()

	ini := mister.ReadIni(iniPath)
	lg.Printf("ini: found=%v osd_rotate=%d direct_video=%d vga_scaler=%d fb_terminal=%d analog-visible=%v",
		ini.Found, ini.OSDRotate, ini.DirectVideo, ini.VGAScaler, ini.FBTerminal, ini.AnalogVisible())
	if ini.Found && !ini.AnalogVisible() {
		lg.Printf("WARNING: the framebuffer cannot reach the analog port with this MiSTer.ini; on a CRT-only setup add direct_video=1 (or vga_scaler=1 + a 15 kHz video_mode) under a [Menu] section")
	}
	rotation := gfx.RotNone
	switch h.settings.Rotation {
	case "left":
		rotation = gfx.RotLeft
	case "right":
		rotation = gfx.RotRight
	case "off":
	default: // auto
		switch ini.OSDRotate {
		case 1:
			rotation = gfx.RotRight
		case 2:
			rotation = gfx.RotLeft
		}
	}

	// data: cache, else the embedded snapshot
	rows, meta, source := h.loadData()
	upd, _ := data.ParseMetaTime(meta.Updated)
	now := time.Now()
	trusted := !upd.IsZero() && now.After(upd.Add(-24*time.Hour))
	h.clock = platform.Clock{Now: h.now, Trusted: trusted}
	h.dataUpdated = upd
	ds := data.Ingest(rows, meta.Hash, upd)
	lg.Printf("data: %d rows from %s, hash %.8s, updated %s, clock trusted=%v (%v)", len(rows), source, meta.Hash, meta.Updated, trusted, time.Since(t0).Round(time.Millisecond))

	// platform
	var err error
	if h.console, err = mister.AcquireConsole(lg); err != nil {
		fmt.Fprintln(os.Stderr, "misterzine:", err)
		lg.Printf("console: %v", err)
		return 2
	}
	defer h.cleanup()
	defer func() {
		if r := recover(); r != nil {
			lg.Printf("PANIC: %v\n%s", r, debug.Stack())
			h.cleanup()
			code = 70
		}
	}()
	if h.cmd, err = mister.NewCmd(lg); err != nil {
		lg.Printf("cmd: %v", err)
	}
	if h.fb, err = mister.OpenFB(h.cmd, canvasW, canvasH, lg); err != nil {
		lg.Printf("fb: %v", err)
		return 3
	}
	h.input = mister.OpenInput(lg)
	go func() {
		for ev := range h.input.Events() {
			h.events <- ev
		}
	}()
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// pictures
	h.client = fetch.NewClient(buildinfo.Version)
	h.img = images.New(filepath.Join(root, "shots"), h.client, lg, 24<<20)
	h.img.SetPrefetch(picsFor(ds), h.settings.Prefetch)

	// app
	favSet := h.favs.Set()
	cfg := app.Config{
		PhysW: canvasW, PhysH: canvasH, Rotation: rotation, SafeInsetX: h.settings.InsetX, SafeInsetY: h.settings.InsetY,
		Now: h.now, TimerNow: time.Now, ClockTrusted: trusted, Favorites: favSet, Images: h.img, Scroll: h.settings.Scroll, HoldDelay: h.settings.HoldDelay,
		Screensaver:          h.settings.Screensaver,
		RememberSort:         h.settings.RememberSort,
		LastSort:             h.settings.LastSort,
		FavoritesUnavailable: h.favLoadFailed,
		Progress:             func() (int, int) { return h.img.Progress() },
		Launcher:             launcherEnabled,
		Status: func(i int) data.Status {
			if i < len(h.status) {
				return h.status[i]
			}
			return data.StatusUnknown
		},
		Alternatives: func(r *data.Row) []string { return scan.Alternatives(h.alts, r) },
		Exists: func(rel string) bool {
			_, err := os.Stat(filepath.Join(card, filepath.FromSlash(rel)))
			return err == nil
		},
		Launch:          h.requestLaunch,
		Quit:            h.stop,
		Version:         buildinfo.String(),
		Support:         h.supportHooks(),
		FavChanged:      func() { h.favDirty = true },
		FiltersChanged:  func() { h.dirty = true },
		SettingsChanged: func() { h.setDirty = true },
		Action: func(kind, arg string) {
			lg.Printf("action: %s %s", kind, arg)
			if kind == "update" {
				h.startUpdate()
				return
			}
			if kind == "update-cancel" {
				go h.cancelUpdate(arg)
				return
			}
			if kind == "update-dismiss" {
				h.dismissUpdate(arg)
				return
			}
			if kind == "rotation" {
				h.settings.Rotation = arg
				h.setDirty = true
			}
			if kind == "refresh" {
				h.requestCheck()
			}
			if kind == "rescan" {
				h.requestScan()
			}
			if kind == "prefetch" {
				h.settings.Prefetch = arg == "on"
				h.setDirty = true
				h.img.SetPrefetch(picsFor(h.a.Data()), h.settings.Prefetch)
			}
			if kind == "clearimg" {
				go h.img.ClearCache()
			}
			if kind == "launcher" {
				if arg == "on" {
					if err := launcherEnable(); err != nil {
						lg.Printf("launcher: %v", err)
						h.a.Notice("could not enable: "+err.Error(), 6*time.Second)
						return
					}
					launcherStart()
				} else {
					if err := launcherDisable(); err != nil {
						lg.Printf("launcher: %v", err)
						h.a.Notice("could not disable: "+err.Error(), 6*time.Second)
						return
					}
					launcherStop()
				}
			}
		},
	}
	var seen *data.SeenRecord
	if hasState && (len(h.state.Seen.Cur) > 0 || h.state.Seen.T != "") {
		seen = &h.state.Seen
	}
	h.a = app.New(cfg, ds, seen)
	h.dirty = true // persist the new visit even without input
	h.initUpdates()
	h.a.SetPrefetch(h.settings.Prefetch)
	// Keep the user's filter choices alongside the startup sort preference.
	h.a.SetFilters(h.state.Filters)
	h.a.SetNet(h.netLabel(ds))
	if ini.Found && !ini.AnalogVisible() && !hasState { // first run only: HDMI users need nothing
		h.a.Notice("CRT only? add direct_video=1 under [Menu], see README", 20*time.Second)
	}
	if h.favLoadFailed {
		h.a.Notice(app.FavoritesUnavailableNotice, 12*time.Second)
	}
	h.present()
	lg.Printf("first frame at %v", time.Since(t0).Round(time.Millisecond))

	h.debugEnabled = debugAddr != ""
	h.fb.MeasureTiming = h.debugEnabled
	if debugAddr != "" {
		if debugsrv.Serve(debugAddr, debugsrv.Hooks{
			Run:    h.runOnUI,
			Inject: func(ev platform.Event) { h.events <- ev },
			Shot:   func() *image.RGBA { return h.a.Logical() },
			State: func() any {
				return map[string]any{
					"version": buildinfo.String(), "screen": h.a.Screen().String(), "cursor": h.a.CursorKey(),
					"sort": h.a.Sort().String(), "rows": len(h.a.Data().Rows), "fb": h.fb.Geometry().String(),
					"search": h.a.Search(), "filters": h.a.Filters(), "rotation": h.a.Rotation().String(), "inset": fmt.Sprint(h.a.Inset()), "devices": h.input.Devices(),
					"sysfs": mister.SysfsMode(), "uptime": time.Since(t0).String(), "frames": h.stats.String(),
					"update":      h.a.UpdateState(),
					"screensaver": h.a.Screensaver(), "screensaver_active": h.a.ScreensaverActive(),
				}
			},
			Quit: h.stop,
			Goto: func(k string) { h.a.MoveToKey(k); h.present() },
			Log:  filepath.Join(root, "log.txt"),
		}, lg) {
			h.a.Notice("Remote debug enabled", 8*time.Second)
		}
	}

	// the pad's menu button: Main takes the screen back and grabs the
	// input devices; a probe every quarter second notices (it costs tens of
	// milliseconds, so it runs off the UI goroutine)
	h.lostCh = make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-h.quit:
				return
			case <-time.After(250 * time.Millisecond):
			}
			if h.input.ScreenLost() {
				select {
				case h.lostCh <- struct{}{}:
				default:
				}
				return
			}
		}
	}()

	// card scan: cores, MRA presence, then alternatives
	h.requestScan()

	// freshness: first check shortly after boot, then every 30 minutes;
	// while the clock is unset (the first seconds after a cold boot, before
	// NTP) or the last check failed, every 15 seconds instead
	go func() {
		wait := 300 * time.Millisecond
		for {
			select {
			case <-h.quit:
				return
			case <-time.After(wait):
			}
			h.runOnUI(func() {
				if !time.Now().Before(h.nextCheck) {
					h.requestCheck()
				}
			})
			wait = 15 * time.Second
		}
	}()

	// the loop
	for {
		var tick <-chan time.Time
		next := h.a.NextTick()
		wait := 250 * time.Millisecond
		if !next.IsZero() {
			if d := time.Until(next); d < wait {
				wait = d
			}
		}
		if wait < 0 {
			wait = 0
		}
		tick = time.After(wait)
		select {
		case <-h.quit:
			h.saveAll(true)
			return 0
		case s := <-sig:
			lg.Printf("signal %v", s)
			h.stop()
			h.saveAll(true)
			return 130
		case ev := <-h.events:
			h.handleEvent(ev)
			h.drainEvents()
		case f := <-h.uiRun:
			f()
		case u := <-h.updates:
			h.receiveUpdate(u)
		case s := <-h.netCh:
			h.a.SetNet(s)
			h.img.SetOffline(s == "no connection")
		case <-h.img.Ready():
			h.a.Invalidate()
		case <-h.img.ProgressReady():
			if h.a.Screen() == app.ScreenOptions {
				h.a.Invalidate()
			}
		case r := <-h.scanCh:
			h.receiveScan(r)
		case <-h.lostCh:
			lg.Printf("Main took the screen back (menu button): leaving")
			h.stop()
			continue
		case <-tick:
		}
		h.a.Tick(time.Now())
		h.present()
		if h.a.Repeating() {
			// a held key: run at the framebuffer's pace until it is released,
			// nothing else (saves, checks) gets between two frames
			h.frameLoop()
		}
		h.autosave(time.Now(), false)
		if h.launch != "" {
			h.saveAll(true)
			return h.doLaunch()
		}
	}
}

// handleEvent feeds one input event to the app (the screenshot key is the
// host's own).
func (h *host) handleEvent(ev platform.Event) {
	h.recordSupportInput(ev)
	if h.debugEnabled {
		h.lg.Printf("input: %v %s +%dms %s", ev.Key, map[bool]string{true: "down", false: "up"}[ev.Pressed], ev.At.Sub(h.lastEv).Milliseconds(), ev.Source)
		h.lastEv = ev.At
	}
	if ev.Key == platform.KeyScreenshot {
		if ev.Pressed {
			h.screenshot()
		}
		return
	}
	h.a.Handle(ev)
}

func (h *host) drainEvents() {
	for {
		select {
		case ev := <-h.events:
			h.handleEvent(ev)
		default:
			return
		}
	}
}

// frameLoop runs while a key is held: every vertical blank, take the input
// that arrived, move at most one step, paint, and copy the frame straight
// in (the wait for vsync already happened). Pictures that land are painted
// within the same frame budget; nothing else runs.
func (h *host) frameLoop() {
	// the decoder shares the memory bus and the GC with us: quiet it once
	// the key really repeats (a tap leaves it working on the neighbours)
	paused := false
	var t0 time.Time
	if h.debugEnabled {
		t0 = time.Now()
	}
	iters, p0, late := 0, h.stats.n, 0
	var longWait time.Duration
	longWaits := 0
	defer func() {
		if paused {
			h.img.SetPaused(false)
		}
		h.a.Invalidate()
		if d := time.Since(t0); h.debugEnabled && d > time.Second {
			h.lg.Printf("frame loop: %d frames in %s (%.1f/s), %d presents, %d over budget, %d vsync waits over 20ms (max %s)", iters, d.Round(time.Millisecond), float64(iters)/d.Seconds(), h.stats.n-p0, late, longWaits, longWait.Round(100*time.Microsecond))
		}
	}()
	for h.a.Repeating() {
		iters++
		h.drainEvents()
		select {
		case <-h.quit:
			return
		default:
		}
		if !h.a.Repeating() {
			return
		}
		// paint first, then copy right after the vertical blank: the copy
		// runs ahead of the beam, so the single buffer never tears
		t := time.Now()
		h.a.Frame(t)
		if !paused && h.a.RepeatActive() {
			paused = true
			h.img.SetPaused(true)
		}
		frame, dirty := h.a.Paint()
		var paint time.Duration
		if h.debugEnabled {
			paint = time.Since(t)
			t = time.Now()
		}
		h.fb.WaitVSync()
		if h.debugEnabled {
			w := time.Since(t)
			longWait = max(longWait, w)
			if w > 20*time.Millisecond {
				longWaits++
			}
		}
		if dirty != nil {
			if h.debugEnabled {
				t = time.Now()
			}
			h.fb.PresentWait(frame, dirty, false)
			if h.debugEnabled {
				cp := time.Since(t)
				h.stats.add(paint, 0, cp)
				if paint+cp > 16*time.Millisecond {
					late++
				}
			}
		}
		select {
		case <-h.lostCh:
			h.lg.Printf("Main took the screen back (menu button): leaving")
			h.stop()
			return
		default:
		}
	}
}

// closeQuit elects one shutdown caller without racing concurrent API/input exits.
func (h *host) closeQuit() bool {
	if !h.stopRequested.CompareAndSwap(false, true) {
		return false
	}
	close(h.quit)
	return true
}

func (h *host) stop() {
	if h.closeQuit() {
		// watchdog: whatever else happens, the console comes back
		go func() {
			time.Sleep(3 * time.Second)
			h.lg.Printf("shutdown watchdog fired: forcing exit")
			mister.RestoreAll()
			os.Exit(75)
		}()
	}
}

func (h *host) runOnUI(f func()) {
	done := make(chan struct{})
	select {
	case h.uiRun <- func() { f(); close(done) }:
	case <-h.quit:
		return
	}
	select {
	case <-done:
	case <-h.quit:
	}
}

func (h *host) present() {
	if !h.debugEnabled {
		frame, dirty := h.a.Paint()
		if dirty != nil {
			h.fb.Present(frame, dirty)
		}
		return
	}
	t0 := time.Now()
	frame, dirty := h.a.Paint()
	if dirty != nil {
		t1 := time.Now()
		h.fb.Present(frame, dirty)
		h.stats.add(t1.Sub(t0), h.fb.LastWait, time.Since(t1)-h.fb.LastWait)
		if d := time.Since(t0); d > 40*time.Millisecond && time.Since(h.slowLog) > 5*time.Second {
			h.slowLog = time.Now()
			h.lg.Printf("slow frame: paint %s, present %s", t1.Sub(t0).Round(time.Millisecond), time.Since(t1).Round(time.Millisecond))
		}
	}
}

func (h *host) loadFavorites() {
	var err error
	h.favs, err = store.LoadFavorites(filepath.Join(h.root, "favorites.json"))
	h.favLoadFailed = err != nil && !errors.Is(err, os.ErrNotExist)
	if h.favLoadFailed {
		h.lg.Printf("favorites: %v; editing and saving disabled for this session", err)
	}
}

// autosave runs only on the UI loop, outside held-key rendering. All kinds of
// pending edits arm it; a failed write remains pending and retries at a bounded
// pace, even if navigation continues.
func (h *host) autosave(now time.Time, stateChanged bool) {
	if stateChanged {
		h.dirty = true
		if !h.saveRetry {
			h.saveAt = now.Add(500 * time.Millisecond)
		}
	}
	if !h.pendingSave() {
		return
	}
	if h.saveAt.IsZero() {
		h.saveAt = now.Add(500 * time.Millisecond)
	}
	if now.Before(h.saveAt) {
		return
	}
	h.saveAll(false)
	h.saveRetry = h.pendingSave()
	if h.saveRetry {
		h.saveAt = now.Add(5 * time.Second)
	} else {
		h.saveAt = time.Time{}
	}
}

func (h *host) pendingSave() bool {
	return h.dirty || h.setDirty || (h.favDirty && !h.favLoadFailed)
}

func (h *host) saveAll(final bool) {
	if h.dirty || final {
		st := store.State{Schema: 1,
			LastOpen: h.now().UTC().Format(time.RFC3339), DataHash: h.a.Data().Hash, Filters: h.a.Filters()}
		if s := h.a.Seen(); s != nil {
			st.Seen = s.State
		}
		if err := store.Save(filepath.Join(h.root, "state.json"), st); err != nil {
			h.lg.Printf("state: %v", err)
			h.dirty = true
		} else {
			h.dirty = false
		}
	}
	if !h.favLoadFailed && (h.favDirty || final) {
		h.favs.Apply(h.a.FavoriteSet(), h.now())
		if err := store.Save(filepath.Join(h.root, "favorites.json"), h.favs); err != nil {
			h.lg.Printf("favorites: %v", err)
			h.favDirty = true
		} else {
			h.favDirty = false
		}
	}
	if h.setDirty || final {
		h.settings.InsetX, h.settings.InsetY = h.a.Inset()
		h.settings.Inset = h.settings.InsetX
		h.settings.Scroll = h.a.ScrollSpeed()
		h.settings.HoldDelay = h.a.HoldDelay()
		h.settings.Screensaver = h.a.Screensaver()
		h.settings.RememberSort = h.a.RememberSort()
		h.settings.LastSort = h.a.Sort()
		if err := store.Save(filepath.Join(h.root, "settings.json"), h.settings); err != nil {
			h.lg.Printf("settings: %v", err)
			h.setDirty = true
		} else {
			h.setDirty = false
		}
	}
}

// cleanup restores the machine; safe to call twice.
func (h *host) cleanup() {
	if h.troubleshooting.capture != nil {
		h.finishSupport(true)
	}
	if h.input != nil {
		h.input.Close()
		h.input = nil
	}
	if h.fb != nil {
		h.fb.CloseRestore(h.cmd, h.launch == "")
		h.fb = nil
	}
	if h.console != nil {
		h.console.Restore()
		h.console = nil
	}
	if h.img != nil {
		h.img.Close()
		h.img = nil
	}

}

// requestLaunch validates while the app can still show an error. In particular,
// Start before a scan finishes must not turn a missing core into a silent exit.
func (h *host) requestLaunch(target string) {
	abs, err := mister.LaunchPath(h.card, target)
	message := "Cannot launch: target missing or invalid"
	if err == nil {
		message = "Cannot launch: MiSTer not ready"
		err = h.cmd.Available()
	}
	if err != nil {
		h.lg.Printf("launch %q: %v", target, err)
		h.supportLaunchResult(message, err)
		h.a.Notice(message, 8*time.Second)
		return
	}
	h.launch = abs
	h.supportLaunchResult("Target found; preparing launch", nil)
	h.stop()
}

func (h *host) doLaunch() int {
	// Restore the old framebuffer before Main starts writing a new core's frame.
	h.cleanup()
	marker := filepath.Join(h.root, "launched")
	if err := os.WriteFile(marker, []byte(h.launch+"\n"), 0644); err != nil {
		h.lg.Printf("launch marker: %v", err)
		h.supportLaunchResult("Could not prepare launch", err)
		return 1
	}
	if err := h.cmd.Send("load_core " + h.launch); err != nil {
		h.lg.Printf("launch: %v", err)
		h.supportLaunchResult("Launch command failed", err)
		os.Remove(marker)
		return 1
	}
	h.lg.Printf("launched %s", h.launch)
	h.supportLaunchResult("Command sent; game startup not verified", nil)
	return 0
}

func (h *host) loadData() ([]data.Row, data.Meta, string) {
	cache := filepath.Join(h.root, "cache")
	if b, err := os.ReadFile(filepath.Join(cache, "data.json")); err == nil {
		if rows, err := data.DecodeRows(bytes.NewReader(b)); err == nil && len(rows) > 0 {
			var meta data.Meta
			if mb, err := os.ReadFile(filepath.Join(cache, "meta.json")); err == nil {
				meta, _ = data.DecodeMeta(bytes.NewReader(mb))
			}
			return rows, meta, "cache"
		} else if err != nil {
			h.lg.Printf("cache: %v", err)
			os.Rename(filepath.Join(cache, "data.json"), filepath.Join(cache, "data.json.bad"))
		}
	}
	rows, meta, err := snapshot.Load()
	if err != nil {
		h.lg.Printf("snapshot: %v", err)
		return nil, data.Meta{}, "nothing"
	}
	return rows, meta, "snapshot"
}

// clockTrusted re-checks the clock: it starts at 1970 on a cold boot and
// jumps once NTP answers, at which point relative times and the network
// label make sense again.
func (h *host) clockTrusted() bool {
	if h.clock.Trusted {
		return true
	}
	if !h.dataUpdated.IsZero() && h.now().After(h.dataUpdated.Add(-24*time.Hour)) {
		h.trustClock()
		h.lg.Printf("clock: now set")
	}
	return h.clock.Trusted
}

func (h *host) netLabel(ds *data.Dataset) string {
	if ds.Updated.IsZero() || !h.clock.Trusted {
		return ""
	}
	return "data " + data.RelUpdated(h.now(), ds.Updated)
}

// check runs a freshness check off the UI goroutine.
func (h *host) check(current string, trusted bool) {
	defer h.runOnUI(h.finishCheck)
	h.sendNet("checking" + gfx.Ellipsis)
	if !trusted {
		// the clock is still unset: TLS will fail, but a plain HTTP answer
		// tells the time, so relative dates work before NTP does
		if t, err := h.client.ServerTime(context.Background()); err == nil {
			h.timeSample.Store(&serverClockSample{wall: t, local: time.Now()})
			h.lg.Printf("clock: from the site's Date header, %v ahead of the system clock", t.Sub(time.Now()).Round(time.Second))
			trusted = true
			h.runOnUI(h.trustClock)
		} else {
			h.lg.Printf("clock: %v", err)
		}
	}
	fr, err := h.client.Check(context.Background(), current)
	if err != nil {
		h.lg.Printf("check: %v", err)
		if !trusted {
			h.sendNet("") // the clock is not set yet: nothing to tell the user
		} else if errors.Is(err, fetch.ErrOffline) {
			h.sendNet("no connection")
		} else {
			h.sendNet("check failed")
		}
		h.checkFailed.Store(true)
		return
	}
	h.checkFailed.Store(false)
	if !fr.Changed {
		h.lg.Printf("check: current (%.8s)", fr.Meta.Hash)
		h.sendNet("")
		return
	}
	cache := filepath.Join(h.root, "cache")
	if err := store.WriteAtomic(filepath.Join(cache, "data.json"), fr.RawData); err != nil {
		h.lg.Printf("cache: %v (meta.json left as it was)", err)
	} else if err := store.WriteAtomic(filepath.Join(cache, "meta.json"), fr.RawMeta); err != nil {
		h.lg.Printf("cache: %v", err)
	}
	h.lg.Printf("check: new data %.8s, %d rows", fr.Meta.Hash, len(fr.Rows))
	h.runOnUI(func() { h.swap(fr) })
}

// swap installs fetched data on the UI goroutine.
func (h *host) swap(fr fetch.Fresh) {
	old := h.a.Data()
	upd, _ := data.ParseMetaTime(fr.Meta.Updated)
	ds := data.Ingest(fr.Rows, fr.Meta.Hash, upd)
	news := data.DiffNews(old, fr.Rows)
	// Old status indices belong to the previous row order. Rebuild off the UI.
	h.status = nil
	h.a.SetData(ds, nil)
	h.requestScan()
	h.img.SetPrefetch(picsFor(ds), h.settings.Prefetch)
	h.a.SetNet(h.netLabel(ds))
	if news != "" {
		h.a.Notice(news, 12*time.Second)
	}
	h.dirty = true
}

type scanResult struct {
	index  *scan.Index
	status []data.Status
	hash   string // the dataset the statuses index into
	alts   []scan.Alt
	notice string
	final  bool // alternatives pass finished, including an empty result
}

// scan reads the card off the UI goroutine: cores and MRA stats first (fast),
// alternatives after (slow the first time).
func (h *host) scan(rows []data.Row, hash string) {
	t0 := time.Now()
	idx := scan.ScanCores(h.card)
	st := scan.Statuses(h.card, idx, rows)
	counts := map[data.Status]int{}
	for _, s := range st {
		counts[s]++
	}
	h.lg.Printf("scan: %d cores; current %d, outdated %d, undated %d, not found %d (%v)", len(idx.Cores),
		counts[data.StatusCurrent], counts[data.StatusOutdated], counts[data.StatusFoundUndated], counts[data.StatusNotFound], time.Since(t0).Round(time.Millisecond))
	if !h.sendScan(scanResult{index: idx, status: st, hash: hash, notice: fmt.Sprintf("card: %d current, %d older, %d not found", counts[data.StatusCurrent], counts[data.StatusOutdated], counts[data.StatusNotFound])}) {
		return
	}
	t1 := time.Now()
	alts, err := scan.ScanAlternativesWithError(h.card, filepath.Join(h.root, "cache", "alts.json"))
	if err != nil {
		h.lg.Printf("scan: %v", err)
	}
	h.lg.Printf("scan: %d alternatives (%v)", len(alts), time.Since(t1).Round(time.Millisecond))
	h.sendScan(scanResult{index: idx, status: st, hash: hash, alts: alts, final: true})
}

// screenshot saves the logical canvas (F12 on a keyboard).
func (h *host) screenshot() {
	dir := filepath.Join(h.root, "screenshots")
	os.MkdirAll(dir, 0755)
	p := filepath.Join(dir, h.a.Screen().String()+"-"+time.Now().Format("20060102-150405")+".png")
	f, err := os.Create(p)
	if err != nil {
		h.lg.Printf("screenshot: %v", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, h.a.Logical()); err != nil {
		h.lg.Printf("screenshot: %v", err)
		return
	}
	h.lg.Printf("screenshot: %s", p)
	h.a.Notice("saved "+filepath.Base(p), 3*time.Second)
}

// picsFor lists every picture the rows reference, newest updates first,
// with the list thumbnail slot ahead of the others.
func picsFor(ds *data.Dataset) []images.Pic {
	var out []images.Pic
	for _, i := range ds.Order(data.SortUpdated) {
		r := &ds.Rows[i]
		if r.Img != "" && len(r.ImgSlots) > 0 {
			for _, want := range []string{"snap", "title", "ingame"} {
				for _, s := range r.ImgSlots {
					if s == want {
						out = append(out, images.Pic{Key: r.Img, Slot: s})
					}
				}
			}
		} else if r.Core != "" && !r.IsArcade() {
			out = append(out, images.Pic{Key: r.Core, Slot: "system"})
		}
	}
	return out
}

func openLog(path string) *log.Logger {
	if st, err := os.Stat(path); err == nil && st.Size() > logMax {
		os.Rename(path, strings.TrimSuffix(path, ".txt")+".1.txt")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return log.New(os.Stderr, "", log.Ltime|log.Lmicroseconds)
	}
	return log.New(f, "", log.Ltime|log.Lmicroseconds)
}

// frameStats keeps the last frames' paint, vsync wait and copy times.
type frameStats struct {
	n               int
	paint, wait, cp [256]time.Duration
}

func (f *frameStats) add(p, w, c time.Duration) {
	i := f.n % len(f.paint)
	f.paint[i], f.wait[i], f.cp[i] = p, w, c
	f.n++
}

func (f *frameStats) String() string {
	n := min(f.n, len(f.paint))
	if n == 0 {
		return "none"
	}
	stat := func(a *[256]time.Duration) string {
		var sum, mx time.Duration
		for i := 0; i < n; i++ {
			sum += a[i]
			if a[i] > mx {
				mx = a[i]
			}
		}
		return fmt.Sprintf("avg %s max %s", (sum / time.Duration(n)).Round(100*time.Microsecond), mx.Round(100*time.Microsecond))
	}
	return fmt.Sprintf("%d frames; paint %s; vsync wait %s; copy %s", f.n, stat(&f.paint), stat(&f.wait), stat(&f.cp))
}
