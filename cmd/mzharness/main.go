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
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/images"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/platform/headless"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

const allViews = "shot list; enter; shot details; wait 600; enter; shot screen; back; back; tab; shot filter; back; back; shot options; down*18; enter; shot calibrate; back; end; shot options-bottom; back"

func main() {
	dataPath := flag.String("data", "testdata/data.json", "data.json")
	metaPath := flag.String("meta", "testdata/meta.json", "meta.json")
	rot := flag.String("rot", "none", "none, left or right (how the monitor is turned)")
	inset := flag.Int("inset", 15, "safe-zone inset in px (both axes)")
	out := flag.String("out", "out", "output directory")
	script := flag.String("script", "", "key script; empty = every view")
	nowStr := flag.String("now", "2026-09-08T12:00Z", "virtual clock start")
	status := flag.String("status", "fake", "fake, missing or unknown install statuses")
	seenAge := flag.Duration("seen", 48*time.Hour, "pretend the last look was this long ago (0 = first run)")
	logical := flag.Bool("logical", false, "save the unrotated logical canvas instead of the physical frame")
	imgDir := flag.String("images", "../misterzine/docs/images", "directory laid out like the site's docs/images; empty = placeholders")
	updatePath := flag.String("update-state", "", "render an Update All state JSON without running an updater")
	supportPath := flag.String("support-report", "", "controller diagnostic fixture; no devices are opened")
	appUpdate := flag.String("app-update", "", "available app version fixture")
	scanResult := flag.Bool("scan-result", false, "show completed card scan fixture")
	unchanged := flag.Bool("unchanged", false, "previous visit saw every release")
	filterRotation := flag.Bool("filter-rotation", false, "start with the strict current-rotation filter on")
	titleFont := flag.String("title-font", "tall", "list title font: tall, narrow or normal")
	listShot := flag.String("list-shot", "gameplay", "list thumbnail preference: gameplay or title")
	dateFormat := flag.String("date-format", "mm-dd", "list date column: mm-dd, dd-mm, mon-d, d-mon or yymmdd")
	layout := flag.String("layout", "list", "main view arrangement: list, split or picture")
	buttonLabels := flag.String("button-labels", "mister", "legend button names: mister, xbox, playstation or numbers")
	menuButton := flag.String("menu-button", "options", "what the pad's Menu button does: options or leave")
	canvas := flag.String("canvas", "320x240", "canvas size WxH: 320x240, or a fit-display size such as 360x270 (1080p) or 400x300")
	rememberAlt := flag.Int("remember-alt", 0, "start with galagamw's alternative N (1 or 2) remembered as its version")
	recents := flag.Int("recents", 0, "pretend the first N rows were launched, the last one most recently; turns the Recents view on")
	viewsOff := flag.String("views-off", "", "views left out of the Y cycle, comma separated names (updated, debut, year, alphabetical, maker, favorites, recents); Recents is off unless -recents is given")
	installed := flag.String("installed", "", "Downloader database ids the card has, comma separated (e.g. distribution_mister,jtcores): Sources starts on installed only and the other sources are hidden")
	flag.Parse()

	rows, meta := load(*dataPath, *metaPath)
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
	cfg := app.Config{
		PhysW: cw, PhysH: ch, Rotation: rotation, SafeInsetX: *inset, SafeInsetY: *inset,
		Now:            func() time.Time { return clock },
		ClockTrusted:   true,
		Favorites:      map[string]bool{},
		Launch:         func(p string) { fmt.Println("launch:", p); cmd.Send("load_core " + p) },
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
		Alternatives: func(r *data.Row) []string {
			if r.SN == "galagamw" {
				return []string{"_Arcade/_alternatives/_Galaga/Galaga (Namco).mra",
					"_Arcade/_alternatives/_Galaga/Galaga (Midway set 1, fast shoot hack, bootleg set 2).mra"}
			}
			return nil
		},
	}
	if *imgDir != "" {
		if _, err := os.Stat(*imgDir); err == nil {
			cfg.Images = images.NewLocal(*imgDir)
		}
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
					{Node: "event3", Name: "Microsoft X-Box 360 pad", Vendor: 0x045e, Product: 0x028e, Mapped: true, Direct: true, Menu: "316", Map: "/media/fat/config/inputs/input_045e_028e_v3.map",
						Slots: map[string]uint16{"Up": 802, "Down": 803, "Left": 800, "Right": 801, "A": 305, "B": 304, "X": 308, "Y": 307, "L": 310, "R": 773, "Select": 314, "Start": 315}},
				}
			},
		}
	}
	if *status == "fake" {
		cfg.Status = func(i int) data.Status { return data.Status(1 + i%4) }
	} else if *status == "missing" {
		cfg.Status = func(int) data.Status { return data.StatusNotFound }
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
	a := app.New(cfg, ds, stored)
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
		a.FinishScan("", true)
	}
	a.SetNet("data " + data.RelUpdated(now, upd))
	if *updatePath != "" {
		b, err := os.ReadFile(*updatePath)
		if err != nil {
			die(err)
		}
		var state updater.State
		if err := json.Unmarshal(b, &state); err != nil {
			die(err)
		}
		a.SetUpdate(state, true)
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
		need := map[string]int{"shot": 2, "wait": 2, "hold": 3, "press": 2, "release": 2}[f[0]]
		if len(f) < need {
			die(fmt.Errorf("script: %q needs %d words", tok, need))
		}
		switch f[0] {
		case "shot":
			present()
			p := filepath.Join(*out, f[1]+".png")
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
		case "type":
			for _, ch := range strings.TrimPrefix(tok, "type ") {
				a.Handle(platform.Event{Text: ch, Pressed: true, At: clock, Source: "script"})
				clock = clock.Add(30 * time.Millisecond)
			}
			present()
		case "wait":
			ms, _ := strconv.Atoi(f[1])
			clock = advance(a, clock, time.Duration(ms)*time.Millisecond, present)
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
	for clock.Before(end) {
		clock = clock.Add(10 * time.Millisecond)
		if clock.After(end) {
			clock = end
		}
		a.SaverSettle() // the saver's blur levels come from a worker; the scripted clock waits for them
		if a.Tick(clock) {
			present()
		}
	}
	return clock
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
