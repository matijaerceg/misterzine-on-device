package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// menuHold is how long the pad's Menu button must stay down to quit
// MisterZine while Options -> Menu button is "Options", so the way out is
// always there without changing the setting. A tap opens or closes Options
// when the button is released. menuHint is when the hold engages: past any
// tap (well under 200 ms) but early enough to read the hint before the
// quit. From then on the release does nothing, so the screen never changes
// under a hold.
const (
	menuHold = 2 * time.Second
	menuHint = 300 * time.Millisecond
)

const menuHoldNotice = "keep holding to quit MisterZine"

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
	a.openOptions()
	return true
}

// menuPress starts timing a Menu press in Options mode; menuRelease ends
// it: a tap (released before menuHint) opens or closes Options, a release
// after the hold engaged only takes the hint away. True when the screen
// changed.
func (a *App) menuPress(at time.Time) {
	if !a.menuAt.IsZero() {
		return
	}
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	a.menuAt, a.menuHinted = at, false
}

func (a *App) menuRelease(at time.Time) bool {
	if a.menuAt.IsZero() {
		return false // the hold already quit, or the press was never timed
	}
	if at.IsZero() {
		at = a.cfg.TimerNow()
	}
	engaged := a.menuHinted || at.Sub(a.menuAt) >= menuHint
	a.menuAt, a.menuHinted = time.Time{}, false
	if !engaged {
		return a.menuButton()
	}
	if a.notice == menuHoldNotice {
		a.notice = ""
		a.all = true
	}
	return true
}

// tickMenu shows the hint once a Menu press outlasts a tap and leaves the
// app once it reaches menuHold; true when the screen changed.
func (a *App) tickMenu(now time.Time) bool {
	if a.menuAt.IsZero() {
		return false
	}
	held := now.Sub(a.menuAt)
	if held >= menuHold {
		a.menuAt, a.menuHinted = time.Time{}, false
		if a.notice == menuHoldNotice {
			a.notice = ""
		}
		if a.cfg.Quit != nil {
			a.cfg.Quit()
		}
		a.all = true
		return true
	}
	if held >= menuHint && !a.menuHinted {
		a.menuHinted = true
		a.notice, a.until = menuHoldNotice, a.menuAt.Add(menuHold) // gone with the leave
		a.all = true
		return true
	}
	return false
}

// nextMenuTick is when the held Menu button next needs a look: the hint,
// then the leave; zero while it is up.
func (a *App) nextMenuTick() time.Time {
	if a.menuAt.IsZero() {
		return time.Time{}
	}
	if !a.menuHinted {
		return a.menuAt.Add(menuHint)
	}
	return a.menuAt.Add(menuHold)
}

// openOptions opens Options over the current screen and remembers where
// to go back to: the list, Details, the artwork or Filters (whose browsing
// state is kept aside, since the two panels share it). Screens reached
// from Options itself (Troubleshooting, calibration, a scan or update
// result) keep the earlier origin, so the trip Details -> Options ->
// Troubleshooting -> Options -> back still ends on Details.
func (a *App) openOptions() {
	switch a.screen {
	case ScreenList, ScreenDetails, ScreenShot:
		a.optionsFrom, a.optionsKey, a.filterHeld = a.screen, a.CursorKey(), nil
	case ScreenFilter:
		held := a.panel
		held.yearOpen = copyBools(a.panel.yearOpen)
		held.sectionClosed = copyBools(a.panel.sectionClosed)
		a.optionsFrom, a.optionsKey, a.filterHeld = ScreenFilter, "", &held
	}
	a.openPanel(ScreenOptions)
}

// closeOptions leaves Options for the screen it was opened from. Details
// and the artwork need the same row under the cursor; a row that left the
// view while Options changed the filters lands on the list instead.
func (a *App) closeOptions() {
	from, key, held := a.optionsFrom, a.optionsKey, a.filterHeld
	a.optionsFrom, a.optionsKey, a.filterHeld = ScreenList, "", nil
	switch from {
	case ScreenDetails, ScreenShot:
		if key != "" && a.CursorKey() == key {
			a.screen = from
			a.all = true
			return
		}
	case ScreenFilter:
		if held == nil {
			a.openPanel(ScreenFilter)
			return
		}
		row := a.panel.optionsRow
		a.panel = *held
		a.panel.optionsRow = row
		a.screen = ScreenFilter
		a.buildPanel() // Options may have changed the sources or the rotation filter
		a.all = true
		return
	}
	a.screen = ScreenList
	a.all = true
}

func copyBools(m map[string]bool) map[string]bool {
	if m == nil {
		return nil
	}
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
