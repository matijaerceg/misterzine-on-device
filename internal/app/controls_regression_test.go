package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
)

func TestControlRegressionLegendUsesActiveNode(t *testing.T) {
	a, _, pads, _ := mappingApp()
	first := (*pads)[0]
	second := first
	second.Node = "event8"
	first.Slots = map[string]uint16{"A": 305, "B": 304, "Start": 315}
	(*pads)[0] = first
	*pads = append(*pads, second)
	a.cfg.ControlProfiles.Profiles[padID(first)] = controls.Profile{Style: "mister", Bindings: map[string]uint16{"start": 307, "tab": 315}}
	ev := rawControl(second, 307, platform.KeyTab)
	ev.Pressed = true
	a.controlEvent(ev)
	if got := a.btn("Start"); got != "X" {
		t.Errorf("launch label uses different interface: got %q, want X", got)
	}
}

func TestControlRegressionUnassignedActionLegend(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[0]
	p.Menu = "314+315"
	(*pads)[0] = p
	profile := controls.Profile{Style: "mister"}.Clone()
	profile.Bindings["menu"] = 307
	profile.Bindings["tab"] = 0
	if err := controls.ValidateInputs(a.normalizedControl(p, profile).Inputs); err != nil {
		t.Fatalf("fixture cannot save: %v", err)
	}
	a.cfg.ControlProfiles.Profiles[padID(p)] = profile
	ev := rawControl(p, 307, platform.KeyTab)
	ev.Pressed = true
	mapped, _, _ := a.controlEvent(ev)
	if mapped.Key != platform.KeyMenu {
		t.Fatalf("fixture should now open Menu: %v", mapped.Key)
	}
	if got := a.btn("X"); got == "X" {
		t.Errorf("unassigned Filters is still labelled with X (which now opens Menu)")
	}
	profile.Style = "custom"
	profile.Labels["tab"] = "C"
	if got := a.controlActionLabel("tab", p, profile); got != "-" {
		t.Errorf("unassigned custom label: %q", got)
	}
}

func TestControlPanelRepeatReleaseAndTyping(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[1]
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{Panel: true}
	ev := rawControl(p, 30, platform.KeyOther)
	ev.Text = 'a'
	ev.Pressed = true
	a.controlEvent(ev)
	if _, handled, _ := a.controlEvent(ev); !handled {
		t.Fatal("panel repeat escaped")
	}
	ev.Pressed = false
	a.controlEvent(ev)
	ev.Pressed = true
	if _, handled, _ := a.controlEvent(ev); handled {
		t.Fatal("new press swallowed")
	}
	a.cfg.ControlProfiles.Profiles[padID(p)] = controls.Profile{}
	if got, handled, _ := a.controlEvent(ev); handled || got.Text != 'a' {
		t.Fatal("typing repeat suppressed")
	}
}
