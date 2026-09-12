package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
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
