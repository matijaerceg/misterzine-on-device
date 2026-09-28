package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// SetAppUpdate receives an already validated version from the host: a
// stable release, or in the beta build the newest beta.
func (a *App) SetAppUpdate(version string) {
	if a.appUpdate == version {
		return
	}
	a.appUpdate = version
	if a.screen == ScreenOptions {
		a.buildPanel()
	}
	a.all = true
}

func (a *App) OpenScan() {
	a.scanReady, a.scanError = false, ""
	a.scanWatch = a.watchCard()
	a.screen = ScreenScan
	a.rep = repeater{}
	a.all = true
	if a.cfg.Action != nil {
		a.cfg.Action("rescan", "")
	}
}

// FinishScan never steals focus if the user has left the result screen.
// message is a problem to show, or "". counts says the statuses arrived, so
// the result screen keeps its totals and shows the problem under them; a
// scan that produced nothing shows the problem alone.
func (a *App) FinishScan(message string, counts bool) {
	a.scanReady, a.scanError, a.scanCounts = true, message, counts
	a.scanWatch = a.scanWatch.settle(a, counts)
	a.all = true
}

func (a *App) cardCounts() map[data.Status]int {
	counts := map[data.Status]int{}
	for i := range a.ds.Rows {
		if !a.catalogueIncludes(&a.ds.Rows[i]) {
			continue
		}
		counts[a.status(i)]++
	}
	return counts
}

func (a *App) paintScan(c *gfx.Canvas) {
	a.paintStatus(c)
	box := a.lay.Body.Inset(4)
	c.Box(box, gen.Eva.Line)
	x, y := box.Min.X+4, box.Min.Y+4
	cols := a.sm.Cols(box.Dx() - 8)
	fit := (box.Max.Y-3-a.sm.H-y)/(a.sm.H+2) + 1 // the rows the box holds
	for _, line := range a.scanLines(cols, fit) {
		if line == "" {
			y += a.sm.H + 2
			continue
		}
		for _, part := range gfx.Wrap(line, cols, 8) {
			if y+a.sm.H > box.Max.Y-3 {
				break
			}
			c.Text(x, y, a.sm, part, gen.Eva.Fg)
			y += a.sm.H + 2
		}
	}
	a.paintHint(c, "B Back")
}

// scanLines is the scan screen's text for a box cols characters wide that
// holds fit rows; "" is a blank row.
func (a *App) scanLines(cols, fit int) []string {
	if !a.scanReady {
		return []string{"Checking card...", "", a.btn("B") + " returns while the scan continues."}
	}
	counts := a.cardCounts()
	lines := []string{"Card scan complete", "", "Up to date: " + itoa(counts[data.StatusCurrent]),
		"Older installed: " + itoa(counts[data.StatusOutdated]+counts[data.StatusLikelyOutdated]),
		"Build date unknown: " + itoa(counts[data.StatusFoundUndated]),
		"Not found on card: " + itoa(counts[data.StatusNotFound]),
		"Status unknown: " + itoa(counts[data.StatusUnknown]),
		"", "Compared with the enabled catalogue."}
	if n := a.ds.Facets.Src[data.SrcLocal]; n > 0 {
		lines = append(lines, "Local games not in the catalogue: "+itoa(n))
	}
	if a.scanError != "" && !a.scanCounts {
		lines = []string{a.scanError, "", "Could not finish every scan step.", "See the device log for details."}
	} else if a.scanError != "" {
		lines = append(lines, "", a.scanError+".", "See the device log for details.")
	}
	// what the scan changed (beta) goes under the heading, in the rows the
	// rest leaves: it never pushes a problem off the box
	if change := a.scanWatch.lines(cols, fit-textRows(lines, cols)); len(change) > 0 {
		lines = append(lines[:1], append(change, lines[1:]...)...)
	}
	return lines
}
