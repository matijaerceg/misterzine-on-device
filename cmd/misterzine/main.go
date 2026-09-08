//go:build linux

// misterzine is the device binary: it runs the app on the MiSTer's
// framebuffer from the Scripts menu. See deploy/Scripts/misterzine.sh.
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
)

const (
	canvasW, canvasH = 320, 240
	logMax           = 1 << 20
)

type host struct {
	root, card string
	lg         *log.Logger
	console    *mister.Console
	cmd        *mister.Cmd
	fb         *mister.FB
	input      *mister.Input
	a          *app.App
	settings   store.Settings
	state      store.State
	favs       store.Favorites
	events     chan platform.Event
	uiRun      chan func()
	quit       chan struct{}
	launch     string
	dirty      bool // state needs saving
	favDirty   bool
	setDirty   bool
	clock      platform.Clock
	client     *fetch.Client
	img        *images.Service
	index      *scan.Index
	status     []data.Status
	alts       []scan.Alt
	scanCh     chan scanResult
	dataCh     chan fetch.Fresh
	netCh      chan string
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "console-restore" {
		mister.RestoreAll()
		return
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
	h := &host{root: root, card: card, lg: lg, events: make(chan platform.Event, 256), uiRun: make(chan func(), 8), quit: make(chan struct{}), dataCh: make(chan fetch.Fresh, 1), netCh: make(chan string, 4), scanCh: make(chan scanResult, 1)}

	// settings, state, favorites
	h.settings = store.DefaultSettings()
	if err := store.Load(filepath.Join(root, "settings.json"), &h.settings); err != nil && !errors.Is(err, os.ErrNotExist) {
		lg.Printf("settings: %v", err)
	}
	hasState := true
	if err := store.Load(filepath.Join(root, "state.json"), &h.state); err != nil {
		hasState = false
		if !errors.Is(err, os.ErrNotExist) {
			lg.Printf("state: %v", err)
		}
	}
	if err := store.Load(filepath.Join(root, "favorites.json"), &h.favs); err != nil && !errors.Is(err, os.ErrNotExist) {
		lg.Printf("favorites: %v", err)
	}

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
	h.clock = platform.Clock{Now: time.Now, Trusted: trusted}
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
		PhysW: canvasW, PhysH: canvasH, Rotation: rotation, SafeInset: h.settings.Inset,
		Now: time.Now, ClockTrusted: trusted, Favorites: favSet, Images: h.img,
		Progress: func() (int, int) { return h.img.Progress() },
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
		Launch:          func(target string) { h.launch = target; h.stop() },
		Quit:            h.stop,
		Version:         buildinfo.String(),
		FavChanged:      func() { h.favDirty = true },
		SettingsChanged: func() { h.setDirty = true },
		Action: func(kind, arg string) {
			lg.Printf("action: %s %s", kind, arg)
			if kind == "refresh" {
				go h.check(ds.Hash)
			}
			if kind == "rescan" {
				go h.scan(h.a.Data().Rows)
			}
			if kind == "prefetch" {
				h.settings.Prefetch = arg == "on"
				h.setDirty = true
				h.img.SetPrefetch(picsFor(h.a.Data()), h.settings.Prefetch)
			}
			if kind == "clearimg" {
				go h.img.ClearCache()
			}
		},
	}
	var seen *data.SeenRecord
	if hasState && (len(h.state.Seen.Cur) > 0 || h.state.Seen.T != "") {
		seen = &h.state.Seen
	}
	h.a = app.New(cfg, ds, seen)
	h.a.SetPrefetch(h.settings.Prefetch)
	if hasState {
		if h.state.Sort == "debut" {
			h.a.SetSort(data.SortDebut)
		}
		h.a.SetFilters(h.state.Filters)
		if h.state.CursorK != "" {
			h.a.MoveToKey(h.state.CursorK)
		}
	}
	h.a.SetNet(h.netLabel(ds))
	if ini.Found && !ini.AnalogVisible() {
		h.a.Notice("CRT only? add direct_video=1 under [Menu], see README", 20*time.Second)
	}
	h.present()
	lg.Printf("first frame at %v", time.Since(t0).Round(time.Millisecond))

	if debugAddr != "" {
		debugsrv.Serve(debugAddr, debugsrv.Hooks{
			Run:    h.runOnUI,
			Inject: func(ev platform.Event) { h.events <- ev },
			Shot:   func() *image.RGBA { return h.a.Logical() },
			State: func() any {
				return map[string]any{
					"version": buildinfo.String(), "screen": h.a.Screen().String(), "cursor": h.a.CursorKey(),
					"sort": h.a.Sort().String(), "rows": len(ds.Rows), "fb": h.fb.Geometry().String(),
					"rotation": h.a.Rotation().String(), "inset": h.a.Inset(), "devices": h.input.Devices(),
					"sysfs": mister.SysfsMode(), "uptime": time.Since(t0).String(),
				}
			},
			Quit: h.stop,
			Goto: func(k string) { h.a.MoveToKey(k); h.present() },
			Log:  filepath.Join(root, "log.txt"),
		}, lg)
	}

	// card scan: cores, MRA presence, then alternatives
	go h.scan(ds.Rows)

	// freshness: first check shortly after boot, then every 30 minutes
	go func() {
		time.Sleep(300 * time.Millisecond)
		h.check(ds.Hash)
		t := time.NewTicker(30 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-h.quit:
				return
			case <-t.C:
				h.check(h.a.Data().Hash)
			}
		}
	}()

	// the loop
	saveAt := time.Time{}
	lastState := h.snapshotState()
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
			if ev.Key == platform.KeyScreenshot {
				if ev.Pressed {
					h.screenshot()
				}
				break
			}
			h.a.Handle(ev)
			// drain whatever else arrived
		drain:
			for {
				select {
				case ev := <-h.events:
					h.a.Handle(ev)
				default:
					break drain
				}
			}
		case f := <-h.uiRun:
			f()
		case fr := <-h.dataCh:
			h.swap(fr)
		case s := <-h.netCh:
			h.a.SetNet(s)
			h.img.SetOffline(s == "offline")
		case <-h.img.Ready():
			h.a.Invalidate()
		case r := <-h.scanCh:
			first := h.index == nil
			h.index, h.status, h.alts = r.index, r.status, r.alts
			h.a.Refilter()
			if first || r.notice != "" {
				h.a.Notice(r.notice, 8*time.Second)
			}
		case <-tick:
		}
		h.a.Tick(time.Now())
		h.present()
		if st := h.snapshotState(); st != lastState {
			lastState = st
			h.dirty = true
			saveAt = time.Now().Add(500 * time.Millisecond)
		}
		if (h.dirty || h.favDirty || h.setDirty) && !saveAt.IsZero() && time.Now().After(saveAt) {
			h.saveAll(false)
			saveAt = time.Time{}
		}
		if h.launch != "" {
			h.saveAll(true)
			h.doLaunch()
			return 0
		}
	}
}

func (h *host) stop() {
	select {
	case <-h.quit:
	default:
		close(h.quit)
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
	h.uiRun <- func() { f(); close(done) }
	<-done
}

func (h *host) present() {
	frame, dirty := h.a.Paint()
	if dirty != nil {
		h.fb.Present(frame, dirty)
	}
}

func (h *host) snapshotState() string {
	return h.a.CursorKey() + "|" + h.a.Sort().String() + "|" + fmt.Sprint(h.a.Filters())
}

func (h *host) saveAll(final bool) {
	if h.dirty || final {
		f := h.a.Filters()
		st := store.State{Schema: 1, CursorK: h.a.CursorKey(), Sort: strings.ToLower(h.a.Sort().String()), Filters: f,
			LastOpen: time.Now().UTC().Format(time.RFC3339), DataHash: h.a.Data().Hash}
		if s := h.a.Seen(); s != nil {
			st.Seen = s.State
		}
		if err := store.Save(filepath.Join(h.root, "state.json"), st); err != nil {
			h.lg.Printf("state: %v", err)
		}
		h.dirty = false
	}
	if h.favDirty || final {
		h.favs.Apply(h.a.FavoriteSet(), time.Now())
		if err := store.Save(filepath.Join(h.root, "favorites.json"), h.favs); err != nil {
			h.lg.Printf("favorites: %v", err)
		}
		h.favDirty = false
	}
	if h.setDirty || final {
		h.settings.Inset = h.a.Inset()
		switch h.a.Rotation() {
		case gfx.RotLeft:
			h.settings.Rotation = "left"
		case gfx.RotRight:
			h.settings.Rotation = "right"
		default:
			h.settings.Rotation = "off"
		}
		if !h.setDirty && h.settings.Rotation != "" {
			// untouched: keep "auto" if that is what the file said
			var on store.Settings
			if store.Load(filepath.Join(h.root, "settings.json"), &on) == nil && on.Rotation == "auto" {
				h.settings.Rotation = "auto"
			}
		}
		if err := store.Save(filepath.Join(h.root, "settings.json"), h.settings); err != nil {
			h.lg.Printf("settings: %v", err)
		}
		h.setDirty = false
	}
}

// cleanup restores the machine; safe to call twice.
func (h *host) cleanup() {
	if h.img != nil {
		h.img.Close()
		h.img = nil
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
}

func (h *host) doLaunch() {
	abs, err := mister.LaunchPath(h.card, h.launch)
	if err != nil {
		h.lg.Printf("launch %q: %v", h.launch, err)
		return
	}
	h.cleanup()
	if err := h.cmd.Send("load_core " + abs); err != nil {
		h.lg.Printf("launch: %v", err)
	}
	h.lg.Printf("launched %s", abs)
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

func (h *host) netLabel(ds *data.Dataset) string {
	if ds.Updated.IsZero() || !h.clock.Trusted {
		return ""
	}
	return "data " + data.RelUpdated(time.Now(), ds.Updated)
}

// check runs a freshness check off the UI goroutine.
func (h *host) check(current string) {
	h.netCh <- "checking.."
	fr, err := h.client.Check(context.Background(), current)
	if err != nil {
		h.lg.Printf("check: %v", err)
		if errors.Is(err, fetch.ErrOffline) {
			h.netCh <- "offline"
		} else {
			h.netCh <- "check failed"
		}
		return
	}
	if !fr.Changed {
		h.lg.Printf("check: current (%.8s)", fr.Meta.Hash)
		h.netCh <- ""
		return
	}
	cache := filepath.Join(h.root, "cache")
	if err := store.WriteAtomic(filepath.Join(cache, "data.json"), fr.RawData); err != nil {
		h.lg.Printf("cache: %v", err)
	}
	if err := store.WriteAtomic(filepath.Join(cache, "meta.json"), fr.RawMeta); err != nil {
		h.lg.Printf("cache: %v", err)
	}
	h.lg.Printf("check: new data %.8s, %d rows", fr.Meta.Hash, len(fr.Rows))
	h.dataCh <- fr
}

// swap installs fetched data on the UI goroutine.
func (h *host) swap(fr fetch.Fresh) {
	old := h.a.Data()
	upd, _ := data.ParseMetaTime(fr.Meta.Updated)
	ds := data.Ingest(fr.Rows, fr.Meta.Hash, upd)
	news := data.DiffNews(old, fr.Rows)
	if h.index != nil {
		h.status = scan.Statuses(h.card, h.index, fr.Rows)
	}
	h.a.SetData(ds, nil)
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
	alts   []scan.Alt
	notice string
}

// scan reads the card off the UI goroutine: cores and MRA stats first (fast),
// alternatives after (slow the first time).
func (h *host) scan(rows []data.Row) {
	t0 := time.Now()
	idx := scan.ScanCores(h.card)
	st := scan.Statuses(h.card, idx, rows)
	counts := map[data.Status]int{}
	for _, s := range st {
		counts[s]++
	}
	h.lg.Printf("scan: %d cores; current %d, outdated %d, undated %d, not found %d (%v)", len(idx.Cores),
		counts[data.StatusCurrent], counts[data.StatusOutdated], counts[data.StatusFoundUndated], counts[data.StatusNotFound], time.Since(t0).Round(time.Millisecond))
	h.scanCh <- scanResult{index: idx, status: st, alts: h.alts, notice: fmt.Sprintf("card: %d current, %d older, %d not found", counts[data.StatusCurrent], counts[data.StatusOutdated], counts[data.StatusNotFound])}
	t1 := time.Now()
	alts := scan.ScanAlternatives(h.card, filepath.Join(h.root, "cache", "alts.json"))
	h.lg.Printf("scan: %d alternatives (%v)", len(alts), time.Since(t1).Round(time.Millisecond))
	h.scanCh <- scanResult{index: idx, status: st, alts: alts}
}

// screenshot saves the logical canvas (F12 on a keyboard).
func (h *host) screenshot() {
	dir := filepath.Join(h.root, "debug")
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
