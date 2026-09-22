package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/report"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func supportApp() (*App, *time.Time) {
	a, clock := saverApp()
	a.SetData(data.Ingest([]data.Row{{K: "test", Title: "Installed game", Base: "Arcade", MRA: "_Arcade/Game.mra"}}, "fixture", *clock), nil)
	return a, clock
}

func TestSupportCaptureSuppressesActionsAndFreezes(t *testing.T) {
	a, clock := supportApp()
	starts, finishes, launches := 0, 0, 0
	a.cfg.Launch = func(string) { launches++ }
	a.cfg.Support = &SupportHooks{
		Start: func(from, until time.Time) {
			starts++
			if !from.Equal(clock.Add(3*time.Second)) || !until.Equal(clock.Add(9*time.Second)) {
				t.Fatal("wrong capture window")
			}
		},
		Finish: func() support.Report { finishes++; return support.Report{Tested: true, Version: "test"} },
	}
	a.OpenTroubleshooting()
	if starts != 0 {
		t.Fatal("opening menu started recording")
	}
	a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: *clock})
	a.Handle(platform.Event{Key: platform.KeyEnter, At: *clock})
	for _, key := range []platform.Key{platform.KeyStart, platform.KeyEnter, platform.KeyBack, platform.KeyTab, platform.KeySpace, platform.KeyDown} {
		a.Handle(platform.Event{Key: key, Pressed: true, Text: 'a', At: clock.Add(4 * time.Second)})
		a.Handle(platform.Event{Key: key, At: clock.Add(4 * time.Second)})
	}
	if a.Screen() != ScreenTroubleshooting || a.Search() != "" || launches != 0 || a.Repeating() || !a.supportCapturing() {
		t.Fatal("test input escaped capture")
	}
	*clock = clock.Add(9 * time.Second)
	a.Tick(*clock)
	if finishes != 1 || a.support.mode != "result" || !a.NextTick().IsZero() {
		t.Fatal("test did not freeze automatically")
	}
	a.Tick(clock.Add(5 * time.Minute))
	if finishes != 1 || a.ScreensaverActive() {
		t.Fatal("frozen photo result changed/dimmed")
	}
	a.actSupport(platform.KeyBack)
	a.actSupport(platform.KeyBack)
	if a.Screen() != ScreenOptions {
		t.Fatal("cannot leave troubleshooting")
	}
}

func TestSupportLaunchRequiresFreshConfirmAndCapturesTarget(t *testing.T) {
	a, clock := supportApp()
	launches := 0
	a.cfg.Support = &SupportHooks{Launch: func(game, target string) support.Report {
		launches++
		if game != "Installed game" || target != "_Arcade/Game.mra" {
			t.Fatalf("wrong target: %q %q", game, target)
		}
		return support.Report{Launch: support.Launch{Result: "Test failure"}}
	}}
	a.OpenTroubleshooting()
	a.actSupport(platform.KeyDown)
	a.actSupport(platform.KeyDown) // past the pad tester
	press := platform.Event{Key: platform.KeyEnter, Pressed: true, At: *clock}
	a.Handle(press)
	a.Handle(press) // held/duplicate confirm must not launch
	a.Handle(platform.Event{Key: platform.KeyStart, Pressed: true, At: *clock})
	if launches != 0 || a.support.mode != "launch" {
		t.Fatal("launch happened before explicit confirmation")
	}
	a.Handle(platform.Event{Key: platform.KeyEnter, At: *clock})
	a.Handle(press)
	if launches != 1 || a.support.mode != "result" || a.support.report.Launch.Result != "Test failure" {
		t.Fatal("confirm did not use support launch path")
	}
}

func TestSupportReportPaginationPreservesEvidence(t *testing.T) {
	for _, rot := range []gfx.Rotation{gfx.RotNone, gfx.RotLeft, gfx.RotRight} {
		a, _ := supportApp()
		a.SetRotation(rot)
		a.OpenTroubleshooting()
		name := "A very long controller name with distinctive ending END-OF-NAME"
		a.support.mode = "result"
		a.support.report = support.Report{Version: "vtest", Tested: true,
			Devices: []support.Device{{Node: "event2", Name: name, Vendor: 0x1234, Product: 0x5678}},
			Signals: []support.Signal{{Node: "event2", Type: 1, Code: 299, Down: 2, Up: 2}}}
		pages := a.supportPages()
		var all []string
		for i, page := range pages {
			all = append(all, page...)
			a.support.page = i
			a.Invalidate()
			a.Paint()
			for _, line := range page {
				if len(line) > a.sm.Cols(a.lay.Body.Inset(4).Dx()) {
					t.Fatalf("overflow: %q", line)
				}
			}
		}
		text := strings.Join(all, " ")
		for _, want := range []string{"Button 299", "1234:5678", "END-OF-NAME", "Skipped by normal input reader"} {
			if !strings.Contains(text, want) {
				t.Fatalf("lost %q in %s", want, text)
			}
		}
	}
}

// Send a report: the menu's fifth entry asks first, waits for the host
// while it sends (no key leaves), then shows the code or where the card
// copy is. The UI's part names the rule hiding each local game.
func TestSendReportFlow(t *testing.T) {
	a, clock := supportApp()
	cat := []data.Row{{K: "gigandes", Title: "Gigandes", Base: "Arcade", Src: "jtbindb", Core: "jttaitox", SN: "gigandes", Rot: "Horizontal", Updated: "2026-09-07"}}
	local := []data.Row{
		{K: "local:gigandes", Title: "Gigandes (World bazset)", Base: "Arcade", Src: data.SrcLocal, SN: "gigandes", Core: "Gigandes_baz", MRA: "_Arcade/Gigandes (World bazset).mra", Standin: true, Rot: "Horizontal"},
		{K: "local:volfied", Title: "Volfied (World, rev 1)", Base: "Arcade", Src: data.SrcLocal, SN: "volfied", Core: "Volfied", MRA: "_Arcade/Volfied (World, rev 1) .mra", Rot: "Vertical (CW)"},
	}
	a.cfg.FilterRotation = true
	a.SetData(data.Ingest(data.MergeLocal(cat, local), "fixture", *clock), nil)
	var part report.AppPart
	var done func(report.Outcome)
	a.cfg.Support = &SupportHooks{SendReport: func(p report.AppPart, d func(report.Outcome)) { part, done = p, d }}
	press := func(k platform.Key) {
		a.Handle(platform.Event{Key: k, Pressed: true, At: *clock})
		a.Handle(platform.Event{Key: k, At: *clock})
		a.Paint()
	}

	a.OpenTroubleshooting()
	for i := 0; i < 6; i++ {
		press(platform.KeyDown) // the cursor stops on the last entry
	}
	press(platform.KeyEnter)
	if a.support.mode != "report" || done != nil {
		t.Fatalf("mode %q: the report must ask before sending", a.support.mode)
	}
	press(platform.KeyEnter)
	if a.support.mode != "report-sending" || done == nil {
		t.Fatalf("mode %q after A", a.support.mode)
	}
	for _, k := range []platform.Key{platform.KeyBack, platform.KeyEnter, platform.KeyDown} {
		press(k)
	}
	if a.support.mode != "report-sending" || a.Screen() != ScreenTroubleshooting {
		t.Fatal("a key left the screen while the report was on its way")
	}

	got := map[string]report.LocalGame{}
	for _, g := range part.Local {
		got[g.K] = g
	}
	if g := got["local:gigandes"]; g.StandsFor != "gigandes" || g.Hidden != "" || g.Core != "Gigandes_baz" {
		t.Fatalf("stand-in: %+v", g)
	}
	if g := got["local:volfied"]; !strings.Contains(g.Hidden, "Filter by rotation") {
		t.Fatalf("hidden local game: %+v", g)
	}
	if part.Catalogue != 1 || part.Rows != 3 || !strings.Contains(part.Effective, `filter by rotation "h"`) {
		t.Fatalf("list part: %+v", part)
	}

	done(report.Outcome{Code: "K7Q2", Saved: "misterzine/report.txt"})
	a.Paint()
	if a.support.mode != "report-done" || a.support.outcome.Code != "K7Q2" {
		t.Fatalf("mode %q outcome %+v", a.support.mode, a.support.outcome)
	}
	press(platform.KeyBack)
	if a.support.mode != "menu" {
		t.Fatalf("B from the result: %q", a.support.mode)
	}

	// Not sent: the screen says why and where the copy is.
	press(platform.KeyEnter)
	press(platform.KeyEnter)
	done(report.Outcome{Problem: "No connection to the report service.", Saved: "misterzine/report.txt"})
	a.Paint()
	if a.support.mode != "report-done" || a.support.outcome.Code != "" {
		t.Fatalf("failed send: %q %+v", a.support.mode, a.support.outcome)
	}
	press(platform.KeyBack)
	press(platform.KeyBack)
	if a.Screen() == ScreenTroubleshooting {
		t.Fatal("B from the menu stays on Troubleshooting")
	}
}
