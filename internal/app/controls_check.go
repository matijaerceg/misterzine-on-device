package app

import (
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
	"sort"
	"strings"
	"time"
)

var checkOrder = []string{"up", "down", "left", "right", "enter", "back", "tab", "space", "pageup", "pagedown", "start", "select", "menu"}
var checkNames = []string{"Up", "Down", "Left", "Right", "OK", "Back", "Filters", "View", "Page up", "Page down", "Launch", "Toggles", "Menu"}

const controlHoldDuration = time.Second

var basicOrder = []string{"right", "left", "down", "up", "enter", "back"}

type controlsView struct {
	nav         repeater
	mode        string
	roleOptions bool
	device      support.Pad
	devices     []support.Pad
	draft       controls.Profile
	next, now   time.Time
	message     string
	cursor, top int
	check       controllerCheck
}
type controllerCheck struct {
	introHeld                                map[string]platform.Event
	introAt, introMissing                    time.Time
	introChord, introCancelled, introBlocked bool

	proven                  map[uint16]bool
	entryHeld               map[uint16]bool
	automatic               bool
	swap                    uint16
	step                    int
	basic                   map[uint16]string
	tested                  map[uint16]bool
	discovered              []uint16
	held                    map[uint16]bool
	ignored                 map[string]bool
	opened                  time.Time
	at                      time.Time
	gesture                 int
	cancelled               bool
	missingSince            time.Time
	dismissAt               time.Time
	dismissEvent            platform.Event
	dismissToken            string
	selected, editing, last uint16
	repeats                 int
	scroll, from, target    float64
	scrollAt                time.Time
	group, pick             int
}
type controlTableRow struct {
	code   uint16
	action string
}

func actionName(id string) string {
	for i, a := range checkOrder {
		if a == id {
			return checkNames[i]
		}
	}
	return "--"
}

// controlLabelEditable reports whether an action's custom name can appear in
// MisterZine's legends: only legend tokens, never directions, Menu (its hold
// notice names no button) or an unassigned input.
func controlLabelEditable(action string) bool {
	if action == "" || action == "menu" || controlDirectionName(action) != "" {
		return false
	}
	for _, id := range controlTokens {
		if id == action {
			return true
		}
	}
	return false
}

// Direction defaults describe the assigned direction even when the raw input
// has no MiSTer slot (or is a button or key repurposed as a direction).
func controlDirectionName(id string) string {
	switch id {
	case "up", "down", "left", "right":
		return actionName(id)
	}
	return ""
}
func requiredAction(id string) bool {
	for _, a := range basicOrder {
		if a == id {
			return true
		}
	}
	return false
}
func (a *App) observeControl(node string) {
	if h := a.cfg.Controls; h != nil && h.Observe != nil {
		h.Observe(node)
	}
}
func (a *App) clearControlTransition() {
	d := &a.mapping.check
	a.mapping.nav = repeater{}
	d.ignored = map[string]bool{}
	for token := range a.controlInput.held {
		d.ignored[token] = true
	}
	d.held = map[uint16]bool{}
	d.introHeld = map[string]platform.Event{}
	d.introAt = time.Time{}
	d.introMissing = time.Time{}
	d.introChord = false
	d.introCancelled = false
	d.at = time.Time{}
	d.gesture = 0
	d.cancelled = false
	d.missingSince = time.Time{}
	d.dismissAt = time.Time{}
	d.dismissToken = ""
	a.clearControlHolds()
	a.all = true
}
func (a *App) setControlMode(mode string) {
	a.mapping.mode = mode
	a.mapping.next = a.controlNow().Add(frameDur)
	a.mapping.cursor = 0
	a.mapping.message = ""
	a.clearControlTransition()
}
func (a *App) OpenControls() {
	if !a.featureAllowed(access.Controls) {
		return
	}
	a.controlInput.previous = a.screen
	a.mapping = controlsView{mode: "devices", devices: a.controlDevices(), next: a.controlNow().Add(time.Second)}
	a.screen = ScreenControls
	a.clearControlHolds()
	a.all = true
	candidates := []support.Pad{}
	for _, p := range a.mapping.devices {
		if p.Direct || p.Keyboard {
			if controlID(p) == a.controlInput.current && p.Node == a.controlInput.node {
				a.editControl(p)
				return
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 1 {
		a.editControl(candidates[0])
	}
}
func (a *App) exitControlCheck() {
	if a.controlInput.inviteWait == nil {
		a.controlInput.inviteWait = map[string]map[string]bool{}
	}
	waiting := map[string]bool{}
	for token := range a.controlInput.held {
		waiting[token] = true
	}
	if len(waiting) > 0 {
		a.controlInput.inviteWait[controlID(a.mapping.device)] = waiting
	}

	a.controlInput.blocked = map[string]bool{}
	for token := range a.controlInput.held {
		a.controlInput.blocked[token] = true
	}
	a.observeControl("")
	a.clearControlTransition()
	a.screen = a.controlInput.previous
	if a.screen == ScreenControls {
		a.screen = ScreenOptions
	}
	a.saver.lastInput = a.controlNow()
	a.buildPanel()
	a.all = true
}

// Legacy translation is read-only until a successful save. Names are frozen to
// physical codes before any action can move, including inherited preset labels.
func (a *App) normalizedControl(p support.Pad, old controls.Profile) controls.Profile {
	q := old.Clone()
	if q.MappingVersion == 2 {
		return q
	}
	q.Inputs = map[uint16]string{}
	effective := controls.Effective(a.controlBase(p, old), old)
	for _, id := range checkOrder {
		if code := effective[id]; controls.ValidCode(code) {
			q.Inputs[code] = id
			if q.Names[code] == "" {
				if old.Reviewed || len(old.Bindings) > 0 || old.Style != "" {
					q.Names[code] = storedButtonName(a.controlActionLabel(id, p, old))
				}
			}
		}
	}
	if old.Style != "" && old.Style != "custom" {
		for _, code := range p.Slots {
			if controls.ValidCode(code) && q.Names[code] == "" {
				sample := old.Clone()
				sample.Bindings = map[string]uint16{"enter": code}
				sample.Labels = nil
				sample.Names = nil
				q.Names[code] = storedButtonName(a.controlActionLabel("enter", p, sample))
			}
		}
	}
	_, explicitMenu := old.Bindings["menu"]
	q.LegacyMenuChord = strings.Contains(p.Menu, "+") && !explicitMenu
	q.MappingVersion = 2
	q.BasicsComplete = controls.ValidateInputs(q.Inputs) == nil
	q.Bindings = map[string]uint16{}
	q.Labels = map[string]string{}
	q.Style = ""
	q.NameBase = ""
	return q
}
func (a *App) editControl(p support.Pad) {
	a.editControlEntry(p, false)
}
func (a *App) editControlEntry(p support.Pad, automatic bool) {
	a.mapping.device = p
	a.mapping.draft = a.normalizedControl(p, a.controlTemplate(p))
	a.mapping.draft.Name = p.Name
	d := controllerCheck{automatic: automatic, entryHeld: map[uint16]bool{}, tested: map[uint16]bool{}, proven: map[uint16]bool{}, held: map[uint16]bool{}, opened: a.controlNow()}
	a.mapping.check = d
	a.observeControl(p.Node)
	a.setControlMode("table")
	for _, pad := range a.controlDevices() {
		if pad.Node == p.Node && controlID(pad) == controlID(p) {
			for _, c := range pad.Held {
				a.mapping.check.entryHeld[c] = true
			}
		}
	}
	if !p.Direct && !p.Keyboard {
		a.mapping.mode = "unsupported"
		return
	}
	if automatic {
		mode := "invitation"
		if p.Keyboard {
			mode = "initial-role"
		}
		a.setControlMode(mode)
		return
	}
	a.startBasicSetup()
}
func (a *App) startBasicSetup() {
	v := &a.mapping
	v.draft = a.normalizedControl(v.device, a.controlTemplate(v.device))
	v.draft.Name = v.device.Name
	if v.device.Keyboard {
		v.draft.Panel = true
	}
	v.check.step = 0
	v.check.basic = map[uint16]string{}
	a.setControlMode("basic")
}
func (a *App) completeBasicSetup() {
	v := &a.mapping
	for code, id := range v.draft.Inputs {
		if requiredAction(id) {
			delete(v.draft.Inputs, code)
		}
	}
	for code, id := range v.check.basic {
		v.draft.Inputs[code] = id
	}
	v.draft.BasicsComplete = true
	a.setControlMode("retry-basic")
	a.commitControl(v.draft)
}
func (a *App) cancelBasicSetup() {
	a.mapping.check.basic = nil
	a.mapping.check.step = 0
	a.exitControlCheck()
}
func (a *App) connectedControl() bool {
	for _, p := range a.controlDevices() {
		if p.Node == a.mapping.device.Node && controlID(p) == controlID(a.mapping.device) {
			return true
		}
	}
	return false
}
func (a *App) commitControl(candidate controls.Profile) bool {
	v := &a.mapping
	if !a.connectedControl() {
		a.exitControlCheck()
		return false
	}
	if err := controls.ValidateInputs(candidate.Inputs); err != nil && v.mode != "initial-role" && !(v.mode == "role" && !candidate.Panel && !candidate.BasicsComplete) {
		v.message = err.Error()
		return false
	}
	h := a.cfg.Controls
	if h == nil || h.Save == nil || h.Problem != "" {
		v.message = "Cannot save. Repair controls.json first."
		return false
	}
	f := a.cfg.ControlProfiles.Clone()
	candidate.Reviewed = true
	if v.mode == "initial-role" {
		candidate.BasicsComplete = false
	}
	f.Profiles[controlID(v.device)] = candidate.Clone()
	if err := h.Save(f); err != nil {
		v.message = "Save failed. Previous setup kept. OK to retry."
		a.all = true
		return false
	}
	a.cfg.ControlProfiles = f
	a.controlInput.legendGen++
	v.draft = candidate.Clone()
	a.controlsChanged()
	if v.mode == "initial-role" || v.mode == "role" {
		a.exitControlCheck()
		return true
	}
	for code := range v.check.basic {
		v.check.tested[code] = true
		v.check.proven[code] = true
	}
	v.check.basic = nil
	a.setControlMode("table")
	a.scrollToControl(v.check.editing)
	return true
}

// Invitations are driven only by fresh physical presses, never enumeration.
func (a *App) nextControlDiscovery() time.Time     { return time.Time{} }
func (a *App) discoverControls(now time.Time) bool { return false }
func (a *App) inviteControl(raw platform.Event) bool {
	if a.cfg.Controls == nil || !a.featureAllowed(access.Controls) || raw.Cancelled || raw.Code == 0 || raw.Node == "" || len(a.controlInput.inviteWait[raw.DeviceID]) > 0 {
		return false
	}
	if a.screen != ScreenList || a.lock != nil || a.cfg.ArcadeIntro || a.cfg.ArcadeBack || a.cab.active || a.splash.enabled || a.saver.active {
		return false
	}
	for _, p := range a.controlDevices() {
		id := controlID(p)
		if id != raw.DeviceID || p.Node != raw.Node {
			continue
		}
		profile, known := a.cfg.ControlProfiles.Profiles[id]
		if known {
			p2 := a.normalizedControl(p, profile)
			known = p2.BasicsComplete && controls.ValidateInputs(p2.Inputs) == nil || p.Keyboard && profile.Reviewed && !profile.Panel
		}
		if known {
			return false
		}
		a.controlInput.previous = a.screen
		a.screen = ScreenControls
		a.mapping = controlsView{}
		a.editControlEntry(p, true)
		return true
	}
	return false
}
func (a *App) controlTableRows() []controlTableRow {
	v := &a.mapping
	rows := []controlTableRow{}
	for _, id := range checkOrder {
		codes := []int{}
		for code, action := range v.draft.Inputs {
			if action == id {
				codes = append(codes, int(code))
			}
		}
		sort.Ints(codes)
		if len(codes) == 0 {
			rows = append(rows, controlTableRow{action: id})
		}
		for _, c := range codes {
			rows = append(rows, controlTableRow{uint16(c), id})
		}
	}
	seen := map[uint16]bool{}
	for _, c := range v.check.discovered {
		if v.draft.Inputs[c] == "" && !seen[c] {
			rows = append(rows, controlTableRow{code: c})
			seen[c] = true
		}
	}
	codes := []int{}
	for code := range v.draft.Names {
		if v.draft.Inputs[code] == "" && !seen[code] {
			codes = append(codes, int(code))
		}
	}
	sort.Ints(codes)
	for _, c := range codes {
		rows = append(rows, controlTableRow{code: uint16(c)})
	}
	return rows
}
func (a *App) scrollToControl(code uint16) {
	if code == 0 {
		return
	}
	d := &a.mapping.check
	rows := a.controlTableRows()
	visible := a.controlVisibleRows()
	for i, r := range rows {
		if r.code == code {
			target := float64(centeredTop(i, len(rows), visible))
			if target != d.target {
				d.from = d.scroll
				d.target = target
				d.scrollAt = a.controlNow()
			}
			return
		}
	}
}
func (a *App) discoverControlCode(code uint16) {
	d := &a.mapping.check
	for _, c := range d.discovered {
		if c == code {
			return
		}
	}
	d.discovered = append(d.discovered, code)
}
func (a *App) advanceControlHold(now time.Time) {
	v := &a.mapping
	d := &v.check
	if d.at.IsZero() || d.cancelled || now.Sub(d.at) < controlHoldDuration || v.heldControlCount() != d.gesture {
		return
	}
	switch v.mode {
	case "table":
		if d.gesture == 2 {
			a.exitControlCheck()
		} else {
			d.editing = v.heldControlInput()
			a.setControlMode("button")
		}
	case "basic":
		a.cancelBasicSetup()
	case "retry-basic":
		a.commitControl(v.draft)
		a.clearControlTransition()
	}
}
func (a *App) popupControlKey(ev platform.Event) platform.Key {
	if ev.DeviceID == controlID(a.mapping.device) && ev.Code != 0 {
		if id := a.mapping.draft.Inputs[ev.Code]; id != "" {
			return controls.Key(id)
		}
		return platform.KeyOther
	}
	return ev.Key
}
func (a *App) handleControls(raw, ev platform.Event, fresh bool) bool {
	v := &a.mapping
	d := &v.check
	a.saver.lastInput = a.controlNow()
	if !a.featureAllowed(access.Controls) {
		a.exitControlCheck()
		return true
	}
	if v.mode == "devices" {
		if fresh && !raw.Cancelled {
			if ev.Key == platform.KeyBack {
				a.exitControlCheck()
			} else if ev.Key == platform.KeyDown {
				v.cursor = min(len(v.devices)-1, v.cursor+1)
			} else if ev.Key == platform.KeyUp {
				v.cursor = max(0, v.cursor-1)
			} else if ev.Key == platform.KeyEnter && len(v.devices) > 0 {
				if v.roleOptions {
					a.editControlRole(v.devices[v.cursor])
				} else {
					a.editControl(v.devices[v.cursor])
				}
			}
			a.all = true
		}
		return true
	}
	token := controlToken(raw)
	if raw.DeviceID == controlID(v.device) && raw.Node == v.device.Node && d.entryHeld[raw.Code] {
		if !raw.Pressed {
			delete(d.entryHeld, raw.Code)
			delete(d.ignored, token)
		}
		return true
	}
	if d.ignored[token] {
		if !raw.Pressed {
			delete(d.ignored, token)
		}
		return true
	}
	if v.mode == "invitation" || v.mode == "initial-role" {
		return a.handleControlInvitation(raw, fresh)
	}
	if fresh && !raw.Cancelled && raw.Keyboard && raw.Code == 1 && (raw.DeviceID != controlID(v.device) || raw.Node != v.device.Node) && (v.mode == "basic" || v.mode == "retry-basic" || v.mode == "unsupported") {
		a.cancelBasicSetup()
		return true
	}
	if raw.Cancelled {
		if raw.DeviceID != controlID(v.device) || raw.Node != v.device.Node {
			if token == d.dismissToken {
				d.dismissAt = time.Time{}
				d.dismissToken = ""
			}
			return true
		}
		delete(d.held, raw.Code)
		d.at = time.Time{}
		d.cancelled = true
		d.selected = 0
		if token == d.dismissToken {
			d.dismissAt = time.Time{}
			d.dismissToken = ""
		}
		a.all = true
		return true
	}
	if !raw.At.IsZero() && raw.At.Before(d.opened) {
		return true
	}
	interactive := v.mode == "table" || v.mode == "basic" || v.mode == "retry-basic" || v.mode == "unsupported"
	if !interactive {
		key := a.popupControlKey(ev)
		if !ev.Pressed {
			v.nav.release(key)
		}
		if fresh {
			a.actControlPopup(key)
			if key >= platform.KeyUp && key <= platform.KeyRight {
				v.nav.press(key, a.controlNow())
			}
		}
		return true
	}
	if raw.DeviceID != controlID(v.device) || raw.Node != v.device.Node {
		if raw.Code == 0 || raw.DeviceID == "" || raw.Node == "" || raw.Direction || raw.Code >= 0x300 || ev.Key >= platform.KeyUp && ev.Key <= platform.KeyRight {
			return true
		}
		known := false
		for _, p := range a.controlDevices() {
			if controlID(p) == raw.DeviceID && p.Node == raw.Node {
				if _, direction := mistercadeDirection(p, raw.Code); direction {
					return true
				}
				if raw.Code == 103 || raw.Code == 105 || raw.Code == 106 || raw.Code == 108 || raw.Code >= 0x220 && raw.Code <= 0x223 {
					return true
				}
				for _, slot := range []string{"Up", "Down", "Left", "Right"} {
					if p.Slots[slot] == raw.Code {
						return true
					}
				}
				known = true
			}
		}
		if !known {
			return true
		}
		if fresh && d.dismissToken == "" {
			d.dismissAt = a.controlNow()
			d.dismissToken = token
			d.dismissEvent = raw
		}
		if !raw.Pressed && token == d.dismissToken {
			d.dismissAt = time.Time{}
			d.dismissToken = ""
		}
		a.all = true
		return true
	}
	if raw.Code == 0 {
		return true
	}
	before := v.mode
	a.advanceControlHold(a.controlNow())
	if v.mode != before || a.screen != ScreenControls {
		return true
	}
	if raw.Pressed {
		if !fresh {
			return true
		}
		if len(d.held) == 0 {
			d.at = a.controlNow()
			d.gesture = 1
			d.cancelled = false
		}
		before := v.heldControlCount()
		d.held[raw.Code] = true
		count := v.heldControlCount()
		if count == 2 && before != count {
			d.at = a.controlNow()
			d.gesture = 2
		}
		if count > 2 {
			d.cancelled = true
			d.selected = 0
		}
		if d.last == raw.Code {
			d.repeats++
		} else {
			d.last = raw.Code
			d.repeats = 1
		}
		if v.mode == "table" {
			a.discoverControlCode(raw.Code)
			d.tested[raw.Code] = true
			d.proven[raw.Code] = true
			d.selected = raw.Code
			a.scrollToControl(raw.Code)
		}
	} else {
		held := d.held[raw.Code]
		delete(d.held, raw.Code)
		if !held {
			return true
		}
		if len(d.held) > 0 {
			// Releasing one report of the same physical direction is not
			// a partial release of a two-control gesture.
			for code := range d.held {
				if controlGestureCode(v.device, code) != controlGestureCode(v.device, raw.Code) {
					d.cancelled = true
					d.selected = 0
				}
			}
			a.all = true
			return true
		}
		valid := !d.cancelled && d.gesture == 1
		d.at = time.Time{}
		d.gesture = 0
		if valid && v.mode == "basic" {
			code := controlGestureCode(v.device, raw.Code)
			if !controls.ValidCode(code) {
				v.message = "That input is reserved. Choose another."
			} else if id := d.basic[code]; id != "" {
				v.message = "Already used for " + actionName(id) + ". Choose another."
			} else {
				d.basic[code] = basicOrder[d.step]
				d.step++
				v.message = ""
				if d.step == len(basicOrder) {
					a.completeBasicSetup()
				}
			}
		}
	}
	a.all = true
	return true
}
func (a *App) controlButtonRows() []string {
	rows := []string{"Change action"}
	if controlLabelEditable(a.mapping.draft.Inputs[a.mapping.check.editing]) {
		rows = append(rows, "Change OSD label")
	}
	return rows
}

func (a *App) actControlPopup(k platform.Key) {
	v := &a.mapping
	d := &v.check
	a.all = true
	if k == platform.KeyBack || k == platform.KeyMenu {
		if v.mode == "role" {
			a.exitControlCheck()
		} else if v.mode == "initial-role" {
			a.cancelBasicSetup()
		} else {
			a.setControlMode("table")
		}
		return
	}
	if v.mode == "labels" {
		a.actControlLabels(k)
		return
	}
	count := 0
	switch v.mode {
	case "button":
		count = len(a.controlButtonRows())
	case "actions":
		count = len(checkOrder) + 1
	case "swap-target":
		count = len(a.controlSwapCandidates())
	case "swap-confirm":
		count = 2
	case "role", "initial-role":
		count = 2
	}
	if k == platform.KeyUp {
		v.cursor = max(0, v.cursor-1)
	}
	if k == platform.KeyDown {
		v.cursor = min(count-1, v.cursor+1)
	}
	if k == platform.KeyHome {
		v.cursor = 0
	}
	if k == platform.KeyEnd {
		v.cursor = count - 1
	}
	if k != platform.KeyEnter {
		return
	}
	switch v.mode {
	case "button":
		switch a.controlButtonRows()[v.cursor] {
		case "Change action":
			a.setControlMode("actions")
			for i, id := range checkOrder {
				if v.draft.Inputs[d.editing] == id {
					v.cursor = i
				}
			}
		case "Change OSD label":
			a.setControlMode("labels")
			d.group = 0
			d.pick = -2
		}
	case "swap-target":
		codes := a.controlSwapCandidates()
		if v.cursor >= 0 && v.cursor < len(codes) {
			d.swap = codes[v.cursor]
			a.setControlMode("swap-confirm")
		}
	case "swap-confirm":
		if v.cursor == 1 {
			a.setControlMode("table")
		} else {
			a.commitControlSwap()
		}
	case "initial-role":
		if v.cursor == 1 {
			a.startBasicSetup()
		} else {
			candidate := a.normalizedControl(v.device, controls.Profile{})
			candidate.Name = v.device.Name
			candidate.Panel = false
			a.commitControl(candidate)
		}
	case "role":
		candidate := v.draft.Clone()
		candidate.Panel = v.cursor == 1
		if candidate.Panel && !candidate.BasicsComplete {
			a.startBasicSetup()
		} else {
			a.commitControl(candidate)
		}
	case "actions":
		if !controls.ValidCode(d.editing) {
			v.message = "This input is reserved."
			return
		}
		id := ""
		if v.cursor < len(checkOrder) {
			id = checkOrder[v.cursor]
		}
		old := v.draft.Inputs[d.editing]
		if id != "" && id == pairedControlAction(old) {
			a.startControlSwap()
			return
		}
		if old != id && requiredAction(old) {
			proven := false
			for c, other := range v.draft.Inputs {
				if c != d.editing && other == old && d.proven[c] {
					proven = true
				}
			}
			if !proven {
				v.message = "Test another " + actionName(old) + " button first."
				return
			}
		}
		candidate := v.draft.Clone()
		delete(candidate.Inputs, d.editing)
		if id != "" {
			candidate.Inputs[d.editing] = id
			if id == "menu" {
				candidate.LegacyMenuChord = false
			}
		}
		a.commitControl(candidate)
	}
}
func (a *App) actControlLabels(k platform.Key) {
	v := &a.mapping
	d := &v.check
	if k == platform.KeyPageUp || k == platform.KeyPageDown {
		dir := 1
		if k == platform.KeyPageUp {
			dir = -1
		}
		d.group = (d.group + dir + len(nameGroups)) % len(nameGroups)
		d.pick = -1
		return
	}
	if d.pick == -2 {
		if k == platform.KeyDown {
			d.pick = -1
		}
		if k == platform.KeyEnter {
			candidate := v.draft.Clone()
			delete(candidate.Names, d.editing)
			a.commitControl(candidate)
		}
		return
	}
	if d.pick == -1 {
		if k == platform.KeyLeft {
			d.group = (d.group + len(nameGroups) - 1) % len(nameGroups)
		}
		if k == platform.KeyRight {
			d.group = (d.group + 1) % len(nameGroups)
		}
		if k == platform.KeyUp {
			d.pick = -2
		}
		if k == platform.KeyDown || k == platform.KeyEnter {
			d.pick = 0
		}
		return
	}
	labels := nameGroups[d.group].labels
	switch k {
	case platform.KeyUp:
		d.pick -= 4
		if d.pick < 0 {
			d.pick = -1
		}
	case platform.KeyDown:
		d.pick = min(len(labels)-1, d.pick+4)
	case platform.KeyLeft:
		d.pick = max(0, d.pick-1)
	case platform.KeyRight:
		d.pick = min(len(labels)-1, d.pick+1)
	case platform.KeyEnter:
		if !controls.ValidCode(d.editing) {
			v.message = "This input is reserved."
			return
		}
		candidate := v.draft.Clone()
		candidate.Names[d.editing] = labels[d.pick]
		a.commitControl(candidate)
	}
}
func (a *App) tickControls(now time.Time) bool {
	v := &a.mapping
	d := &v.check
	v.now = now
	v.next = now.Add(frameDur)
	a.saver.lastInput = now
	if v.mode == "devices" {
		v.devices = a.controlDevices()
		if v.roleOptions {
			v.devices = a.controlKeyboards()
		}
		v.cursor = max(0, min(v.cursor, len(v.devices)-1))
		return a.all
	}
	if !a.connectedControl() {
		a.exitControlCheck()
		a.notice = "Controller disconnected. Previous setup kept."
		a.until = now.Add(3 * time.Second)
		return true
	}
	if v.mode == "invitation" || v.mode == "initial-role" {
		animating := a.ControlsAnimating()
		a.tickControlInvitation(now)
		a.all = a.all || animating
		return a.all
	}
	if !d.dismissAt.IsZero() {
		valid := false
		for _, p := range a.controlInput.devices {
			if p.Node == d.dismissEvent.Node && controlID(p) == d.dismissEvent.DeviceID {
				valid = !p.HeldKnown
				for _, c := range p.Held {
					valid = valid || c == d.dismissEvent.Code
				}
			}
		}
		if !valid {
			d.dismissAt = time.Time{}
			d.dismissToken = ""
		} else if now.Sub(d.dismissAt) >= controlHoldDuration {
			a.exitControlCheck()
			return true
		}
		a.all = true
	}
	missing := []uint16{}
	for _, p := range a.controlInput.devices {
		if p.Node != v.device.Node || !p.HeldKnown {
			continue
		}
		for c := range d.held {
			if c >= 0x300 {
				continue
			}
			found := false
			for _, held := range p.Held {
				found = found || c == held
			}
			if !found {
				missing = append(missing, c)
			}
		}
	}
	if len(missing) > 0 {
		if d.missingSince.IsZero() {
			d.missingSince = now
		}
		if now.Sub(d.missingSince) >= 150*time.Millisecond {
			a.clearControlTransition()
			d.selected = 0
			for token := range a.controlInput.held {
				for _, code := range missing {
					if strings.HasPrefix(token, controlID(v.device)+"/"+v.device.Node+"/") && strings.HasSuffix(token, fmt.Sprintf("/%d/0", code)) {
						delete(a.controlInput.held, token)
						delete(a.controlInput.keys, token)
						delete(d.ignored, token)
					}
				}
			}
		}
		return true
	}
	d.missingSince = time.Time{}
	a.advanceControlHold(now)
	if v.mode == "button" || v.mode == "actions" || v.mode == "labels" || v.mode == "role" || v.mode == "swap-target" || v.mode == "swap-confirm" {
		if key := v.nav.due(now, func(k platform.Key, _ int) time.Duration {
			if k >= platform.KeyUp && k <= platform.KeyRight {
				return repeatPage
			}
			return 0
		}); key != platform.KeyNone {
			a.actControlPopup(key)
		}
	}
	if !d.scrollAt.IsZero() {
		t := min(1.0, float64(now.Sub(d.scrollAt))/float64(repeatStep))
		d.scroll = d.from + (d.target-d.from)*t
		if t >= 1 {
			d.scrollAt = time.Time{}
		}
		a.all = true
	}
	if !d.at.IsZero() {
		a.all = true
	}
	return a.all
}
func (a *App) ControlsAnimating() bool {
	d := &a.mapping.check
	return a.screen == ScreenControls && (!d.introAt.IsZero() && !d.introCancelled || !d.dismissAt.IsZero() || !d.scrollAt.IsZero() || !d.at.IsZero() && !d.cancelled)
}
func (a *App) controlLatestText() string {
	d := &a.mapping.check
	if d.last == 0 {
		return "Latest input: --"
	}
	s := "Latest: " + a.controlCode(d.last) + " > " + actionName(a.mapping.draft.Inputs[d.last])
	if d.repeats > 1 {
		s += fmt.Sprintf(" x%d", d.repeats)
	}
	return s
}
