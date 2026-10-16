//go:build linux

package main

import (
	"path/filepath"

	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
)

func (h *host) configureControls(cfg *app.Config, root string) {
	path := filepath.Join(root, "controls.json")
	profiles, err := controls.Load(path)
	cfg.ControlProfiles = profiles
	hooks := &app.ControlsHooks{Devices: h.input.ControlDevices, Observe: h.input.ObserveControls}
	if err != nil {
		h.lg.Printf("controls: could not read saved profiles: %v", err)
		hooks.Problem = err.Error()
	}
	hooks.Save = func(f controls.File) error {
		if err != nil {
			return err
		}
		if saveErr := controls.Save(path, f); saveErr != nil {
			h.lg.Printf("controls: save failed: %v", saveErr)
			return saveErr
		}
		return nil
	}
	hooks.Changed = func(f controls.File, enabled bool) {
		panels := map[string]bool{}
		if enabled {
			for id, p := range f.Profiles {
				if p.Panel {
					panels[id] = true
				}
			}
		}
		h.input.SetPanels(panels)
	}
	cfg.Controls = hooks
	hooks.Changed(profiles, access.Controls.Allowed(cfg.AccessMonth, cfg.ShowBetaFeatures))
}
