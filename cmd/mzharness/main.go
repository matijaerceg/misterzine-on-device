// mzharness runs the app on the headless platform and writes PNGs, so every
// view can be reviewed on the PC in both orientations and the device is only
// needed for what a PNG cannot show.
//
//	go run ./cmd/mzharness -rot left -out out/tate -script "shot list; enter; shot details"
//
// Script tokens, separated by ";": "KEY" or "KEY*N" taps a key, "hold KEY MS"
// holds it (repeat fires), "wait MS" advances the virtual clock, "shot NAME"
// saves NAME.png. Keys: up down left right enter back space tab pageup
// pagedown home end.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/images"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/headless"
	"github.com/matijaerceg/misterzine-on-device/internal/report"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

const allViews = "shot list; enter; shot details; wait 600; enter; shot screen; back; back; tab; shot filter; back; back; shot options; home; pagedown*2; right; down*4; enter; shot calibrate; back; end; shot options-bottom; back"

// harnessCode is the code -beta-locked asks for.
const harnessCode = "123456"

func main() {
	// Fixture timestamps must not depend on the machine running the harness.
	time.Local = time.UTC
	dataPath := flag.String("data", "testdata/data.json", "data.json")
	metaPath := flag.String("meta", "testdata/meta.json", "meta.json")
	rot := flag.String("rot", "none", "none, left or right (how the monitor is turned)")
	inset := flag.Int("inset", 15, "safe-zone inset in px (both axes)")
	out := flag.String("out", "out", "output directory")
	script := flag.String("script", "", "key script; empty = every view")
	nowStr := flag.String("now", "2026-09-08T12:00Z", "virtual clock start")
	status := flag.String("status", "fake", "fake, missing or unknown install statuses")
	romIssue := flag.String("rom-issue", "", "missing ROM archive warning fixture")
	seenAge := flag.Duration("seen", 48*time.Hour, "pretend the last look was this long ago (0 = first run)")
	logical := flag.Bool("logical", false, "save the unrotated logical canvas instead of the physical frame")
	imgDir := flag.String("images", "../misterzine/docs/images", "directory laid out like the site's docs/images; empty = placeholders")
	imgLoading := flag.Bool("images-loading", false, "pictures the -images directory lacks stay loading, as on a card still fetching them, instead of missing")
	updatePath := flag.String("update-state", "", "render an Update All state JSON without running an updater")
	updateRestart := flag.Bool("update-restart", false, "show the updated-program restart prompt")
	supportPath := flag.String("support-report", "", "controller diagnostic fixture; no devices are opened")
	reportSend := flag.String("report-send", "", "fake Troubleshooting -> Send a report: sent or failed; nothing is uploaded")
	appUpdate := flag.String("app-update", "", "available app version fixture")
	scanResult := flag.Bool("scan-result", false, "show completed card scan fixture")
	scanChanges := flag.Bool("scan-changes", false, "with -scan-result or -update-state: the card changed since the statuses on show (9 releases older and 3 not found are up to date now, 1 up to date is gone), and the scan finishes, which says what it changed")
	unchanged := flag.Bool("unchanged", false, "previous visit saw every release")
	filterRotation := flag.Bool("filter-rotation", false, "start with the strict current-rotation filter on")
	titleFont := flag.String("title-font", "tall", "list title font: tall, narrow or normal")
	listShot := flag.String("list-shot", "gameplay", "list thumbnail preference: gameplay or title")
	dateFormat := flag.String("date-format", "mm-dd", "list date column: mm-dd, dd-mm, mon-d, d-mon or yymmdd")
	layout := flag.String("layout", "list", "main view arrangement: list, split, picture or text")
	motion := flag.Bool("motion", false, "run the page and layout transitions on the scripted clock (off: every shot is an end state); the frames command records them")
	buttonLabels := flag.String("button-labels", "mister", "legend button names: mister, xbox, playstation or numbers")
	menuButton := flag.String("menu-button", "options", "what the pad's Menu button does: options or leave")
	launcher := flag.Bool("launcher", false, "start with the main menu shortcut on, so Open at boot, Return after game and Exit chord are live; the Options row toggles it")
	canvas := flag.String("canvas", "320x240", "canvas size WxH: 320x240, or a fit-display size such as 360x270 (1080p) or 400x300")
	rememberAlt := flag.Int("remember-alt", 0, "start with galagamw's alternative N (1 or 2) remembered as its version")
	recents := flag.Int("recents", 0, "pretend the first N rows were launched, the last one most recently; turns the Recents view on")
	viewsOff := flag.String("views-off", "", "views left out of the Y cycle, comma separated names (updated, debut, year, alphabetical, maker, core, favorites, recents); Recents is off unless -recents is given")
	installed := flag.String("installed", "", "Downloader database ids the card has, comma separated (e.g. distribution_mister,jtcores): Sources starts on installed only and the other sources are hidden")
	showNonArcade := flag.Bool("show-non-arcade", false, "include console, computer and other cores")
	arcadeIntro := flag.Bool("arcade-intro", false, "show upgrade explanation")
	splash := flag.Bool("splash", false, "show the startup logo fade")
	localPath := flag.String("local", "", "local rows fixture (a data.json-shaped array of rows the card scan would add)")
	betaBuild := flag.Bool("beta", false, "render the Patreon beta build (MisterZine Arcade) instead of the free one")
	betaLocked := flag.Bool("beta-locked", false, "render the beta build locked behind the test code "+harnessCode+" (implies -beta; the unlock is saved in a temporary folder)")
	switchFree := flag.Bool("switch-free", false, "a card that can take the beta back to the free version, which the lock screen offers; with -update-state the switch it starts reads that state (and -update-restart offers the restart) instead of the app opening on it")
	flag.Parse()
	beta.Set(*betaBuild || *betaLocked)
	unlockDir := ""
	if *betaLocked {
		sum := sha256.Sum256([]byte(harnessCode))
		beta.Batch, beta.CodeSHA256 = "harness", hex.EncodeToString(sum[:])
		dir, err := os.MkdirTemp("", "mzharness-beta-")
		if err != nil {
			die(err)
		}
		defer os.RemoveAll(dir)
		unlockDir = dir
	}

	rows, meta := load(*dataPath, *metaPath)
	if *localPath != "" {
		local, _ := load(*localPath, "")
		rows = data.MergeLocal(rows, local)
	}
	upd, _ := data.ParseMetaTime(meta.Updated)
	ds := data.Ingest(rows, meta.Hash, upd)

	now, err := time.Parse("2006-01-02T15:04Z", *nowStr)
	if err != nil {
		die(err)
	}
	clock := now
	var rotation gfx.Rotation
	switch *rot {
	case "left":
		rotation = gfx.RotLeft
	case "right":
		rotation = gfx.RotRight
	}
	var cw, ch int
	if _, err := fmt.Sscanf(*canvas, "%dx%d", &cw, &ch); err != nil || cw < 120 || ch < 120 {
		die(fmt.Errorf("-canvas %q: want WxH, at least 120x120", *canvas))
	}
	disp := headless.NewDisplay(cw, ch)
	cmd := &headless.Cmd{}
	var a *app.App
	var switchState *updater.State // -switch-free with -update-state
	cfg := app.Config{ShowNonArcade: *showNonArcade, ArcadeIntro: *arcadeIntro,
		PhysW: cw, PhysH: ch, Rotation: rotation, SafeInsetX: *inset, SafeInsetY: *inset,
		Now:            func() time.Time { return clock },
		ClockTrusted:   true,
		Favorites:      map[string]bool{},
		Launch:         func(p string) { fmt.Println("launch:", p); cmd.Send("load_core " + p) },
		ROMIssue:       func(string, bool) (string, bool) { return *romIssue, *romIssue != "" },
		Quit:           func() { fmt.Println("quit") },
		Version:        "harness",
		RememberSort:   true,
		FollowRotation: true,
		FilterRotation: *filterRotation,
		InstalledOnly:  *installed != "",
		ViewsOff:       viewsOffList(*viewsOff, *recents > 0),
		TitleFont:      *titleFont,
		ListShot:       *listShot,
		DateFormat:     *dateFormat,
		ListLayout:     *layout,
		ButtonLabels:   *buttonLabels,
		MenuButton:     *menuButton,
		Launcher:       func() bool { return *launcher },
		// a card that can fetch the app by itself, so the -app-update fixture
		// renders the Update MisterZine only row as a real card would
		CanUpdateApp:    func() bool { return true },
		CanSwitchToFree: func() bool { return *switchFree },
		Action: func(kind, arg string) {
			if kind == "launcher" {
				*launcher = arg == "on"
			}
			if kind == "update" && arg == updater.ModeFree && switchState != nil {
				// what the host's start of the run hands back
				a.SetUpdate(*switchState, true)
				a.SetUpdateRestart(switchState.ID, *updateRestart)
			}
		},
		Alternatives: func(r *data.Row) []string {
			if r.SN == "galagamw" {
				return []string{"_Arcade/_alternatives/_Galaga/Galaga (Namco).mra",
					"_Arcade/_alternatives/_Galaga/Galaga (Midway set 1, fast shoot hack, bootleg set 2).mra"}
			}
			if r.K == "local:orphanf" {
				return []string{"_Arcade/_Extra/Orphan Fighter (set 2).mra"}
			}
			return nil
		},
	}
	if beta.Check(unlockDir) != nil {
		// as the device does: the lock screen until the code is entered
		cfg.BetaUnlock = func(code string) error { return beta.Unlock(unlockDir, code) }
	}
	if *romIssue != "" {
		// the background check knows every file, and finds the same fault
		cfg.ROMKnown = func(string) (string, bool, bool) { return *romIssue, true, true }
		cfg.ROMProgress = func() (int, int) { return 3, 3 }
	}
	if *imgDir != "" {
		if _, err := os.Stat(*imgDir); err == nil {
			cfg.Images = images.NewLocal(*imgDir)
		}
	}
	if *imgLoading {
		cfg.Images = loadingImages{cfg.Images}
	}
	for i := min(*recents, len(rows)) - 1; i >= 0; i-- {
		// launches a day apart, the newest one yesterday
		cfg.RecentLaunches = append(cfg.RecentLaunches, data.Recent{K: rows[i].K, At: now.Add(-time.Duration(len(cfg.RecentLaunches)+1) * 24 * time.Hour).UTC().Format(time.RFC3339)})
	}
	if *rememberAlt > 0 {
		for i := range rows {
			if rows[i].SN == "galagamw" {
				alts := cfg.Alternatives(&rows[i])
				cfg.Versions = map[string]string{rows[i].K: alts[min(*rememberAlt, len(alts))-1]}
			}
		}
	}
	if *supportPath != "" {
		b, err := os.ReadFile(*supportPath)
		if err != nil {
			die(err)
		}
		var report support.Report
		if err := json.Unmarshal(b, &report); err != nil {
			die(err)
		}
		cfg.Support = &app.SupportHooks{
			Start:  func(time.Time, time.Time) {},
			Finish: func() support.Report { return report },
			Load:   func() support.Report { return report },
			Launch: func(game, target string) support.Report {
				r := report
				r.Launch = support.Launch{Game: game, Target: target, Result: "Cannot launch: target missing or invalid", Detail: "Example failure for visual review"}
				return r
			},
			// two pads for the pad tester: one defined in MiSTer by position
			// (A right, B bottom), one only readable for Start
			Pads: func() []support.Pad {
				return []support.Pad{
					{Node: "event0", Name: "USB Arcade Controller", Vendor: 0x1234, Product: 0x5678, Map: "Linux default", Slots: map[string]uint16{"Start": 315}},
					{Node: "event3", Name: "Microsoft X-Box 360 pad", Vendor: 0x045e, Product: 0x028e, Mapped: true, Direct: true, Menu: "316", OK: "B", Back: "A", Map: "/media/fat/config/inputs/input_045e_028e_v3.map",
						Slots: map[string]uint16{"Up": 802, "Down": 803, "Left": 800, "Right": 801, "A": 305, "B": 304, "X": 308, "Y": 307, "L": 310, "R": 773, "Select": 314, "Start": 315}},
				}
			},
		}
	}
	if *reportSend != "" {
		if cfg.Support == nil {
			cfg.Support = &app.SupportHooks{Load: func() support.Report { return support.Report{} }}
		}
		o := report.Outcome{Code: "K7M4", Saved: "misterzine/report.txt"}
		if *reportSend == "failed" {
			o = report.Outcome{Problem: "No connection to the report service.", Saved: "misterzine/report.txt"}
		}
		cfg.Support.SendReport = func(_ report.AppPart, done func(report.Outcome)) { done(o) }
	}
	if *status == "fake" {
		cfg.Status = func(i int) data.Status { return data.Status(1 + i%4) }
	} else if *status == "missing" {
		cfg.Status = func(int) data.Status { return data.StatusNotFound }
	}
	// -scan-changes: the statuses before the scan, where they differ from the
	// ones it finds; scanned turns to the found ones
	scanned := false
	if *scanChanges && cfg.Status != nil {
		found := cfg.Status
		before := map[int]data.Status{}
		older, missing, gone := 9, 3, 1
		for i := range ds.Rows {
			if r := &ds.Rows[i]; !r.IsArcade() || r.Deprecated {
				continue // outside the enabled catalogue the scan screen counts
			}
			switch st := found(i); {
			case st == data.StatusCurrent && older > 0:
				before[i], older = data.StatusOutdated, older-1
			case st == data.StatusCurrent && missing > 0:
				before[i], missing = data.StatusNotFound, missing-1
			case st == data.StatusNotFound && gone > 0:
				before[i], gone = data.StatusCurrent, gone-1
			}
		}
		cfg.Status = func(i int) data.Status {
			if st, ok := before[i]; ok && !scanned {
				return st
			}
			return found(i)
		}
	}
	// favorite a few rows so the star shows
	for i := 0; i < len(rows) && i < 40; i += 7 {
		cfg.Favorites[rows[i].K] = true
	}
	var stored *data.SeenRecord
	if *seenAge > 0 {
		// a previous visit that saw everything but the newest 5 rows in the updated sort
		prev := data.Ingest(rows, "", upd)
		order := prev.Order(data.SortUpdated)
		cur := map[string]string{}
		for n, i := range order {
			if n < 5 && !*unchanged {
				continue
			}
			cur[rows[i].K] = rows[i].Updated
		}
		stored = &data.SeenRecord{T: now.Add(-*seenAge).UTC().Format(time.RFC3339), Cur: cur}
	}
	a = app.New(cfg, ds, stored)
	if *motion {
		a.EnablePageTransitions()
	}
	if *splash {
		a.StartSplash()
	}
	if *installed != "" {
		var dbs []data.DB
		for _, id := range strings.Split(*installed, ",") {
			dbs = append(dbs, data.DB{ID: strings.TrimSpace(id)})
		}
		a.SetHiddenSources(data.HiddenSources(dbs), true)
		a.Refilter()
	}
	a.SetAppUpdate(*appUpdate)
	if *scanResult {
		a.OpenScan()
		scanned = true
		a.FinishScan("", true)
	}
	a.SetCatalogChecked(now)
	if *updatePath != "" {
		b, err := os.ReadFile(*updatePath)
		if err != nil {
			die(err)
		}
		var state updater.State
		if err := json.Unmarshal(b, &state); err != nil {
			die(err)
		}
		if *switchFree {
			switchState = &state // the lock screen's switch starts it
		} else {
			if *scanChanges && !state.Active() {
				// the run's last running snapshot first, so its end asks for
				// the rescan, as the host does
				run := state
				run.Status = "running"
				a.SetUpdate(run, true)
				a.SetUpdate(state, true)
				a.RescanAfterUpdate()
				scanned = true
				a.CardScanned(true)
			} else {
				a.SetUpdate(state, true)
			}
			a.SetUpdateRestart(state.ID, *updateRestart)
		}
	}

	if err := os.MkdirAll(*out, 0755); err != nil {
		die(err)
	}
	present := func() {
		frame, dirty := a.Paint()
		if dirty != nil {
			disp.Present(frame, dirty)
		}
	}
	present()
	shot := func(name string) {
		p := filepath.Join(*out, name+".png")
		var err error
		if *logical {
			err = savePNG(p, a.Logical())
		} else {
			err = disp.SavePNG(p)
		}
		if err != nil {
			die(err)
		}
		fmt.Println("wrote", p, "screen", a.Screen())
	}
	s := *script
	if s == "" {
		s = allViews
	}
	for _, tok := range strings.Split(s, ";") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		f := strings.Fields(tok)
		need := map[string]int{"shot": 2, "wait": 2, "hold": 3, "press": 2, "release": 2, "frames": 4}[f[0]]
		if len(f) < need {
			die(fmt.Errorf("script: %q needs %d words", tok, need))
		}
		switch f[0] {
		case "shot":
			present()
			shot(f[1])
		case "type":
			for _, ch := range strings.TrimPrefix(tok, "type ") {
				a.Handle(platform.Event{Text: ch, Pressed: true, At: clock, Source: "script"})
				clock = clock.Add(30 * time.Millisecond)
			}
			present()
		case "wait":
			ms, _ := strconv.Atoi(f[1])
			clock = advance(a, clock, time.Duration(ms)*time.Millisecond, present)
		case "frames": // frames NAME MS COUNT: COUNT shots NAME-00.. MS apart (with -motion, a transition frame by frame)
			ms, _ := strconv.Atoi(f[2])
			count, _ := strconv.Atoi(f[3])
			for i := 0; i < count; i++ {
				if i > 0 {
					clock = advance(a, clock, time.Duration(ms)*time.Millisecond, present)
				}
				present()
				shot(fmt.Sprintf("%s-%02d", f[1], i))
			}
		case "press", "release": // one edge, for chords such as Select held
			k := platform.ParseKey(f[1])
			if k == platform.KeyNone {
				die(fmt.Errorf("unknown key %q", f[1]))
			}
			a.Handle(platform.Event{Key: k, Pressed: f[0] == "press", At: clock})
			clock = clock.Add(30 * time.Millisecond)
			present()
		case "hold":
			k := platform.ParseKey(f[1])
			ms, _ := strconv.Atoi(f[2])
			a.Handle(platform.Event{Key: k, Pressed: true, At: clock})
			present()
			clock = advance(a, clock, time.Duration(ms)*time.Millisecond, present)
			a.Handle(platform.Event{Key: k, Pressed: false, At: clock})
		default:
			name, n := f[0], 1
			if i := strings.Index(name, "*"); i > 0 {
				n, _ = strconv.Atoi(name[i+1:])
				name = name[:i]
			}
			k := platform.ParseKey(name)
			if k == platform.KeyNone {
				die(fmt.Errorf("unknown key %q", name))
			}
			for j := 0; j < n; j++ {
				a.Handle(platform.Event{Key: k, Pressed: true, At: clock})
				clock = clock.Add(30 * time.Millisecond)
				a.Handle(platform.Event{Key: k, Pressed: false, At: clock})
				clock = clock.Add(30 * time.Millisecond)
				present()
			}
		}
	}
	if len(cmd.Lines) > 0 {
		fmt.Println("commands:", cmd.Lines)
	}
}

// advance moves the virtual clock in 10 ms steps, ticking the app.
func advance(a *app.App, clock time.Time, d time.Duration, present func()) time.Time {
	end := clock.Add(d)
	nextFrame := clock.Add(16667 * time.Microsecond)
	for clock.Before(end) {
		clock = clock.Add(10 * time.Millisecond)
		if clock.After(end) {
			clock = end
		}
		a.SaverSettle() // the saver's blur levels come from a worker; the scripted clock waits for them
		changed := a.Tick(clock)
		for !clock.Before(nextFrame) {
			if a.OptionSamplesRunning() {
				changed = a.OptionSampleFrame() || changed
			}
			if a.LayoutTransitionRunning() {
				// the layout motion steps once per displayed frame
				if a.LayoutMotionFrame() {
					changed = true
					present()
				}
			}
			nextFrame = nextFrame.Add(16667 * time.Microsecond)
		}
		if changed {
			present()
		}
	}
	return clock
}

// loadingImages reports every picture its directory lacks as still loading.
type loadingImages struct{ app.Images }

func (l loadingImages) Get(req app.ImageReq) (*image.RGBA, app.ImageState) {
	if l.Images != nil {
		if img, st := l.Images.Get(req); img != nil {
			return img, st
		}
	}
	return nil, app.ImageLoading
}

func (l loadingImages) Want(reqs []app.ImageReq) {
	if l.Images != nil {
		l.Images.Want(reqs)
	}
}

func (l loadingImages) SetPaused(p bool) {
	if l.Images != nil {
		l.Images.SetPaused(p)
	}
}

func load(dataPath, metaPath string) ([]data.Row, data.Meta) {
	f, err := os.Open(dataPath)
	if err != nil {
		die(err)
	}
	defer f.Close()
	rows, err := data.DecodeRows(f)
	if err != nil {
		die(err)
	}
	var meta data.Meta
	if mf, err := os.Open(metaPath); err == nil {
		meta, _ = data.DecodeMeta(mf)
		mf.Close()
	}
	return rows, meta
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// viewsOffList is the harness's Options -> Views state.
func viewsOffList(names string, recents bool) []string {
	var out []string
	for _, n := range strings.Split(names, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	if !recents {
		out = append(out, "recents")
	}
	return out
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "mzharness:", err)
	os.Exit(1)
}
