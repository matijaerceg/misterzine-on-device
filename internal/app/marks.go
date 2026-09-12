package app

import (
	"sort"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The list can carry marker lines between its rows: the last-look divider
// and "no changes" line under the updated order, and a header before every
// maker under the Maker order. marks holds, in ascending order, the view
// positions each marker line precedes (len(view) = after the last row), so
// screen lines are view positions plus the markers before them.

// rebuildMarks derives the marker lines from the view, the sort order and
// the last-look state (split/topMark, set by rebuild).
func (a *App) rebuildMarks() {
	a.marks = a.marks[:0]
	switch {
	case a.mode == data.SortMaker:
		prev := ""
		for pos, i := range a.view {
			if m := a.ds.Der[i].Maker; pos == 0 || m != prev {
				a.marks = append(a.marks, pos)
				prev = m
			}
		}
	case a.topMark:
		a.marks = append(a.marks, 0)
	case a.split >= 0:
		a.marks = append(a.marks, a.split+1)
	}
}

// marksBefore counts the marker lines above view position pos, the one
// right before it included.
func (a *App) marksBefore(pos int) int {
	return sort.SearchInts(a.marks, pos+1)
}

// markAt reports whether a marker line sits right before view position pos.
func (a *App) markAt(pos int) bool {
	i := sort.SearchInts(a.marks, pos)
	return i < len(a.marks) && a.marks[i] == pos
}

// markLine is the screen line of marker k: the rows and markers before it.
func (a *App) markLine(k int) int {
	return a.marks[k] + k
}

// markText is what marker k says.
func (a *App) markText(k int) string {
	if a.mode == data.SortMaker {
		if pos := a.marks[k]; pos < len(a.view) {
			if m := a.ds.Der[a.view[pos]].Maker; m != "" {
				return m
			}
		}
		return "Unknown maker"
	}
	if a.topMark {
		return a.noChangesLabel()
	}
	return a.seen.Label(a.cfg.Now(), a.cfg.ClockTrusted)
}
