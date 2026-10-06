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
		if row.lockedHelp != "" {
			row.help = row.lockedHelp
		}
		row.value = row.kind // Preserve the feature identity when sharing the code-entry action.
		row.kind = "enter-code"
		row.vals = []string{f.Since.Short()}
		row.idx = 0
		row.opensPage = true
		row.disabled = true
		row.requiredMonth = f.Since
		row.earlyAccess = f.Beta && !f.Fancy
		if row.earlyAccess {
			row.help += " Free for everyone after beta."
		} else {
			row.help += " Your code unlocks it forever."
		}
		current := a.cfg.AccessMonth.Short()
		if !a.cfg.AccessMonth.Valid() {
			current = "none"
		}
		row.help = "Needs " + f.Since.Short() + "+. Your access: " + current + ". " + row.help
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

// Your access is available even with beta hidden and with no code entered.
func (a *App) accessEntries() []panelEntry {
	E := []panelEntry{
		{text: "Your access: " + a.cfg.AccessMonth.Short(), kind: "access-info", help: "The month is coverage, not expiry. You do not need to stay subscribed."},
		{text: "Enter MisterZine code", kind: "enter-code", opensPage: true, help: "Newer codes include earlier features. Older codes never reduce your access."},
	}
	for _, entry := range access.Catalog() {
		f := entry.Feature
		state := "Needs " + f.Since.Short() + "+"
		if f.Covered(a.cfg.AccessMonth) {
			state = "Unlocked"
			if f.Beta && !a.cfg.ShowBetaFeatures {
				state = "Unlocked; beta off"
			}
			if !f.Beta && !f.Fancy {
				state = "Free"
			}
		}
		help := f.Category() + ". " + state + ". "
		if f.Fancy {
			help += "Your code unlocks it forever."
		} else if f.Beta {
			help += "Free for everyone after beta."
		}
		row := panelEntry{text: entry.Name + featureMarks(f), kind: "access-info", value: entry.ID, feature: true, help: help}
		E = append(E, row, panelEntry{text: state, info: true})
	}
	return E
}

func (a *App) openAccess() { a.openPanel(ScreenAccess) }
func (a *App) closeAccess() {
	a.openOptions()
	for i, e := range a.panel.entries {
		if e.kind == "your-access" {
			a.panel.cursor = i
			break
		}
	}
	a.all = true
}
