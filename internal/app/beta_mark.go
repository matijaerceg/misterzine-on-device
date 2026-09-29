package app

import (
	"image"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
)

// betaMarkText is the mark at the right end of the status bar on the list
// and the lock screen in the Patreon beta: dark capitals on the amber of
// the beta sign after a beta core's title, so it reads as a mark rather
// than a label. The bar is the one strip every list layout (list, split,
// picture, text) and both orientations share, so the mark sits in the
// same place in all of them.
const betaMarkText = "BETA"

// paintBetaMark draws the mark in a beta build and returns where the
// status bar's own text has to end, as the right edge of its room (text
// right-aligned to that edge less 2 keeps a gap before the mark). The
// free build draws nothing and keeps the whole bar.
func (a *App) paintBetaMark(c *gfx.Canvas) (right int) {
	st := a.lay.Status
	if !beta.On() {
		return st.Max.X
	}
	w := a.sm.Width(betaMarkText) + 4
	r := image.Rect(st.Max.X-2-w, st.Min.Y+1, st.Max.X-2, st.Max.Y-1)
	c.Fill(r, pal.Warn)
	c.Text(r.Min.X+2, st.Min.Y+2, a.sm, betaMarkText, pal.AccentOn)
	return r.Min.X - 2
}
