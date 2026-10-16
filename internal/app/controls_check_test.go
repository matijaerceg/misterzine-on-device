package app

import (
	"errors"
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"testing"
	"time"
)

func checkPress(a *App, c *time.Time, code uint16, pressed bool) {
	e := rawControl(a.mapping.device, code, platform.KeyOther)
	e.At = *c
	e.Pressed = pressed
	a.Handle(e)
}
func checkHold(a *App, c *time.Time, codes ...uint16) {
	for _, code := range codes {
		checkPress(a, c, code, true)
	}
	*c = c.Add(1100 * time.Millisecond)
	a.Tick(*c)
	for _, code := range codes {
		checkPress(a, c, code, false)
	}
}
func basicSetup(a *App, c *time.Time) {
	for _, code := range []uint16{801, 800, 803, 802, 304, 305} {
		mappingTap(a, c, rawControl(a.mapping.device, code, platform.KeyOther))
	}
}
func TestControlBasicSequenceAndAtomicSave(t *testing.T) {
	a, c, _, saves := mappingApp()
	a.OpenControls()
	if a.mapping.mode != "basic" {
		t.Fatal(a.mapping.mode)
	}
	mappingTap(a, c, rawControl(a.mapping.device, 801, platform.KeyOther))
	mappingTap(a, c, rawControl(a.mapping.device, 801, platform.KeyOther))
	if a.mapping.check.step != 1 || *saves != 0 {
		t.Fatal("duplicate/early save")
	}
	for _, code := range []uint16{800, 803, 802, 304, 305} {
		mappingTap(a, c, rawControl(a.mapping.device, code, platform.KeyOther))
	}
	if a.mapping.mode != "table" || *saves != 1 || len(a.mapping.check.tested) != 6 {
		t.Fatal(a.mapping.mode, *saves)
	}
	if a.mapping.draft.Inputs[803] != "down" || a.mapping.draft.Inputs[802] != "up" {
		t.Fatal("wrong Down/Up prompt order")
	}
	checkPress(a, c, 330, true)
	if !a.mapping.check.tested[330] || a.mapping.check.selected != 330 {
		t.Fatal("first down missing")
	}
	checkPress(a, c, 330, false)
	checkHold(a, c, 330)
	if a.mapping.mode != "button" || a.mapping.check.editing != 330 {
		t.Fatal("wrong edit target")
	}
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "table" || a.mapping.draft.Inputs[330] != "start" || *saves != 2 {
		t.Fatal("action not saved")
	}
	checkHold(a, c, 304, 305)
	if a.screen == ScreenControls || *saves != 2 {
		t.Fatal("exit saved")
	}
}
func TestControlRequiredProtectionAndMultipleBindings(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	checkHold(a, c, 304)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "actions" || a.mapping.message == "" {
		t.Fatal("removed final OK")
	}
	a.actControlPopup(platform.KeyBack)
	checkHold(a, c, 330)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 4
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.draft.Inputs[304] != "enter" || a.mapping.draft.Inputs[330] != "enter" {
		t.Fatal("assignment swapped")
	}
	checkHold(a, c, 304)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.draft.Inputs[304] != "start" {
		t.Fatal("proven replacement blocked")
	}
}
func TestControlKeyboardRoleAndTyping(t *testing.T) {
	for _, panel := range []bool{false, true} {
		a, c, pads, saves := mappingApp()
		p := (*pads)[1]
		a.controlInput.current = padID(p)
		a.controlInput.node = p.Node
		a.OpenControls()
		basicSetup(a, c)
		if a.mapping.mode != "table" || *saves != 1 {
			t.Fatal("manual keyboard setup missing")
		}
		a.exitControlCheck()
		a.screen = ScreenOptions
		a.openControlRole()
		a.mapping.cursor = 0
		if panel {
			a.mapping.cursor = 1
		}
		a.actControlPopup(platform.KeyEnter)
		e := rawControl(p, 30, platform.KeyOther)
		e.Text = 'a'
		got := a.mapControlEvent(e)
		if (got.Text == 0) != panel {
			t.Fatal("wrong typing role", got)
		}
		erase := rawControl(p, 14, platform.KeyBackspace)
		gotErase := a.mapControlEvent(erase)
		if (gotErase.Key == platform.KeyBackspace) == panel {
			t.Fatal("wrong keyboard erase behavior")
		}
		e.Code = 304
		e.Text = 'z'
		got = a.mapControlEvent(e)
		if got.Key != platform.KeyEnter || got.Text != 0 {
			t.Fatal("keyboard assignment failed")
		}
	}
}
func TestControlNameImmediateSaveDefaultBackAndFailure(t *testing.T) {
	a, c, _, saves := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	checkHold(a, c, 304)
	a.mapping.cursor = 1
	a.actControlPopup(platform.KeyEnter)
	a.mapping.check.group = 4
	a.mapping.check.pick = 0
	a.cfg.Controls.Save = func(controls.File) error { return errors.New("disk") }
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "labels" || a.mapping.draft.Names[304] != "" {
		t.Fatal("failed save leaked")
	}
	a.cfg.Controls.Save = func(f controls.File) error { *saves++; return f.Validate() }
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "table" || a.mapping.draft.Names[304] != "LP" {
		t.Fatal("label not saved")
	}
	checkHold(a, c, 304)
	a.mapping.cursor = 1
	a.actControlPopup(platform.KeyEnter)
	a.actControlPopup(platform.KeyBack)
	if a.mapping.draft.Names[304] != "LP" {
		t.Fatal("back changed name")
	}
	checkHold(a, c, 304)
	a.mapping.cursor = 1
	a.actControlPopup(platform.KeyEnter)
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.draft.Names[304] != "" {
		t.Fatal("default did not reset")
	}
}
func TestControlHoldArbitrationAndStaleRelease(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	checkPress(a, c, 304, true)
	checkPress(a, c, 305, true)
	if !a.mapping.check.held[304] || !a.mapping.check.held[305] {
		t.Fatal("chord not highlighted")
	}
	checkPress(a, c, 305, false)
	*c = c.Add(3 * time.Second)
	a.Tick(*c)
	checkPress(a, c, 304, false)
	if a.mapping.mode != "table" || a.screen != ScreenControls {
		t.Fatal("partial chord acted")
	}
	(*pads)[0].HeldKnown = true
	checkPress(a, c, 304, true)
	*c = c.Add(100 * time.Millisecond)
	a.Tick(*c)
	*c = c.Add(200 * time.Millisecond)
	a.Tick(*c)
	*c = c.Add(3 * time.Second)
	a.Tick(*c)
	if a.mapping.mode != "table" {
		t.Fatal("stale hold edited")
	}
}
func TestControlScrollAndVirginRows(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	if a.mapping.check.tested[315] {
		t.Fatal("optional not virgin")
	}
	for code := uint16(330); code < 350; code++ {
		mappingTap(a, c, rawControl(a.mapping.device, code, platform.KeyOther))
	}
	d := &a.mapping.check
	if d.target <= 0 || d.scroll >= d.target {
		t.Fatal("did not animate")
	}
	*c = c.Add(100 * time.Millisecond)
	a.Tick(*c)
	if d.scroll <= 0 || d.scroll >= d.target {
		t.Fatal("not intermediate")
	}
	*c = c.Add(150 * time.Millisecond)
	a.Tick(*c)
	if d.scroll != d.target {
		t.Fatal("not settled")
	}
	checkPress(a, c, 802, true)
	checkPress(a, c, 803, true)
	if d.selected != 803 || !d.held[802] || !d.held[803] {
		t.Fatal("second press lost")
	}
}
func TestControlAggregateReleaseOwnership(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	p := a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)]
	p.Inputs[330] = "up"
	a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)] = p
	a.exitControlCheck()
	checkPress(a, c, 802, true)
	checkPress(a, c, 330, true)
	checkPress(a, c, 802, false)
	if !a.down[platform.KeyUp] {
		t.Fatal("first release ended aggregate")
	}
	checkPress(a, c, 330, false)
	if a.down[platform.KeyUp] {
		t.Fatal("last release stuck")
	}
}
func TestControlDiscoveryDeferralAndDisconnect(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.controlInput.current = ""
	a.screen = ScreenList
	*c = c.Add(2 * time.Second)
	if a.discoverControls(*c) {
		t.Fatal("invited without input")
	}
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	if a.mapping.mode != "invitation" || a.mapping.check.step != 0 {
		t.Fatal("no invitation")
	}
	e := rawControl((*pads)[1], 1, platform.KeyBack)
	e.Pressed = true
	e.At = *c
	a.Handle(e)
	*c = c.Add(1100 * time.Millisecond)
	a.Tick(*c)
	if a.screen == ScreenControls || *saves != 0 {
		t.Fatal("defer saved")
	}
	e.Pressed = false
	a.Handle(e)
	*c = c.Add(2 * time.Second)
	if a.discoverControls(*c) {
		t.Fatal("queued invitation")
	}
	mappingTap(a, c, rawControl((*pads)[1], 28, platform.KeyEnter))
	if a.mapping.mode != "initial-role" || a.mapping.device.Node != (*pads)[1].Node {
		t.Fatal("deferred every controller")
	}
	*pads = (*pads)[:1]
	a.Tick(*c)
	if a.screen == ScreenControls {
		t.Fatal("disconnect stayed")
	}
}
func TestControlBasicCancelAndFailedRetry(t *testing.T) {
	a, c, _, saves := mappingApp()
	a.OpenControls()
	checkHold(a, c, 304)
	if a.screen == ScreenControls || *saves != 0 {
		t.Fatal("cancel saved")
	}
	a.OpenControls()
	a.cfg.Controls.Save = func(controls.File) error { return errors.New("full") }
	basicSetup(a, c)
	if a.mapping.mode != "retry-basic" {
		t.Fatal("no retry")
	}
	a.cfg.Controls.Save = func(f controls.File) error { *saves++; return f.Validate() }
	checkHold(a, c, 304)
	if a.mapping.mode != "table" || *saves != 1 {
		t.Fatal("retry failed")
	}
}

func TestControlHeldOnEntryCancelledAndForeignDirection(t *testing.T) {
	a, c, pads, saves := mappingApp()
	(*pads)[0].Held = []uint16{304}
	a.OpenControls()
	checkHold(a, c, 304)
	if a.mapping.mode != "basic" || a.mapping.check.step != 0 {
		t.Fatal("entry hold acted")
	}
	(*pads)[0].Held = nil
	e := rawControl((*pads)[1], 103, platform.KeyOther)
	e.Pressed = true
	a.Handle(e)
	*c = c.Add(3 * time.Second)
	a.Tick(*c)
	if a.screen != ScreenControls {
		t.Fatal("foreign direction deferred")
	}
	e.Pressed = false
	a.Handle(e)
	checkPress(a, c, 801, true)
	e = rawControl(a.mapping.device, 801, platform.KeyOther)
	e.Cancelled = true
	a.Handle(e)
	if a.mapping.check.step != 0 || *saves != 0 {
		t.Fatal("synthetic release assigned")
	}
}
func TestControlLegacyMigrationAndIndependentDevices(t *testing.T) {
	a, c, pads, saves := mappingApp()
	p := (*pads)[0]
	legacy := controls.Profile{Reviewed: true, Style: "custom", Labels: map[string]string{"enter": "R"}}
	a.cfg.ControlProfiles.Profiles[padID(p)] = legacy
	a.OpenControls()
	if a.mapping.mode != "basic" || *saves != 0 {
		t.Fatal("manual setup bypassed")
	}
	code := uint16(304)
	if a.mapping.draft.Names[code] != "R" || a.mapping.draft.MappingVersion != 2 {
		t.Fatal("name migration lost")
	}
	basicSetup(a, c)
	checkHold(a, c, 330)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	if *saves != 2 {
		t.Fatal("no migration save")
	}
	other := p
	other.InstanceID = "0001_0002_1234567890abcdef"
	other.Node = "event7"
	*pads = append(*pads, other)
	a.exitControlCheck()
	a.controlInput.current = other.InstanceID
	a.controlInput.node = other.Node
	a.OpenControls()
	if a.mapping.mode != "basic" {
		t.Fatal("template bypassed required setup")
	}
	basicSetup(a, c)
	q := a.cfg.ControlProfiles.Profiles[other.InstanceID]
	q.Inputs[330] = "tab"
	a.cfg.ControlProfiles.Profiles[other.InstanceID] = q
	if a.cfg.ControlProfiles.Profiles[padID(p)].Inputs[330] != "start" {
		t.Fatal("instance alias")
	}
}
func TestControlRedoCancelPreservesProfileAndNames(t *testing.T) {
	a, c, _, saves := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	saved := a.mapping.draft.Clone()
	a.exitControlCheck()
	a.screen = ScreenOptions
	a.OpenControls()
	if a.mapping.mode != "basic" {
		t.Fatal("redo missing")
	}
	mappingTap(a, c, rawControl(a.mapping.device, 350, platform.KeyOther))
	a.cancelBasicSetup()
	if a.screen != ScreenOptions || a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)].Inputs[801] != saved.Inputs[801] || *saves != 1 || a.mapping.check.basic != nil {
		t.Fatal("redo cancellation leaked")
	}
}
func TestControlLabelsFollowRemapAndKeyboardDefaultReset(t *testing.T) {
	a, _, pads, _ := mappingApp()
	p := (*pads)[0]
	p.Menu = "314+315"
	old := controls.Profile{Style: "playstation", Bindings: map[string]uint16{"start": 307}}
	q := a.normalizedControl(p, old)
	if q.Names[307] != "Triangle" || !q.LegacyMenuChord {
		t.Fatal("legacy appearance/chord lost", q)
	}
	q.Inputs[307] = "tab"
	if a.controlActionLabel("tab", p, q) == "-" {
		t.Fatal("physical name lost")
	}
	f := controls.New()
	f.Profiles[padID(p)] = q
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestControlExitHeldRepeatCannotNavigate(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	checkPress(a, c, 304, true)
	checkPress(a, c, 305, true)
	*c = c.Add(1100 * time.Millisecond)
	a.Tick(*c)
	previous := a.screen
	checkPress(a, c, 304, true)
	if a.screen != previous {
		t.Fatal("held repeat escaped exit")
	}
	checkPress(a, c, 304, false)
	checkPress(a, c, 305, false)
	if len(a.controlInput.blocked) != 0 {
		t.Fatal("released exit inputs remain blocked")
	}
}

func TestControlReplacementMustBePressedThisVisit(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	p := a.mapping.draft.Clone()
	p.Inputs[330] = "enter"
	a.cfg.ControlProfiles.Profiles[controlID(a.mapping.device)] = p
	a.mapping.draft = p
	a.mapping.check.proven[330] = false
	checkHold(a, c, 304)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "actions" {
		t.Fatal("unpressed replacement accepted")
	}
	a.actControlPopup(platform.KeyBack)
	mappingTap(a, c, rawControl(a.mapping.device, 330, platform.KeyOther))
	checkHold(a, c, 304)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.draft.Inputs[304] != "start" {
		t.Fatal("proven replacement rejected")
	}
}

func TestControlKeyboardInvitationAcceptAndManualRemap(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenList
	p := (*pads)[1]
	mappingTap(a, c, rawControl(p, 28, platform.KeyEnter))
	if a.mapping.mode != "initial-role" || *saves != 0 {
		t.Fatal("trigger selected keyboard choice")
	}
	mappingTap(a, c, rawControl(p, 28, platform.KeyEnter))
	if a.screen == ScreenControls || *saves != 1 {
		t.Fatal("keyboard not accepted")
	}
	q := a.cfg.ControlProfiles.Profiles[controlID(p)]
	if q.Panel || q.BasicsComplete || !q.Reviewed {
		t.Fatal("wrong keyboard role", q)
	}
	mappingTap(a, c, rawControl(p, 30, platform.KeyOther))
	if a.screen == ScreenControls {
		t.Fatal("accepted keyboard invited again")
	}
	a.OpenControls()
	if a.mapping.mode != "basic" {
		t.Fatal("manual keyboard remapping skipped")
	}
	basicSetup(a, c)
	if !a.mapping.draft.Panel {
		t.Fatal("manual mapping did not set controller role")
	}
}
func TestControlFastHighlightAndOneSecondHold(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	checkPress(a, c, 304, true)
	*c = c.Add(20 * time.Millisecond)
	checkPress(a, c, 305, true)
	checkPress(a, c, 304, false)
	if !a.mapping.check.held[305] || a.mapping.check.held[304] {
		t.Fatal("quick switch lost highlight")
	}
	checkPress(a, c, 305, false)
	checkPress(a, c, 330, true)
	*c = c.Add(999 * time.Millisecond)
	a.Tick(*c)
	if a.mapping.mode != "table" {
		t.Fatal("early hold")
	}
	*c = c.Add(time.Millisecond)
	a.Tick(*c)
	if a.mapping.mode != "button" {
		t.Fatal("one second did not edit")
	}
	checkPress(a, c, 330, true)
	if a.mapping.mode != "button" {
		t.Fatal("transition repeat selected popup")
	}
}

func TestControlInitialKeyboardSaveRetry(t *testing.T) {
	a, c, pads, saves := mappingApp()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[1], 28, platform.KeyEnter))
	a.cfg.Controls.Save = func(controls.File) error { return errors.New("full") }
	mappingTap(a, c, rawControl((*pads)[1], 28, platform.KeyEnter))
	if a.mapping.mode != "initial-role" || len(a.cfg.ControlProfiles.Profiles) != 0 {
		t.Fatal("failed acceptance leaked")
	}
	a.cfg.Controls.Save = func(f controls.File) error { *saves++; return f.Validate() }
	mappingTap(a, c, rawControl((*pads)[1], 28, platform.KeyEnter))
	if a.screen == ScreenControls || *saves != 1 {
		t.Fatal("retry failed")
	}
}
func TestControlNoInvitationsDuringSetupOrAfterCancel(t *testing.T) {
	a, c, pads, _ := mappingApp()
	a.screen = ScreenOptions
	a.OpenControls()
	mappingTap(a, c, rawControl((*pads)[1], 28, platform.KeyEnter))
	if a.mapping.device.Node != (*pads)[0].Node {
		t.Fatal("foreign input changed setup")
	}
	checkHold(a, c, 304)
	if a.screen != ScreenOptions {
		t.Fatal("cancel did not return to Options")
	}
	a.screen = ScreenList
	a.Tick(*c)
	if a.screen == ScreenControls {
		t.Fatal("queued invitation")
	}
	mappingTap(a, c, rawControl((*pads)[0], 304, platform.KeyEnter))
	if a.screen != ScreenControls || a.mapping.mode != "invitation" {
		t.Fatal("fresh press did not reinvite cancelled controller")
	}
	a.exitControlCheck()
	a.screen = ScreenList
	mappingTap(a, c, rawControl((*pads)[1], 28, platform.KeyEnter))
	if a.mapping.mode != "initial-role" {
		t.Fatal("other controller deferred too")
	}
}

func TestControlDirectionalDefaultNamesFollowAssignment(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	for i, id := range []string{"up", "down", "left", "right"} {
		code := uint16(340 + i)
		a.mapping.draft.Inputs[code] = id
		want := actionName(id)
		if got := a.currentControlName(code); got != want {
			t.Fatalf("table %s = %q", id, got)
		}
		// Remove aliases so runtime chooses this otherwise unrecognized input too.
		p := a.mapping.draft.Clone()
		for other, action := range p.Inputs {
			if other != code && action == id {
				delete(p.Inputs, other)
			}
		}
		if got := a.controlActionLabel(id, a.mapping.device, p); got != want {
			t.Fatalf("legend %s = %q", id, got)
		}
		a.mapping.draft.Names[code] = "R"
		if a.currentControlName(code) != "R" {
			t.Fatal("custom direction name lost")
		}
		delete(a.mapping.draft.Names, code)
		if a.currentControlName(code) != want {
			t.Fatal("default reset did not restore direction")
		}
	}
}

func TestControlLabelEditingOnlyForLegendActions(t *testing.T) {
	a, c, _, _ := mappingApp()
	a.OpenControls()
	basicSetup(a, c)
	for _, code := range []uint16{802, 330} {
		checkHold(a, c, code)
		rows := a.controlButtonRows()
		if len(rows) != 1 || rows[0] != "Change action" {
			t.Fatal("non-legend label editor offered", rows)
		}
		a.actControlPopup(platform.KeyDown)
		a.actControlPopup(platform.KeyEnter)
		if a.mapping.mode != "actions" {
			t.Fatal("hidden row broke navigation")
		}
		a.actControlPopup(platform.KeyBack)
	}
	checkHold(a, c, 330)
	a.actControlPopup(platform.KeyEnter)
	a.mapping.cursor = 10
	a.actControlPopup(platform.KeyEnter)
	checkHold(a, c, 330)
	if rows := a.controlButtonRows(); len(rows) != 2 || rows[1] != "Change OSD label" {
		t.Fatal("assigned action has no label editor", rows)
	}
	a.actControlPopup(platform.KeyDown)
	a.actControlPopup(platform.KeyEnter)
	if a.mapping.mode != "labels" {
		t.Fatal("label editor not reachable")
	}
	a.actControlPopup(platform.KeyBack)
	a.actControlPopup(platform.KeyBack)
	// Page up/down names reach the troubleshooting result legend; Menu never does.
	for _, tc := range []struct {
		cursor int
		rows   int
	}{{8, 2}, {12, 1}} {
		checkHold(a, c, 330)
		a.actControlPopup(platform.KeyEnter)
		a.mapping.cursor = tc.cursor
		a.actControlPopup(platform.KeyEnter)
		checkHold(a, c, 330)
		if rows := a.controlButtonRows(); len(rows) != tc.rows {
			t.Fatalf("%s label rows = %v", checkOrder[tc.cursor], rows)
		}
		a.actControlPopup(platform.KeyBack)
	}
	if a.mapping.draft.Inputs[330] != "menu" {
		t.Fatal("menu assignment lost")
	}
	if actionName("") != "--" || actionName("menu") != "Menu" {
		t.Fatal("unassigned action must read --")
	}
}
