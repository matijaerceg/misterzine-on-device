package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// romReportHelp explains Options -> Data -> ROM report, naming the file the
// background check writes when the host gives one. Worded so the path keeps
// a line of its own in tate at the widest inset (four lines of 30).
func (a *App) romReportHelp() string {
	s := "Shows every game with a ROM problem, clearing other filters."
	if a.cfg.ROMList != "" {
		s += " Zip files to fetch: " + a.cfg.ROMList
	}
	return s
}

// openROMReport is the ROM report on the device: the list narrowed to every
// game a ROM problem touches, in any version, with the other filters, the
// search and a view showing only part of the catalogue put aside, so the
// list agrees with the file the background check writes. While the check
// has not finished, or found nothing the list can show, a notice says so
// and nothing changes. The notices fit tate's status bar at the widest
// inset (31 columns).
func (a *App) openROMReport() {
	if a.cfg.ROMKnown == nil {
		a.Notice("ROM check unavailable", 4*time.Second)
		return
	}
	if a.cfg.ROMProgress != nil {
		if done, total := a.cfg.ROMProgress(); total == 0 {
			a.Notice("ROM check runs after the scan", 4*time.Second)
			return
		} else if done < total {
			a.Notice("ROM check: "+itoa(done)+" of "+itoa(total)+" files"+gfx.Ellipsis, 4*time.Second)
			return
		}
	}
	mode := a.mode
	if viewNarrows(mode) {
		if m, ok := a.fullView(); ok {
			mode = m
		}
	}
	key := a.CursorKey()
	oldMode, oldFilters, oldQuery := a.mode, a.filters, a.query
	a.mode, a.filters, a.query = mode, data.Filters{ROM: data.ROMAny}, ""
	a.cursor, a.top = 0, 0
	a.rebuild()
	if a.shown == 0 {
		a.mode, a.filters, a.query = oldMode, oldFilters, oldQuery
		a.rebuild()
		if key != "" {
			a.moveToKey(key)
		}
		if a.romProblems() {
			a.Notice("Problems only in hidden games", 4*time.Second)
		} else {
			a.Notice("No ROM problems found", 4*time.Second)
		}
		return
	}
	if a.cfg.FiltersChanged != nil {
		a.cfg.FiltersChanged()
	}
	if mode != oldMode && a.cfg.SettingsChanged != nil {
		a.cfg.SettingsChanged()
	}
	a.optionsFrom, a.optionsKey, a.filterHeld = ScreenList, "", nil
	a.closeOptions()
	if a.shown == 1 {
		a.Notice("1 game with a ROM problem", 5*time.Second)
	} else {
		a.Notice(itoa(a.shown)+" games with ROM problems", 5*time.Second)
	}
}

// romProblems reports whether the ROM check found a problem in any row,
// listed or not.
func (a *App) romProblems() bool {
	for i := range a.ds.Rows {
		if s := a.romState(i); s == data.ROMLaunchIssue || s == data.ROMOtherIssue {
			return true
		}
	}
	return false
}
