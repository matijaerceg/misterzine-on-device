package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// cardWatch is a card scan whose outcome the player is told: Options ->
// Rescan card, and the scan an Update All run ends with. It holds the statuses the scan started from, by row key, and
// once the scan has finished, what it changed.
type cardWatch struct {
	before map[string]data.Status // every status known when the scan started
	done   bool
	change cardChange
}

// cardChange counts the releases of the enabled catalogue whose status a
// scan changed, among those whose status was known before it.
type cardChange struct {
	updated   int // on the card before, older or of unknown date; up to date now
	installed int // not found on the card before; up to date now
	lost      int // up to date before; older, of unknown date or not found now
}

// watchCard notes the statuses a scan starts from. Nil while no status is
// known at all: a first scan has nothing to compare with, and every release
// it finds would otherwise read as just installed.
func (a *App) watchCard() *cardWatch {
	before := map[string]data.Status{}
	for i := range a.ds.Rows {
		if st := a.status(i); st != data.StatusUnknown {
			before[a.ds.Rows[i].K] = st
		}
	}
	if len(before) == 0 {
		return nil
	}
	return &cardWatch{before: before}
}

// settle ends the watch when its scan has finished: counts says the scan
// delivered statuses, and without them there is nothing to report. It
// returns the watch to keep, nil for none; a settled watch stays as it is.
func (w *cardWatch) settle(a *App, counts bool) *cardWatch {
	if w == nil || w.done {
		return w
	}
	if !counts {
		return nil
	}
	w.done, w.change = true, a.cardChange(w.before)
	w.before = nil
	return w
}

// cardChange compares the statuses on show with before, over the rows the
// scan screen counts. A release new to the list since, one whose status was
// unknown then or is unknown now, and one that has left the list count
// nowhere: none of them says what the card did.
func (a *App) cardChange(before map[string]data.Status) cardChange {
	var c cardChange
	for i := range a.ds.Rows {
		r := &a.ds.Rows[i]
		if !a.catalogueIncludes(r) {
			continue
		}
		was, ok := before[r.K]
		now := a.status(i)
		if !ok || now == data.StatusUnknown || now == was {
			continue
		}
		switch {
		case now == data.StatusCurrent && was == data.StatusNotFound:
			c.installed++
		case now == data.StatusCurrent:
			c.updated++
		case was == data.StatusCurrent:
			c.lost++
		}
	}
	return c
}

// lines say what the scan changed, in the scan screen's words, in at most
// room rows of cols characters: the number first, then how it breaks down,
// then what stopped being up to date, as far as room allows. "Checking
// card..." until the scan has finished; none without a watch.
func (w *cardWatch) lines(cols, room int) []string {
	if w == nil || room < 1 {
		return nil
	}
	if !w.done {
		return []string{"Checking card..."}
	}
	c := w.change
	head := "Newly up to date: " + itoa(c.updated+c.installed)
	parts := ""
	switch {
	case c.updated > 0 && c.installed > 0:
		parts = "(" + itoa(c.updated) + " updated, " + itoa(c.installed) + " installed)"
	case c.updated > 0:
		parts = "(updated)"
	case c.installed > 0:
		parts = "(installed)"
	}
	lines := []string{head}
	if parts != "" && len(head)+1+len(parts) <= cols {
		lines[0] += " " + parts
	} else if parts != "" {
		// a narrow column breaks before the bracket, not inside it
		lines = append(lines, parts)
	}
	if c.lost > 0 {
		lines = append(lines, "No longer up to date: "+itoa(c.lost))
	}
	for len(lines) > 0 && textRows(lines, cols) > room {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// textRows is how many rows lines take in a column cols characters wide,
// painted as the scan screen paints them: a blank line takes a row too.
func textRows(lines []string, cols int) int {
	n := 0
	for _, line := range lines {
		if line == "" {
			n++
			continue
		}
		n += len(gfx.Wrap(line, cols, 8))
	}
	return n
}

// RescanAfterUpdate is told when the host has asked for the card scan a
// finished update run ends with: the run's screen then reports what that
// scan changed. A MisterZine-only run and the switch to the free version
// leave the games alone and report nothing.
func (a *App) RescanAfterUpdate() {
	if a.update.Mode == updater.ModeApp || a.update.Mode == updater.ModeFree {
		return
	}
	a.updateView.card = a.watchCard()
	if a.updateView.card != nil && a.screen == ScreenUpdate {
		a.all = true
	}
}

// CardScanned is told when a scan of the rows on show has finished, whatever
// asked for it; counts says it delivered statuses. The first one after an
// update run settles that run's report.
func (a *App) CardScanned(counts bool) {
	if w := a.updateView.card; w != nil && !w.done {
		a.updateView.card = w.settle(a, counts)
		if a.screen == ScreenUpdate {
			a.all = true
		}
	}
}
