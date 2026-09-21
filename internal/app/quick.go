package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Quick toggles: with Select held on the main list, Y cycles Layout,
// X switches Art type (the two Options rows people flip while browsing),
// Left/Right turn the display (rotateBy) and A stars or unstars the row
// (favorite.go). Select alone does nothing,
// the legend names the chords while it is down, and every other button
// waits until Select is released (actList), so nothing moves or launches
// by accident. Only a pad defined in MiSTer has a Select (its slot is read
// by input_mapping.go); keyboards use Options and Details.

// quickHeld reports whether Select is down, so Y and X are chords.
func (a *App) quickHeld() bool { return a.down[platform.KeySelect] }

// listShowsArt reports whether the list layout has a picture for Art type
// to change. The text layout is rows alone, so the chord is left out of
// the legend and does nothing there; Options keeps the row, since the
// choice still applies to the layouts you switch back to.
func (a *App) listShowsArt() bool { return a.ListLayout() != "text" }

func (a *App) cycleListLayout() bool {
	i := 0
	for n, s := range listLayouts {
		if s == a.ListLayout() {
			i = n
		}
	}
	motion := a.captureLayoutMotion()
	a.cfg.ListLayout = listLayouts[(i+1)%len(listLayouts)]
	a.setRotation(a.rot) // the rows and the pane change shape
	a.startLayoutMotion(motion)
	a.settingsChanged()
	a.Notice("Layout: "+a.cfg.ListLayout, 2*time.Second)
	return true
}

func (a *App) cycleListShot() bool {
	if !a.listShowsArt() {
		return false // no picture on screen: the chord is not offered
	}
	if a.ListShot() == "title" {
		a.cfg.ListShot = "gameplay"
	} else {
		a.cfg.ListShot = "title"
	}
	a.settingsChanged()
	a.all = true
	a.Notice("Art type: "+a.cfg.ListShot, 2*time.Second)
	return true
}

// rotations is the order Select + Left/Right step through, the Options
// Rotation row's order: monitor turned clockwise, upright, counter-clockwise.
var rotations = []gfx.Rotation{gfx.RotRight, gfx.RotNone, gfx.RotLeft}

// rotateBy turns the display one step (Select + Right the next choice,
// Select + Left the previous) and saves it. A display following the INI
// stops doing so, as the Options Rotation row demands, and the notice says
// so, since the chord is easy to hit by accident and the INI setting
// would otherwise seem ignored from then on.
func (a *App) rotateBy(d int) bool {
	i := 1
	for n, r := range rotations {
		if r == a.rot {
			i = n
		}
	}
	rot := rotations[(i+d+len(rotations))%len(rotations)]
	followed := a.cfg.FollowRotation
	a.cfg.FollowRotation = false
	a.SetRotation(rot)
	if a.cfg.Action != nil {
		a.cfg.Action("rotation", map[gfx.Rotation]string{gfx.RotNone: "off", gfx.RotLeft: "left", gfx.RotRight: "right"}[rot])
	}
	a.settingsChanged()
	notice := "Rotation: " + rotationNames[rot]
	if followed {
		notice += ". Follow INI rotation is now off"
	}
	a.Notice(notice, 3*time.Second)
	return true
}

// rotationNames are the Options Rotation row's words for each choice.
var rotationNames = map[gfx.Rotation]string{gfx.RotRight: "monitor CW", gfx.RotNone: "horizontal", gfx.RotLeft: "monitor CCW"}

func (a *App) settingsChanged() {
	if a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
}
