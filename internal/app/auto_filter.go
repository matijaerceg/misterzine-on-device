package app

import "github.com/matijaerceg/misterzine-on-device/internal/data"

func (a *App) iniFilter() string {
	if a.cfg.FilterRotation && (a.cfg.IniOrientation == "h" || a.cfg.IniOrientation == "v") {
		return a.cfg.IniOrientation
	}
	return ""
}

func (a *App) effectiveFilters() data.Filters {
	f := a.filters
	if orientation := a.iniFilter(); orientation != "" {
		f.MatchRotation = orientation
		f.RotOff = nil // keep saved manual rotation choices, temporarily superseded
	}
	return f
}

func (a *App) filtersActive() bool {
	f := a.effectiveFilters()
	return f.Active()
}

func (a *App) iniFilterLabel() string {
	if a.iniFilter() == "v" {
		return "INI: vertical only"
	}
	return "INI: horizontal only"
}
