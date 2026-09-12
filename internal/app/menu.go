package app

import "github.com/matijaerceg/misterzine-on-device/internal/platform"

// menuButton carries out the pad's MiSTer menu (OSD) button, which reaches
// the app instead of Main while the pad is held (input.go grabs a defined
// pad): Options from any screen, closing it when it is up, or leaving to
// the MiSTer menu, as Options -> Menu button says.
func (a *App) menuButton() bool {
	if a.MenuButton() == "leave" {
		if a.cfg.Quit != nil {
			a.cfg.Quit()
		}
		return false
	}
	if a.screen == ScreenOptions {
		return a.actPanel(platform.KeyBack) // closes it, remembering the row
	}
	if a.screen == ScreenUpdate && a.update.Active() {
		return false // a running update: hold B cancels, Menu waits
	}
	a.rep = repeater{}
	a.openPanel(ScreenOptions)
	return true
}
