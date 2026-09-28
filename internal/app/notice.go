package app

// A notice (Notice) takes the bar it shows in, the status bar on most
// screens, until it expires or until a press changes what that bar would
// say without it: Select+Y and then Y names the new view at once, the
// first letter typed shows the search, a filter that changes the count
// shows the count. A press that leaves the bar as it was keeps the
// notice, so a jump's month stays up while the cursor moves under it, and
// a newer notice still replaces an older one.
//
// What changes without a press (a card scan's counts, a catalogue update,
// the App update mark, the connection text) waits for the notice as
// before, so a startup hint or the catalogue news is not cut short by
// work the app does on its own.

// noticeBar is what the bar under a notice says, in a form a later
// reading compares with: the screen, which names a panel's or
// Troubleshooting's title, the lock screen, and on the list and the scan
// screen the status line itself. Details (whose notice takes the legend)
// and the artwork keep theirs until it expires or the screen changes.
type noticeBar struct {
	screen Screen
	locked bool
	status statusBar
}

// noticeBarNow reads what a notice would cover now; false on the screens
// that show none (Update All, the lock screen's switch to the free
// version, calibration), where a notice waits for the screen that shows
// it.
func (a *App) noticeBarNow() (noticeBar, bool) {
	if a.lock != nil {
		return noticeBar{locked: true}, !a.lock.switching
	}
	switch a.screen {
	case ScreenUpdate, ScreenCalibrate:
		return noticeBar{}, false
	case ScreenList, ScreenScan:
		return noticeBar{screen: a.screen, status: a.statusBar()}, true
	}
	return noticeBar{screen: a.screen}, true
}

// noticeHeld reports a notice that the rule applies to. The Menu hold's
// hint belongs to the button, not the bar: it stays while the button is
// down, however the bar changes under it.
func (a *App) noticeHeld() bool {
	return a.notice != "" && a.notice != menuHoldNotice
}

// readNoticeBar takes what the notice covers as it reads now. It runs as
// each press or tick begins, so what changed since without a press counts
// as the notice's ground.
func (a *App) readNoticeBar() {
	if a.noticeHeld() {
		a.noticeOver, a.noticeRead = a.noticeBarNow()
	}
}

// settleNotice runs as each press or tick ends: it reads the ground of a
// notice set since, the press's own or the host's, and takes down a
// notice whose bar the press changed. True when it took one down.
func (a *App) settleNotice() bool {
	if !a.noticeHeld() {
		return false
	}
	bar, shown := a.noticeBarNow()
	if !a.noticeRead {
		a.noticeOver, a.noticeRead = bar, shown
		return false
	}
	if shown && bar == a.noticeOver {
		return false
	}
	a.notice = ""
	a.all = true
	return true
}
