package app

import (
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"github.com/matijaerceg/misterzine-on-device/internal/gfx"
	"image/color"
)

func (a *App) ShowBetaFeatures() bool { return a.cfg.ShowBetaFeatures }
func (a *App) featureAllowed(f access.Feature) bool {
	return f.Allowed(a.cfg.AccessMonth, a.cfg.ShowBetaFeatures)
}
func (a *App) featureVisible(f access.Feature) bool { return f.Visible(a.cfg.ShowBetaFeatures) }
func featureMarks(f access.Feature) string {
	s := ""
	if f.Fancy {
		s += " " + gfx.Star
	}
	if f.Beta {
		s += " " + gfx.Beta
	}
	return s
}

func (a *App) gatedOption(row panelEntry, f access.Feature) panelEntry {
	row.text += featureMarks(f)
	row.feature = true
	if !a.featureAllowed(f) {
		row.value = row.kind // Preserve the feature identity when sharing the code-entry action.
		row.kind = "enter-code"
		row.vals = []string{"Unlock"}
		row.idx = 0
		row.opensPage = true
		row.disabled = true
		row.requiredMonth = f.Since
		row.earlyAccess = f.Beta && !f.Fancy
		if row.earlyAccess {
			row.vals = []string{"Early access"}
			row.help += " Free for everyone after beta."
		} else {
			row.help += " " + a.btn("A") + ": Unlock."
		}
	}
	return row
}

func (a *App) accessChanged() {
	a.controlsChanged()
	selected := ""
	if a.panel.cursor < len(a.panel.entries) {
		selected = a.panel.entries[a.panel.cursor].text
	}
	a.membersStart()
	if a.cfg.AccessChanged != nil {
		a.cfg.AccessChanged(a.cfg.AccessMonth, a.cfg.ShowBetaFeatures)
	}
	a.buildPanel()
	for i, e := range a.panel.entries {
		if e.text == selected {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}

func (a *App) openCode() {
	a.lock = newBetaLock()
	a.lock.optional = true
	a.all = true
}

func (a *App) SetROMList(path string) { a.cfg.ROMList = path; a.all = true }

// paintFeatureText keeps the fitted label's placement and highlights only its
// Fancy and Beta glyphs. Use the theme's warning colour, including locked rows.
func paintFeatureText(c *gfx.Canvas, x, y int, font *gfx.Font, text string, col color.RGBA) {
	c.Text(x, y, font, text, col)
	for i := 0; i < len(text); i++ {
		if text[i] == gfx.Star[0] || text[i] == gfx.Beta[0] {
			c.Text(x+i*font.W, y, font, text[i:i+1], pal.Warn)
		}
	}
}
