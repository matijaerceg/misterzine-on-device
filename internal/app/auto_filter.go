package app

import "github.com/matijaerceg/misterzine-on-device/internal/data"

// rotationFilter is the orientation the strict filter enforces: the
// interface's current rotation, whatever set it (INI or manual), or empty
// when the filter is off.
func (a *App) rotationFilter() string {
	if !a.cfg.FilterRotation {
		return ""
	}
	if a.rot.Rotated() {
		return "v"
	}
	return "h"
}

func (a *App) effectiveFilters() data.Filters {
	f := a.filters
	if orientation := a.rotationFilter(); orientation != "" {
		f.MatchRotation = orientation
		f.RotOff = nil // keep saved manual rotation choices, temporarily superseded
	}
	return f
}

func (a *App) filtersActive() bool {
	f := a.effectiveFilters()
	return f.Active()
}

func (a *App) rotationFilterLabel() string {
	if a.rotationFilter() == "v" {
		return "Rotation: vertical only"
	}
	return "Rotation: horizontal only"
}
