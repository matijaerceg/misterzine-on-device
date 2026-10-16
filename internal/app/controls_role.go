package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

func (a *App) controlKeyboards() []support.Pad {
	pads := []support.Pad{}
	for _, p := range a.controlDevices() {
		if p.Keyboard {
			pads = append(pads, p)
		}
	}
	return pads
}

// Keyboard use targets the attached keyboard, even when a pad opens Options.
// Multiple keyboards get an explicit chooser instead of changing an arbitrary one.
func (a *App) openControlRole() {
	if !a.featureAllowed(access.Controls) {
		return
	}
	a.controlInput.previous = a.screen
	a.mapping = controlsView{mode: "devices", roleOptions: true, devices: a.controlKeyboards(), next: a.controlNow().Add(time.Second)}
	a.screen = ScreenControls
	a.clearControlTransition()
	if len(a.mapping.devices) == 1 {
		a.editControlRole(a.mapping.devices[0])
	}
}

func (a *App) editControlRole(p support.Pad) {
	v := &a.mapping
	v.device = p
	old, known := a.cfg.ControlProfiles.Profiles[controlID(p)]
	if !known {
		old, known = a.cfg.ControlProfiles.Profiles[padID(p)]
	}
	if !known {
		old = controls.Profile{}
	}
	v.draft = a.normalizedControl(p, old)
	// Normal defaults alone don't prove that a keyboard encoder has six controls.
	v.draft.BasicsComplete = known && v.draft.BasicsComplete
	v.draft.Name = p.Name
	v.check = controllerCheck{entryHeld: map[uint16]bool{}, tested: map[uint16]bool{}, proven: map[uint16]bool{}, opened: a.controlNow()}
	for _, code := range p.Held {
		v.check.entryHeld[code] = true
	}
	a.observeControl(p.Node)
	a.setControlMode("role")
	if v.draft.Panel {
		v.cursor = 1
	}
}
