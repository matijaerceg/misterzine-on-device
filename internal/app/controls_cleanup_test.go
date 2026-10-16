package app

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

func chooseControlAction(a *App, id string) {
	a.actControlPopup(platform.KeyEnter)
	for i, action := range checkOrder {
		if action == id {
			a.mapping.cursor = i
		}
	}
	a.actControlPopup(platform.KeyEnter)
}

func TestControlPairedSwapsRequireConfirmation(t *testing.T) {
	for _, pair := range [][2]uint16{{304, 305}, {305, 304}, {800, 801}, {801, 800}, {802, 803}, {803, 802}} {
		a, c, _, saves := mappingApp()
		a.OpenControls()
		basicSetup(a, c)
		old := a.mapping.draft.Clone()
		checkHold(a, c, pair[0])
		chooseControlAction(a, old.Inputs[pair[1]])
		if a.mapping.mode != "swap-confirm" || *saves != 1 || !reflect.DeepEqual(old, a.mapping.draft) {
			t.Fatal("swap changed setup without confirmation", pair)
		}
		a.actControlPopup(platform.KeyBack)
		if *saves != 1 || !reflect.DeepEqual(old, a.mapping.draft) {
			t.Fatal("cancel changed setup", pair)
		}
		checkHold(a, c, pair[0])
		chooseControlAction(a, old.Inputs[pair[1]])
		a.actControlPopup(platform.KeyEnter)
		got := a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)]
		if got.Inputs[pair[0]] != old.Inputs[pair[1]] || got.Inputs[pair[1]] != old.Inputs[pair[0]] || *saves != 2 || a.mapping.mode != "table" {
			t.Fatal("swap did not save both changes", pair, got)
		}
		if err := controls.ValidateInputs(got.Inputs); err != nil {
			t.Fatal(err)
		}
	}
}

func TestControlSwapAliasesProofFailureAndNames(t *testing.T) {
	a, c, _, saves := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	// A saved alias has not yet been physically tested on this visit.
	q := a.mapping.draft.Clone()
	q.Inputs[330] = "back"
	q.Names[304], q.Names[330] = "Red", "Blue"
	a.commitControl(q)
	delete(a.mapping.check.proven, 305)
	checkHold(a, c, 304)
	chooseControlAction(a, "back")
	if a.mapping.mode != "actions" || a.mapping.message == "" {
		t.Fatal("untested replacement offered")
	}
	a.actControlPopup(platform.KeyBack)
	mappingTap(a, c, rawControl(a.mapping.device, 305, platform.KeyOther))
	mappingTap(a, c, rawControl(a.mapping.device, 330, platform.KeyOther))
	checkHold(a, c, 304)
	chooseControlAction(a, "back")
	if a.mapping.mode != "swap-target" || !reflect.DeepEqual(a.controlSwapCandidates(), []uint16{305, 330}) {
		t.Fatal("ambiguous alias was chosen silently")
	}
	a.actControlPopup(platform.KeyDown)
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.check.swap != 330 || a.mapping.mode != "swap-confirm" {
		t.Fatal("wrong swap target")
	}
	previous := a.mapping.draft.Clone()
	a.cfg.Controls.Save = func(controls.File) error { return errors.New("full") }
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "swap-confirm" || a.mapping.message == "" || !reflect.DeepEqual(a.mapping.draft, previous) || !reflect.DeepEqual(a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)], previous) {
		t.Fatal("failed swap leaked or lost retry")
	}
	a.cfg.Controls.Save = func(f controls.File) error { *saves++; return f.Validate() }
	a.actControlPopup(platform.KeyEnter)
	got := a.mapping.draft
	if got.Inputs[304] != "back" || got.Inputs[330] != "enter" || got.Inputs[305] != "back" || got.Names[304] != "Red" || got.Names[330] != "Blue" {
		t.Fatal("swap moved an unrelated alias or physical name", got)
	}
}

func TestControlSwapConfirmationConsumesHeldOK(t *testing.T) {
	a, c, _, saves := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	checkHold(a, c, 304)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 5
	checkPress(a, c, 304, true)
	if a.mapping.mode != "swap-confirm" {
		t.Fatal(a.mapping.mode)
	}
	checkPress(a, c, 304, true)
	*c = c.Add(2 * time.Second)
	a.Tick(*c)
	if *saves != 1 {
		t.Fatal("held OK confirmed swap")
	}
	checkPress(a, c, 304, false)
	checkPress(a, c, 304, true)
	if *saves != 2 || a.mapping.mode != "table" {
		t.Fatal("fresh OK failed")
	}
	checkPress(a, c, 304, false)
	if len(a.mapping.check.held) != 0 {
		t.Fatal("old OK release stranded after swap")
	}
}

func TestControlSwapDisconnectKeepsSavedProfile(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	before := a.mapping.draft.Clone()
	checkHold(a, c, 304)
	chooseControlAction(a, "back")
	*pads = nil
	a.actControlPopup(platform.KeyEnter)
	if a.screen == ScreenControls || *saves != 1 || !reflect.DeepEqual(a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)], before) {
		t.Fatal("disconnected swap changed saved controls")
	}
}

func TestControlKeyboardRolePreservesLegacyModelPreferences(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := &(*pads)[1]
	p.InstanceID = padID(*p) + "_0123456789abcdef"
	a.cfg.ControlProfiles.Profiles[padID(*p)] = controls.Profile{Panel: true, Reviewed: true, Bindings: map[string]uint16{"enter": 40}, Names: map[uint16]string{40: "Red"}}
	a.screen = ScreenOptions
	a.openControlRole()
	if !a.mapping.draft.BasicsComplete || a.mapping.draft.Inputs[40] != "enter" || a.mapping.draft.Names[40] != "Red" {
		t.Fatal("legacy keyboard preferences lost")
	}
	a.actControlPopup(platform.KeyUp)
	a.actControlPopup(platform.KeyEnter)
	got := a.cfg.ControlProfiles.Profiles[p.InstanceID]
	if got.Panel || got.Inputs[40] != "enter" || got.Names[40] != "Red" || a.screen != ScreenOptions {
		t.Fatal("role save changed legacy assignments", got)
	}
}

func TestControlOptionsReplaceLegacyRows(t *testing.T) {
	a, _, pads, _ := mappingApp()
	a.cfg.AccessMonth = 0
	for _, beta := range []bool{false, true, false} {
		a.cfg.ShowBetaFeatures = beta
		kinds := map[string]bool{}
		for _, e := range a.optionsEntries() {
			kinds[e.kind] = true
		}
		if !kinds["controls"] || !kinds["controls-role"] || kinds["ok-button"] || kinds["button-labels"] {
			t.Fatal("wrong control options", beta, kinds)
		}
	}
	a.cfg.ShowBetaFeatures = true
	*pads = (*pads)[:1]
	for _, e := range a.optionsEntries() {
		if e.kind == "controls-role" {
			t.Fatal("keyboard option without keyboard")
		}
	}
}

func TestControlKeyboardOptionsSelectionAndSave(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenOptions
	// The pad opening Options must not be the device whose role is changed.
	a.openControlRole()
	p := (*pads)[1]
	if a.mapping.mode != "role" || a.mapping.device.Node != p.Node {
		t.Fatal("wrong device")
	}
	a.cfg.Controls.Save = func(controls.File) error { return errors.New("full") }
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "role" || len(a.cfg.ControlProfiles.Profiles) != 0 {
		t.Fatal("failed role save leaked")
	}
	a.cfg.Controls.Save = func(f controls.File) error { *saves++; return f.Validate() }
	a.actControlPopup(platform.KeyEnter)
	q := a.cfg.ControlProfiles.Profiles[controlID(p)]
	if a.screen != ScreenOptions || q.Panel || q.BasicsComplete || !q.Reviewed {
		t.Fatal("keyboard acceptance", q)
	}
	a.openControlRole()
	a.actControlPopup(platform.KeyDown)
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "basic" || a.mapping.check.step != 0 || *saves != 1 {
		t.Fatal("unproven keyboard bypassed basics")
	}
	basicSetup(a, c)
	q = a.cfg.ControlProfiles.Profiles[controlID(p)]
	if !q.Panel || !q.BasicsComplete || a.mapping.mode != "table" {
		t.Fatal("panel setup missing")
	}
	a.exitControlCheck()
	a.openControlRole()
	a.actControlPopup(platform.KeyUp)
	a.actControlPopup(platform.KeyEnter)
	q = a.cfg.ControlProfiles.Profiles[controlID(p)]
	if q.Panel || !q.BasicsComplete || a.screen != ScreenOptions {
		t.Fatal("role change lost mapping")
	}
	other := p
	other.Node, other.InstanceID = "event9", padID(other)+"_fedcba9876543210"
	*pads = append(*pads, other)
	a.openControlRole()
	if a.mapping.mode != "devices" || len(a.mapping.devices) != 2 {
		t.Fatal("missing keyboard chooser")
	}
	mappingTap(a, c, rawControl((*pads)[0], 803, platform.KeyDown))
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyBack))
	if a.mapping.mode != "role" || controlID(a.mapping.device) != other.InstanceID {
		t.Fatal("wrong chosen keyboard", a.mapping.mode)
	}
	before := *saves
	a.actControlPopup(platform.KeyBack)
	if a.screen != ScreenOptions || *saves != before {
		t.Fatal("Back changed keyboard use")
	}
}
