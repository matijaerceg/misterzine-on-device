package app

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

var controlSlots = map[string]string{"start": "Start", "enter": "A", "back": "B", "tab": "X", "space": "Y", "pageup": "L", "pagedown": "R", "select": "Select", "up": "Up", "down": "Down", "left": "Left", "right": "Right"}
var controlTokens = map[string]string{"Start": "start", "A": "enter", "B": "back", "X": "tab", "Y": "space", "L": "pageup", "R": "pagedown", "Select": "select", "Menu": "menu"}

func controlToken(ev platform.Event) string {
	return fmt.Sprintf("%s/%s/%s/%d/%d", ev.DeviceID, ev.Node, ev.Source, ev.Code, func() platform.Key {
		if ev.Code == 0 {
			return ev.Key
		}
		return 0
	}())
}

func (a *App) controlDevices() []support.Pad {
	if h := a.cfg.Controls; h != nil && h.Devices != nil {
		a.controlInput.devices = h.Devices()
		a.controlInput.legendGen++
		return a.controlInput.devices
	}
	return nil
}

func (a *App) controlBase(p support.Pad, profile controls.Profile) map[string]uint16 {
	if p.Keyboard && profile.Panel {
		return controls.PanelBase()
	}
	if p.Keyboard {
		return map[string]uint16{"up": 103, "down": 108, "left": 105, "right": 106, "enter": 28, "back": 1, "tab": 15, "space": 57, "pageup": 104, "pagedown": 109}
	}
	m := map[string]uint16{}
	for action, slot := range controlSlots {
		m[action] = p.Slots[slot]
	}
	if a.swapped(p) {
		m["enter"], m["back"] = m["back"], m["enter"]
	}
	if n, err := strconv.ParseUint(p.Menu, 10, 16); err == nil {
		m["menu"] = uint16(n)
	}
	return m
}

func (a *App) controlProfile(id string) controls.Profile {
	if a.featureAllowed(access.Controls) {
		return a.cfg.ControlProfiles.Profiles[id]
	}
	return controls.Profile{}
}

// mapControlEvent keeps press/release pairing in one UI-thread layer. The
// evdev readers, capture, labels and tester all retain the original raw code.
func (a *App) mapControlEvent(ev platform.Event) platform.Event {
	if !a.featureAllowed(access.Controls) || ev.DeviceID == "" || (!ev.Direct && !ev.Keyboard) {
		return ev
	}
	p, ok := a.cfg.ControlProfiles.Profiles[ev.DeviceID]
	if ok && p.MappingVersion == 2 {
		if p.LegacyMenuChord && ev.Key == platform.KeyMenu {
			return ev
		}
		if id := p.Inputs[ev.Code]; id != "" {
			// A typing keyboard's profile that restates a key's keyboard
			// meaning (Space as Y, the default) keeps the key's text, so
			// Space still types once a search has begun (Handle). A
			// Logitech receiver's keyboard and Consumer Control nodes share
			// one identity, so a profile set up on either covers both.
			typed := ev.Keyboard && !p.Panel && controls.Key(id) == ev.Key
			ev.Key = controls.Key(id)
			if !typed {
				ev.Text = 0
			}
			ev.ObservationOnly = false
			return ev
		}
		if !ev.Keyboard || p.Panel {
			ev.Key = platform.KeyOther
			ev.Text = 0
		} else if ev.Code != 0 && ev.Key != platform.KeyBackspace && ev.Key != platform.KeyScreenshot {
			ev.Key = platform.KeyOther
		}
		return ev
	}

	if ev.Keyboard && (!ok || !p.Panel) {
		return ev
	}
	if ev.Pressed {
		a.controlInput.current = ev.DeviceID
		a.controlInput.node = ev.Node
		found := false
		for _, d := range a.controlInput.devices {
			found = found || (controlID(d) == ev.DeviceID && d.Node == ev.Node)
		}
		if !found {
			a.controlDevices()
		}
	}
	if !ok {
		return ev
	}
	// MiSTer's two-button Menu chord stays a recovery route even when one
	// of its individual buttons has a different MisterZine action. A complete
	// remap explicitly replaces Menu too, including when it is skipped.
	if _, explicitMenu := p.Bindings["menu"]; ev.Key == platform.KeyMenu && !explicitMenu {
		for _, d := range a.controlInput.devices {
			if d.Node == ev.Node && controlID(d) == ev.DeviceID && strings.Contains(d.Menu, "+") {
				return ev
			}
		}
	}
	if ev.Keyboard {
		ev.Text = 0
		ev.Key = platform.KeyOther
	}
	base := map[string]uint16{}
	if ev.Keyboard {
		base = controls.PanelBase()
	}
	m := controls.Effective(base, p)
	for _, id := range checkOrder {
		if m[id] != 0 && m[id] == ev.Code {
			ev.Key = controls.Key(id)
			ev.Text = 0
			return ev
		}
	}
	// An overridden action loses its old physical button. An unassigned raw
	// button remains available for future capture, never a second launch key.
	if _, changed := p.Bindings[ev.Key.String()]; changed {
		ev.Key = platform.KeyOther
		ev.Text = 0
	}
	return ev
}

func (a *App) controlsChanged() {
	a.controlInput.legendGen++
	if h := a.cfg.Controls; h != nil && h.Changed != nil {
		h.Changed(a.cfg.ControlProfiles.Clone(), a.featureAllowed(access.Controls))
	}
}

func (a *App) clearControlHolds() {
	a.rep = repeater{}
	a.down = map[platform.Key]bool{}
	a.released = map[platform.Key]time.Time{}
	a.menuAt = time.Time{}
	a.menuHinted = false
	a.menuBar = 0
}

func (a *App) controlCode(code uint16) string {
	if code == 0 {
		return "-"
	}
	if code < 256 {
		for _, row := range []struct {
			start   uint16
			letters string
		}{{16, "QWERTYUIOP"}, {30, "ASDFGHJKL"}, {44, "ZXCVBNM"}, {2, "1234567890"}} {
			if code >= row.start && int(code-row.start) < len(row.letters) {
				return string(row.letters[code-row.start])
			}
		}
		names := map[uint16]string{1: "Esc", 2: "1", 6: "5", 15: "Tab", 28: "Enter", 29: "L Ctrl", 42: "L Shift", 44: "Z", 45: "X", 56: "L Alt", 57: "Space", 104: "PgUp", 109: "PgDn", 97: "R Ctrl", 103: "Up", 108: "Down", 105: "Left", 106: "Right"}
		if s := names[code]; s != "" {
			return s
		}
		return "Key " + support.CodeText(code)
	}
	if code >= 768 {
		return support.CodeText(code)
	}
	return "Btn " + support.CodeText(code)
}

// Button names follow physical codes. Legacy action names remain readable until
// the profile is edited or remapped, when they migrate to physical names.
func (a *App) controlActionLabel(action string, p support.Pad, profile controls.Profile) string {
	code := controls.Effective(a.controlBase(p, profile), profile)[action]
	if profile.MappingVersion == 2 {
		code = 0
		for c, id := range profile.Inputs {
			if id == action && (code == 0 || c < code) {
				code = c
			}
		}
	}
	_, explicitMenu := profile.Bindings["menu"]
	if code == 0 && !(action == "menu" && ((profile.MappingVersion == 2 && profile.LegacyMenuChord) || (profile.MappingVersion == 0 && !explicitMenu && strings.Contains(p.Menu, "+")))) {
		return "-"
	}
	if name := profile.Names[code]; name != "" {
		return displayButtonName(name)
	}
	if profile.MappingVersion == 2 {
		if name := controlDirectionName(action); name != "" {
			return name
		}
		if action == "menu" && code == 0 && profile.LegacyMenuChord {
			return "Menu"
		}
		if p.Keyboard {
			return a.controlCode(code)
		}
		if slot := p.Slot(code); slot != "" {
			return slot
		}
		return a.controlCode(code)
	}
	if profile.Style == "custom" {
		if s := profile.Labels[action]; s != "" {
			return s
		}
	}
	if p.Keyboard && !profile.Panel {
		return a.controlCode(code)
	}
	slot := controlSlots[action]
	if action == "menu" {
		slot = "Menu"
	}
	if code != 0 {
		physicalSlot := p.Slot(code)
		if p.Keyboard {
			for id, c := range controls.PanelBase() {
				if c == code {
					physicalSlot = controlSlots[id]
					if id == "menu" {
						physicalSlot = "Menu"
					}
				}
			}
		}
		if s := physicalSlot; s != "" {
			slot = s
		} else if len(profile.Bindings) > 0 {
			slot = "?"
		}
	}
	set := profile.Style
	if set == "" || set == "default" || set == "custom" {
		set = profile.NameBase
		if set == "" || set == "default" {
			set = a.ButtonLabels()
		}
	}
	if set == "neogeo" {
		return applyLabels(slot, map[string]string{"A": "A", "B": "B", "X": "C", "Y": "D"})
	}
	return applyLabels(slot, buttonLabels[set])
}

func (a *App) controlLegend(s string) (string, bool) {
	if !a.featureAllowed(access.Controls) || a.controlInput.current == "" {
		return s, false
	}
	profile, ok := a.cfg.ControlProfiles.Profiles[a.controlInput.current]
	if !ok {
		return s, false
	}
	var p support.Pad
	found := false
	if a.controlInput.devices == nil {
		a.controlDevices()
	}
	for _, device := range a.controlInput.devices {
		if controlID(device) == a.controlInput.current && (device.Direct || device.Keyboard) {
			p = device
			found = true
			if device.Node == a.controlInput.node {
				break
			}
		}
	}
	if !found {
		return s, false
	}
	key := legendKey{current: a.controlInput.current, node: a.controlInput.node, set: a.ButtonLabels(), swapped: a.swapped(p), gen: a.controlInput.legendGen,
		version: profile.MappingVersion, style: profile.Style, base: profile.NameBase, panel: profile.Panel,
		inputs: len(profile.Inputs), bindings: len(profile.Bindings), names: len(profile.Names), custom: len(profile.Labels)}
	if a.controlInput.legendLabels == nil || key != a.controlInput.legendKey {
		labels := map[string]string{}
		for token, action := range controlTokens {
			labels[token] = a.controlActionLabel(action, p, profile)
		}
		labels["L/R"] = labels["L"] + "/" + labels["R"]
		a.controlInput.legendKey, a.controlInput.legendLabels = key, labels
	}
	return applyLabels(s, a.controlInput.legendLabels), true
}

// legendKey is what the legend's labels depend on. The hint line asks for
// them several times every frame while a list scrolls, and each label costs
// a few map allocations, so they are kept until something here changes:
// legendGen counts profile edits and device refreshes, and the profile
// fields catch an edit made in place.
type legendKey struct {
	current, node, set, style, base string
	swapped, panel                  bool
	gen                             uint64
	version                         int
	inputs, bindings, names, custom int
}

type controlsInput struct {
	inviteWait map[string]map[string]bool
	blocked    map[string]bool
	previous   Screen
	held       map[string]bool
	keys       map[string]platform.Key
	current    string
	node       string
	devices    []support.Pad

	legendKey    legendKey
	legendLabels map[string]string
	legendGen    uint64
}

func (a *App) controlEvent(ev platform.Event) (platform.Event, bool, bool) {
	ev = a.padEvent(ev)
	if ev.InstanceID != "" {
		ev.DeviceID = ev.InstanceID
	}
	raw := ev
	if ev.Pressed && ev.DeviceID != "" && ev.Node != "" {
		a.controlInput.current, a.controlInput.node = ev.DeviceID, ev.Node
	}
	token := controlToken(ev)
	if a.controlInput.held == nil {
		a.controlInput.held = map[string]bool{}
		a.controlInput.keys = map[string]platform.Key{}
	}
	fresh := ev.Pressed && !a.controlInput.held[token]
	if ev.Pressed {
		a.controlInput.held[token] = true
	} else {
		delete(a.controlInput.held, token)
		for id, waiting := range a.controlInput.inviteWait {
			delete(waiting, token)
			if len(waiting) == 0 {
				delete(a.controlInput.inviteWait, id)
			}
		}
	}
	ev = a.mapControlEvent(ev)
	if ev.Pressed {
		if _, exists := a.controlInput.keys[token]; !exists {
			a.controlInput.keys[token] = ev.Key
		}
	} else {
		if key, exists := a.controlInput.keys[token]; exists {
			ev.Key = key
			delete(a.controlInput.keys, token)
		}
	}
	if a.controlInput.blocked[token] {
		if !ev.Pressed {
			delete(a.controlInput.blocked, token)
		}
		if a.screen != ScreenControls {
			return ev, true, false
		}
	}
	if fresh && a.inviteControl(raw) {
		return ev, true, true
	}
	if a.screen != ScreenControls && ev.Key != platform.KeyOther && ev.Key != platform.KeyNone {
		for other, key := range a.controlInput.keys {
			if other != token && key == ev.Key {
				return ev, true, false
			}
		}
	}
	if a.screen == ScreenControls {
		return ev, true, a.handleControls(raw, ev, fresh)
	}
	// Encoders can use printable keys, whose kernel repeats survive screen
	// transitions. Navigation already has its own repeater; only a new
	// physical press may trigger an action. Typing keyboards keep repeats.
	if raw.Pressed && !fresh && raw.Keyboard && a.controlProfile(raw.DeviceID).Panel {
		return ev, true, false
	}
	if ev.ObservationOnly {
		return ev, true, false
	}
	return ev, false, false
}

func (a *App) nextControlsTick() time.Time { return a.mapping.next }

// controlsOptions puts Button mapping, and Keyboard use when a keyboard is
// attached, at the top of the Controls section.
func (a *App) controlsOptions(entries []panelEntry) []panelEntry {
	out := []panelEntry{}
	for _, e := range entries {
		if e.kind == "menu-button" {
			out = append(out, panelEntry{text: "Button mapping", kind: "controls", opensPage: true, help: "Set six basic controls, then check inputs, change actions and name buttons."})
			if len(a.controlKeyboards()) > 0 {
				out = append(out, panelEntry{text: "Keyboard use", kind: "controls-role", child: true, opensPage: true, help: "Choose whether unassigned keys can type into search."})
			}
		}
		out = append(out, e)
	}
	return out
}

func (a *App) controlNow() time.Time {
	now := a.cfg.TimerNow()
	if a.mapping.now.After(now) {
		return a.mapping.now
	}
	return now
}

func controlID(p support.Pad) string {
	if p.InstanceID != "" {
		return p.InstanceID
	}
	return padID(p)
}

// Templates are copied into an unfamiliar device's draft, never linked to it.
func (a *App) controlTemplate(p support.Pad) controls.Profile {
	id := controlID(p)
	if profile, ok := a.cfg.ControlProfiles.Profiles[id]; ok {
		return profile.Clone()
	}
	if profile, ok := a.cfg.ControlProfiles.Profiles[padID(p)]; ok {
		return profile.Clone()
	}
	ids := []string{}
	for key := range a.cfg.ControlProfiles.Profiles {
		if strings.HasPrefix(key, padID(p)+"_") {
			ids = append(ids, key)
		}
	}
	sort.Strings(ids)
	for _, key := range ids {
		profile := a.cfg.ControlProfiles.Profiles[key]
		compatible := true
		codes := []uint16{}
		for _, c := range profile.Bindings {
			codes = append(codes, c)
		}
		for c := range profile.Inputs {
			codes = append(codes, c)
		}
		for _, code := range codes {
			if code == 0 {
				continue
			}
			found := false
			if code >= 0x300 {
				for _, axis := range p.Axes {
					found = found || (code-0x300)/2 == axis
				}
			} else {
				for _, button := range p.Keys {
					found = found || button == code
				}
			}
			compatible = compatible && found
		}
		if compatible {
			q := profile.Clone()
			q.Reviewed = false
			return q
		}
	}
	return controls.Profile{}.Clone()
}
