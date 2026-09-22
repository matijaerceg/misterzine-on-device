package app

import (
	"sort"
	"unicode"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// The list can carry marker lines between its rows: the since-visit status
// row on top under every order, and a header before every group under the
// grouped orders (a maker, a letter, a release year). marks holds, in
// ascending order, the view positions each marker line precedes (len(view) =
// after the last row; the status row and the first header share position 0),
// so screen lines are view positions plus the markers before them.

// groupHeaders reports whether the order shows a header line before each
// of its groups: Maker, A-Z and Year. The date orders keep the month notice
// on a jump instead, and Favorites is short enough to read without them.
func (a *App) groupHeaders() bool {
	return a.mode == data.SortMaker || a.mode == data.SortAlphabetical || a.mode == data.SortYear
}

// groupLabel is what a header says for the group row i belongs to: the
// maker, the letter, or the release year, with the unknown group named.
func (a *App) groupLabel(i int) string {
	if a.mode.Anchored() {
		i = a.ds.SortRow(i) // a standin belongs to its catalogue row's group
	}
	switch a.mode {
	case data.SortMaker:
		if m := a.ds.Der[i].Maker; m != "" {
			return m
		}
		return "Unknown manufacturer"
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
// the last-look state (marker, set by rebuild). The status row is always the
// first line; it mirrors the site, where the in-list divider of earlier
// versions is gone and each changed row carries its own mark instead.
func (a *App) rebuildMarks() {
	a.marks = a.marks[:0]
	if a.marker {
		a.marks = append(a.marks, 0)
	}
	if a.groupHeaders() {
		prev := ""
		for pos := range a.view {
			if k := a.jumpGroupKey(pos); pos == 0 || k != prev {
				a.marks = append(a.marks, pos)
				prev = k
			}
		}
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

// markText is what marker k says: the since-visit status on the first
// line, else the group header. The status sentence gives way to shorter
// forms where the line would lose its tail to the ellipsis.
func (a *App) markText(k int) string {
	if a.marker && k == 0 {
		cols := a.sm.Cols(a.lay.lineRect(0).Dx()) - 2
		forms := a.seen.Status(a.cfg.Now(), a.cfg.ClockTrusted, a.sinceAdded, a.sinceUpdated)
		for _, f := range forms {
			if len(f) <= cols {
				return f
			}
		}
		return forms[len(forms)-1]
	}
	if a.groupHeaders() {
		if pos := a.marks[k]; pos < len(a.view) {
			return a.groupLabel(a.view[pos])
		}
	}
	return ""
}
