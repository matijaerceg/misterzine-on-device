package app

import (
	"sort"
	"unicode"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The list can carry marker lines between its rows: the last-look divider
// and "no changes" line under the updated order, and a header before every
// group under the grouped orders (a maker, a letter, a release year). marks
// holds, in ascending order, the view positions each marker line precedes
// (len(view) = after the last row), so screen lines are view positions plus
// the markers before them.

// groupHeaders reports whether the order shows a header line before each
// of its groups: Maker, A-Z and Year. The date orders keep the month notice
// on a jump instead, and Favorites is short enough to read without them.
func (a *App) groupHeaders() bool {
	return a.mode == data.SortMaker || a.mode == data.SortAlphabetical || a.mode == data.SortYear
}

// groupLabel is what a header says for the group row i belongs to: the
// maker, the letter, or the release year, with the unknown group named.
func (a *App) groupLabel(i int) string {
	switch a.mode {
	case data.SortMaker:
		if m := a.ds.Der[i].Maker; m != "" {
			return m
		}
		return "Unknown maker"
	case data.SortAlphabetical:
		if c := a.ds.Der[i].TitleInitial(); c != '#' {
			return string(unicode.ToUpper(c))
		}
		return "0-9 and symbols"
	case data.SortYear:
		if y := data.ReleaseYear(a.ds.Rows[i].Year); y != "" {
			return y
		}
		return "Year unknown"
	}
	return ""
}

// rebuildMarks derives the marker lines from the view, the sort order and
// the last-look state (split/topMark, set by rebuild).
func (a *App) rebuildMarks() {
	a.marks = a.marks[:0]
	switch {
	case a.groupHeaders():
		prev := ""
		for pos := range a.view {
			if k := a.jumpGroupKey(pos); pos == 0 || k != prev {
				a.marks = append(a.marks, pos)
				prev = k
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

// pinnedHeader is the group name to keep on the top line while its rows
// run on above the screen: the group's own header has scrolled off, so its
// name covers the first line until the next header reaches it. Empty when
// the top line is a header itself, the cursor sits on the top line, or the
// order has no group headers.
func (a *App) pinnedHeader() string {
	if !a.groupHeaders() || a.top == 0 || len(a.view) == 0 {
		return ""
	}
	pos := 0
	for pos < len(a.view) && a.screenLine(pos) < a.top {
		pos++
	}
	if pos >= len(a.view) || a.screenLine(pos)-1 >= a.top || a.screenLine(a.cursor) == a.top {
		return ""
	}
	return a.groupLabel(a.view[pos])
}

// markText is what marker k says.
func (a *App) markText(k int) string {
	if a.groupHeaders() {
		if pos := a.marks[k]; pos < len(a.view) {
			return a.groupLabel(a.view[pos])
		}
		return ""
	}
	if a.topMark {
		return a.noChangesLabel()
	}
	return a.seen.Label(a.cfg.Now(), a.cfg.ClockTrusted)
}
