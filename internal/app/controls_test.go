package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func mappingApp() (*App, *time.Time, *[]support.Pad, *int) {
	a, clock := saverApp()
	a.cfg.AccessMonth = 202610
	a.cfg.ShowBetaFeatures = true
	pads := []support.Pad{{Node: "event3", Name: "Test pad", Vendor: 1, Product: 2, Direct: true, Mapped: true, OK: "B", Menu: "316", Slots: map[string]uint16{"A": 305, "B": 304, "X": 307, "Y": 308, "L": 310, "R": 311, "Start": 315, "Select": 314, "Up": 802, "Down": 803, "Left": 800, "Right": 801}}, {Node: "event4", Name: "Keyboard encoder", Vendor: 3, Product: 4, Keyboard: true}}
	a.controlInput.current, a.controlInput.node = padID(pads[0]), pads[0].Node
	saves := 0
	a.cfg.ControlProfiles = controls.New()
	a.cfg.Controls = &ControlsHooks{Devices: func() []support.Pad { return pads }, Save: func(f controls.File) error { saves++; return f.Validate() }}
	a.cfg.Support = &SupportHooks{Pads: func() []support.Pad { return pads[:1] }}
	return a, clock, &pads, &saves
}

func rawControl(p support.Pad, code uint16, key platform.Key) platform.Event {
	return platform.Event{InstanceID: p.InstanceID, DeviceID: padID(p), Node: p.Node, Source: p.Name, Direct: p.Direct, Keyboard: p.Keyboard, Code: code, Key: key}
}
func mappingTap(a *App, clock *time.Time, ev platform.Event) {
	*clock = clock.Add(30 * time.Millisecond)
	ev.At = *clock
	ev.Pressed = true
	a.Handle(ev)
	*clock = clock.Add(30 * time.Millisecond)
	ev.At = *clock
	ev.Pressed = false
	a.Handle(ev)
}
func TestPanelMappingSuppressesSearchAndLeavesOtherKeyboardsAlone(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[1]
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Panel: true, Style: "neogeo"}
	for _, tc := range []struct {
		code uint16
		key  platform.Key
	}{{2, platform.KeyStart}, {29, platform.KeyEnter}, {56, platform.KeyBack}, {57, platform.KeyTab}, {42, platform.KeySpace}, {6, platform.KeySelect}} {
		ev := rawControl(p, tc.code, platform.KeyOther)
		ev.Text = 'z'
		ev.Pressed = true
		got := a.mapControlEvent(ev)
		if got.Key != tc.key || got.Text != 0 {
			t.Fatal(got)
		}
		ev.DeviceID = "0005_0006"
		if got = a.mapControlEvent(ev); got.Text != 'z' || got.Key != platform.KeyOther {
			t.Fatal("typing keyboard changed", got)
		}
	}
	if a.controlActionLabel("tab", p, a.controlProfile(padID(p))) != "C" {
		t.Fatal("Neo Geo panel labels")
	}
	ev := rawControl(p, 30, platform.KeyOther)
	ev.Text = 'a'
	if a.mapControlEvent(ev).Text != 0 {
		t.Fatal("unassigned panel key typed")
	}
	a.cfg.ShowBetaFeatures = false
	ev.Text = 'a'
	if a.mapControlEvent(ev).Text != 0 {
		t.Fatal("Button mapping is free: the panel stays active with beta off")
	}
}

func TestControlsGatingAndCustomLabels(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[0]
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Style: "custom", Labels: map[string]string{"start": "Y", "enter": "R", "back": "B"}}
	a.controlInput.current = padID(p)
	if a.btn("Hold Start") != "Hold Y" || a.btn("A") != "R" {
		t.Fatal(a.btn("Hold Start"), a.btn("A"))
	}
	// Button mapping is free: Show beta features and codes change nothing
	a.cfg.ShowBetaFeatures = false
	a.cfg.AccessMonth = 0
	if a.btn("Hold Start") != "Hold Y" {
		t.Fatal("beta off dropped the labels")
	}
	a.OpenControls()
	if a.screen != ScreenControls {
		t.Fatal("beta off or no code kept Button mapping shut")
	}
	found := false
	for _, r := range a.optionsEntries() {
		if r.kind == "controls" {
			found = true
			if r.requiredMonth != 0 || strings.Contains(r.text, "*") {
				t.Fatal(r)
			}
		}
	}
	if !found {
		t.Fatal("missing early access row")
	}
}

func TestControlsKeepMiSTerMenuComboAndMatchRawRelease(t *testing.T) {
	a, clock, pads, _ := mappingApp()
	p := (*pads)[0]
	p.Menu = "314+315"
	(*pads)[0] = p
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Bindings: map[string]uint16{"start": 314, "select": 315}}
	ev := rawControl(p, 315, platform.KeyMenu)
	ev.Pressed = true
	if got := a.mapControlEvent(ev); got.Key != platform.KeyMenu {
		t.Fatal("Menu combo lost", got)
	}
	ev = rawControl(p, 802, platform.KeyUp)
	ev.Pressed = true
	ev.At = *clock
	a.Handle(ev)
	profile := a.cfg.ControlProfiles.Profiles[padID(p)]
	profile.Bindings["up"] = 803
	a.cfg.ControlProfiles.Profiles[padID(p)] = profile
	ev.Pressed = false
	a.Handle(ev)
	if a.down[platform.KeyUp] || a.rep.held {
		t.Fatal("mapping change stranded a held direction")
	}
}

func TestRemappedLegendWidthUsesPhysicalButtonName(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[0]
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Bindings: map[string]uint16{"start": 307, "tab": 315}}
	a.controlInput.current = padID(p)
	if got := a.hintWidth(hintChunks("X Filters"), "  "); got != a.sm.Width("Start Filters") {
		t.Fatalf("width=%d", got)
	}
}

func TestControlLegendCached(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[0]
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Style: "custom", Labels: map[string]string{"enter": "R"}}
	a.controlInput.current = padID(p)
	if a.btn("A") != "R" {
		t.Fatal(a.btn("A"))
	}
	// the hint line asks every frame: the labels come from the kept map
	a.controlInput.legendLabels["A"] = "kept"
	if a.btn("A") != "kept" {
		t.Fatal("labels rebuilt", a.btn("A"))
	}
	// an edit in place shows at once, a saved profile through controlsChanged
	a.cfg.ControlProfiles.Profiles[padID(p)].Labels["back"] = "Q"
	if a.btn("B") != "Q" {
		t.Fatal("in-place edit missed", a.btn("B"))
	}
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Style: "custom", Labels: map[string]string{"enter": "S", "back": "Q"}}
	a.controlsChanged()
	if a.btn("A") != "S" {
		t.Fatal("stale after a change", a.btn("A"))
	}
	// another pad gets its own labels
	a.controlInput.current = padID((*pads)[1])
	if a.btn("A") == "S" {
		t.Fatal("labels of the previous pad")
	}
}
