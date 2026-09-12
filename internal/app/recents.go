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
	entry := data.Recent{K: row.K, At: a.cfg.Now().UTC().Format(time.RFC3339)}
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
