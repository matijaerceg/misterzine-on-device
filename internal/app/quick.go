package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Quick toggles: with Select held on the main list, Y cycles List layout,
// X switches List shots (the two Options rows people flip while browsing)
// and A stars or unstars the row (favorite.go). Select alone does nothing,
// the legend names the chords while it is down, and every other button
// waits until Select is released (actList), so nothing moves or launches
// by accident. Only a pad defined in MiSTer has a Select (its slot is read
// by input_mapping.go); keyboards use Options and Details.

// quickHeld reports whether Select is down, so Y and X are chords.
func (a *App) quickHeld() bool { return a.down[platform.KeySelect] }

func (a *App) cycleListLayout() bool {
	i := 0
	for n, s := range listLayouts {
		if s == a.ListLayout() {
			i = n
		}
	}
	a.cfg.ListLayout = listLayouts[(i+1)%len(listLayouts)]
	a.setRotation(a.rot) // the rows and the pane change shape
	a.settingsChanged()
	a.Notice("List layout: "+a.cfg.ListLayout, 2*time.Second)
	return true
}

func (a *App) cycleListShot() bool {
	if a.ListShot() == "title" {
		a.cfg.ListShot = "gameplay"
	} else {
		a.cfg.ListShot = "title"
	}
	a.settingsChanged()
	a.all = true
	a.Notice("List shots: "+a.cfg.ListShot, 2*time.Second)
	return true
}

func (a *App) settingsChanged() {
	if a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
}
