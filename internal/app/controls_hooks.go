package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/controls"
	"github.com/matijaerceg/misterzine-on-device/internal/support"
)

// ControlsHooks separates device discovery and durable writes from the UI.
// Save must finish successfully before a profile becomes active.
type ControlsHooks struct {
	Devices func() []support.Pad
	Save    func(controls.File) error
	Changed func(controls.File, bool)
	Problem string // a failed load is read-only, never overwritten by a save
}
