package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// maxRecents is how many launches the history keeps.
const maxRecents = 100

// Recents is the launch history, newest first, for the host to persist.
func (a *App) Recents() []data.Recent { return a.cfg.RecentLaunches }

// recordLaunch puts the row at the top of the launch history, once, and
// trims the history to maxRecents.
func (a *App) recordLaunch(row *data.Row) {
	if row == nil {
		return
	}
	entry := data.Recent{K: row.K, At: a.cfg.Now().UTC().Format(time.RFC3339), View: a.mode.Name()}
	kept := make([]data.Recent, 0, len(a.cfg.RecentLaunches)+1)
	kept = append(kept, entry)
	for _, r := range a.cfg.RecentLaunches {
		if r.K != row.K && len(kept) < maxRecents {
			kept = append(kept, r)
		}
	}
	a.cfg.RecentLaunches = kept
	if a.cfg.RecentsChanged != nil {
		a.cfg.RecentsChanged()
	}
}

// ResumeAt lands on the game a launch record names, for the launcher's
// reopening after the game: in the current view when it lists the game,
// otherwise back in the view it was launched from (an app that does not
// remember its last view opens in the default one, which need not hold
// the row, and the Recents view only lists what was launched). A game
// the filters hide, or a launch view since turned off, leaves the list
// where it is; the caller hears whether the game was found.
func (a *App) ResumeAt(r data.Recent) bool {
	if a.keyInView(r.K) {
		a.MoveToKey(r.K)
		return true
	}
	if m, ok := data.ParseSort(r.View); ok && m != a.mode && a.viewOn(m) {
		a.SetSort(m)
		if a.keyInView(r.K) {
			a.MoveToKey(r.K)
			return true
		}
	}
	return false
}

// keyInView reports whether the current view lists the row.
func (a *App) keyInView(k string) bool {
	for _, idx := range a.view {
		if a.ds.Rows[idx].K == k {
			return true
		}
	}
	return false
}

// launchedAt is the row's latest launch as a local ISO date ("" when it was
// never launched or the record is unreadable).
func (a *App) launchedAt(key string) string {
	for _, r := range a.cfg.RecentLaunches {
		if r.K != key {
			continue
		}
		t, err := time.Parse(time.RFC3339, r.At)
		if err != nil {
			return ""
		}
		return t.In(a.cfg.Now().Location()).Format("2006-01-02")
	}
	return ""
}
