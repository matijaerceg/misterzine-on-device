package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// MiSTercade V2's Generic HID firmware sends each direction twice: button
// usages 17..20 (Linux BTN_TRIGGER_HAPPY1..4) and the hat. Treat the pair
// as one control for setup gestures, retaining both raw events for the
// check table, release tracking and existing profiles. Do not infer aliases
// from timing: unrelated inputs and other controllers must remain separate.
func mistercadeDirection(p support.Pad, code uint16) (uint16, bool) {
	if p.Keyboard || p.Vendor != 0x16d0 || p.Product != 0x1358 {
		return code, false
	}
	var button, axis uint16
	switch code {
	case 704, 802:
		button, axis = 704, 17 // Up
	case 705, 803:
		button, axis = 705, 17 // Down
	case 706, 800:
		button, axis = 706, 16 // Left
	case 707, 801:
		button, axis = 707, 16 // Right
	default:
		return code, false
	}
	hasButton, hasAxis := false, false
	for _, c := range p.Keys {
		hasButton = hasButton || c == button
	}
	for _, c := range p.Axes {
		hasAxis = hasAxis || c == axis
	}
	if !hasButton || !hasAxis {
		return code, false
	}
	return button, true
}

func controlGestureCode(p support.Pad, code uint16) uint16 {
	code, _ = mistercadeDirection(p, code)
	return code
}

func (v *controlsView) heldControlCount() int {
	groups := map[uint16]bool{}
	for code := range v.check.held {
		groups[controlGestureCode(v.device, code)] = true
	}
	return len(groups)
}

// Prefer the assigned member when editing a pair: profiles made with an
// older firmware may use the hat, while new setup records the button.
func (v *controlsView) heldControlInput() uint16 {
	var selected uint16
	for code := range v.check.held {
		canonical := controlGestureCode(v.device, code)
		if v.draft.Inputs[canonical] != "" {
			return canonical
		}
		if v.draft.Inputs[code] != "" || selected == 0 {
			selected = code
		}
	}
	return selected
}

func (a *App) controlGestureToken(ev platform.Event) string {
	for _, p := range a.controlInput.devices {
		if controlID(p) == ev.DeviceID && p.Node == ev.Node {
			ev.Code = controlGestureCode(p, ev.Code)
			break
		}
	}
	return controlToken(ev)
}

func (a *App) invitationControlCount() int {
	groups := map[string]bool{}
	for _, ev := range a.mapping.check.introHeld {
		groups[a.controlGestureToken(ev)] = true
	}
	return len(groups)
}
