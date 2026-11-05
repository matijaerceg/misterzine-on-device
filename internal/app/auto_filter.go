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
	f.ArcadeOnly = !a.cfg.ShowNonArcade
	f.HideDeprecated = !a.cfg.ShowDeprecated
	f.HideMature = a.Mature() == MatureHide
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
	f.HideDeprecated = false
	f.HideMature = false
	f.ArcadeOnly = false
	if !a.cfg.ShowNonArcade {
		arcadeOff := f.BaseOff["Arcade"]
		f.BaseOff = nil
		if arcadeOff {
			f.BaseOff = map[string]bool{"Arcade": true}
		}
	}
	return f.Active()
}

// sourcesHelp explains Options -> Sources, including why nothing is
// hidden when the card has no downloader.ini.
func (a *App) sourcesHelp() string {
	if a.cfg.InstalledOnly && a.iniKnown && !a.iniFound {
		return "No downloader.ini was found on this card, so nothing is hidden. To hide missing games, use Filters > On the card > found on card."
	}
	return "Show sources listed in your Downloader INI files. To hide missing games, use Filters > On the card > found on card."
}

func (a *App) rotationFilterLabel() string {
	if a.rotationFilter() == "v" {
		return "Rotation: vertical only"
	}
	return "Rotation: horizontal only"
}
