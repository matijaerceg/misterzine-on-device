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
func TestControllerLabelsAtStartup(t *testing.T) {
	setup, clock, pads, _ := mappingApp()
	cfg := setup.cfg
	ds := setup.Data()
	first := (*pads)[0]
	first.InstanceID = "0001_0002_first"
	second := first
	second.InstanceID, second.Node = "0001_0002_second", "event7"
	*pads = []support.Pad{first, second}
	cfg.ControlProfiles.Profiles[first.InstanceID] = controls.Profile{
		MappingVersion: 2, BasicsComplete: true,
		Inputs: map[uint16]string{802: "up", 803: "down", 800: "left", 801: "right", 304: "enter", 305: "back", 307: "tab", 308: "space"},
		Names:  map[uint16]string{304: "A", 305: "B", 307: "C", 308: "D"},
	}
	open := func(id string) *App {
		cfg.LastControlID = id
		return New(cfg, ds, nil)
	}
	// An upgrade has no last-used ID yet. The only configured controller
	// already has the right legends, even with another pad plugged in.
	a := open("")
	if a.btn("A B X Y") != "A B C D" || a.screen != ScreenList {
		t.Fatal("sole mapped stick did not supply startup labels")
	}
	cfg.ControlProfiles.Profiles[second.InstanceID] = controls.Profile{Style: "numbers"}
	a = open("")
	if a.btn("X Y") != "X Y" {
		t.Fatal("chose arbitrarily between saved profiles")
	}
	writes := 0
	cfg.Controls.LastUsedChanged = func() { writes++ }
	for _, pad := range []support.Pad{second, first, first} {
		mappingTap(a, clock, rawControl(pad, 802, platform.KeyUp))
	}
	mappingTap(a, clock, platform.Event{Key: platform.KeyUp, Source: "MiSTer virtual input"})
	if a.LastControlID() != first.InstanceID || writes != 2 {
		t.Fatalf("last control %q, state changes %d", a.LastControlID(), writes)
	}
	// A dock can front different sticks under one USB identity. Only
	// the exposed identity selects a profile, even if the name changes.
	(*pads)[0].Name = "Dock with a different stick"
	(*pads)[0].Node = "event9"
	a = open(a.LastControlID())
	if a.btn("A B X Y") != "A B C D" || a.controlInput.node != "event9" {
		t.Fatal("restart did not restore the stick on its new event node")
	}
	a = open(second.InstanceID)
	if a.btn("X Y") != "3 4" {
		t.Fatal("identical second controller did not retain its own labels")
	}
	*pads = (*pads)[1:]
	a = open(first.InstanceID)
	if a.btn("X Y") != "X Y" || a.LastControlID() != first.InstanceID {
		t.Fatal("disconnected stick showed labels or lost its identity")
	}
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
