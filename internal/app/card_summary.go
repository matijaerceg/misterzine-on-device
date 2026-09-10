package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gen"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// SetAppUpdate receives an already validated stable version from the host.
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
	a.screen = ScreenScan
	a.rep = repeater{}
	a.all = true
	if a.cfg.Action != nil {
		a.cfg.Action("rescan", "")
	}
}

// FinishScan never steals focus if the user has left the result screen.
func (a *App) FinishScan(message string) {
	a.scanReady, a.scanError = true, message
	a.all = true
}

func (a *App) cardCounts() map[data.Status]int {
	counts := map[data.Status]int{}
	for i := range a.ds.Rows {
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
	lines := []string{"Checking card...", "", "B returns while the scan continues."}
	if a.scanReady {
		counts := a.cardCounts()
		lines = []string{"Card scan complete", "", "Up to date: " + itoa(counts[data.StatusCurrent]),
			"Older installed: " + itoa(counts[data.StatusOutdated]),
			"Build date unknown: " + itoa(counts[data.StatusFoundUndated]),
			"Not found on card: " + itoa(counts[data.StatusNotFound]),
			"Status unknown: " + itoa(counts[data.StatusUnknown]),
			"", "Compared with the catalogue."}
		if a.scanError != "" {
			lines = []string{a.scanError, "", "Could not finish every scan step.", "See the device log for details."}
		}
	}
	for _, line := range lines {
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
	a.paintHint(c, "B back")
}
