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
	if a.cfg.InstalledOnly && len(a.hiddenSrc) > 0 {
		f.SrcHidden = a.hiddenSrc
	}
	return f
}

// filtersActive reports narrowing the list should show ("N of M"). The
// sources rule is not one: its sources are treated as if the feed never
// had them.
func (a *App) filtersActive() bool {
	f := a.effectiveFilters()
	f.SrcHidden = nil
	return f.Active()
}

// sourcesHelp explains Options -> Sources, including why nothing is
// hidden when the card has no downloader.ini.
func (a *App) sourcesHelp() string {
	if a.cfg.InstalledOnly && a.iniKnown && !a.iniFound {
		return "No downloader.ini was found on this card, so nothing is hidden. Installed only: games from the databases Downloader is set up to fetch."
	}
	return "Installed only: games from the databases in this card's downloader.ini, read at every card scan; other sources leave the list and Filters."
}

func (a *App) rotationFilterLabel() string {
	if a.rotationFilter() == "v" {
		return "Rotation: vertical only"
	}
	return "Rotation: horizontal only"
}
