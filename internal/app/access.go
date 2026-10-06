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
	if f.Fancy && !f.Covered(a.cfg.AccessMonth) {
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
		row.help += " Your code unlocks it forever."
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
		{text: "Your access: " + a.cfg.AccessMonth.Short(), kind: "access-info", help: "The month covers Supporter extras, not an expiry. You do not need to stay subscribed."},
		{text: "Enter MisterZine code", kind: "enter-code", opensPage: true, help: "Newer codes include earlier Supporter extras. Older codes never reduce your access."},
	}

	// Size the two columns using only the features visible in this overview.
	nameW, statusW := 0, 0
	for _, entry := range access.Catalog() {
		if !entry.Fancy && !a.cfg.ShowBetaFeatures {
			continue
		}
		nameW = max(nameW, a.sm.Width(entry.Name+featureMarks(entry.Feature)))
		statusW = max(statusW, a.sm.Width(a.featureAccessStatus(entry.Feature)))
	}
	columns := nameW+2*a.sm.W+statusW <= a.lay.Body.Dx()-8-a.sm.W
	for _, supporter := range []bool{true, false} {
		heading := "Supporter extras"
		if !supporter {
			heading = "Free beta features"
		}
		E = append(E, panelEntry{text: heading, header: true, info: true})
		if !supporter && !a.cfg.ShowBetaFeatures {
			E = append(E, panelEntry{text: "Enable Beta in Options", kind: "access-info", help: "Enable Beta in Options to try upcoming free features. No code needed."})
			continue
		}
		for _, entry := range access.Catalog() {
			f := entry.Feature
			if f.Fancy != supporter || (!supporter && !f.Beta) {
				continue
			}
			state := a.featureAccessStatus(f)
			help := "Available with Beta enabled. No code needed."
			if f.Fancy {
				help = "Your code unlocks it forever."
				if f.Covered(a.cfg.AccessMonth) {
					help = "Code covers this. " + help
				} else {
					help = state + ". " + help
				}
				if f.Beta {
					if a.cfg.ShowBetaFeatures {
						help += " Beta is on."
					} else {
						help += " Enable Beta to use."
					}
				}
			}
			row := panelEntry{text: entry.Name + featureMarks(f), kind: "access-info", value: entry.ID, feature: true, help: help}
			if columns {
				row.accessStatus = state
				E = append(E, row)
			} else {
				E = append(E, row, panelEntry{text: state, info: true})
			}
		}
	}
	if a.cfg.AccessMonth.Valid() {
		E = append(E, panelEntry{header: true, info: true}, panelEntry{text: "Forget code", kind: "forget-code", opensPage: true, help: "Remove saved unlocks on this device. Settings stay saved. Enter your code again to restore access."})
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

func (a *App) featureAccessStatus(f access.Feature) string {
	if !f.Covered(a.cfg.AccessMonth) {
		return "Needs " + f.Since.Short() + "+"
	}
	if !f.Beta && !f.Fancy {
		return "Free"
	}
	if f.Beta && !a.cfg.ShowBetaFeatures {
		return "Beta off"
	}
	if !f.Fancy {
		return "Available"
	}
	return "Unlocked"
}
