package app

// A nil map is the startup default: only Data is open. This lives on App
// because returning to Filters restores its own saved panel state.
func (a *App) optionSectionOpen(section string) bool {
	if a.optionsOpen == nil {
		return section == "Data"
	}
	return a.optionsOpen[section]
}

func (a *App) visibleOptionsEntries() []panelEntry {
	var out []panelEntry
	hidden := false
	for _, e := range a.optionsEntries() {
		if e.header {
			hidden = e.kind == "options-section" && !a.optionSectionOpen(e.value)
			out = append(out, e)
			if e.kind == "options-section" && !hidden {
				// Match the half-row gap between sections without adding a
				// selectable stop or padding beneath a collapsed heading.
				out = append(out, panelEntry{header: true, info: true})
			}
		} else if !hidden {
			out = append(out, e)
		}
	}
	return out
}

func (a *App) onOptionsSection() bool {
	return a.screen == ScreenOptions && a.panel.cursor < len(a.panel.entries) &&
		a.panel.entries[a.panel.cursor].kind == "options-section"
}

func (a *App) expandOptionsSection(open bool) bool {
	if !a.onOptionsSection() {
		return false
	}
	if a.optionsOpen == nil {
		a.optionsOpen = map[string]bool{"Data": true}
	}
	a.optionsOpen[a.panel.entries[a.panel.cursor].value] = open
	a.buildPanel()
	return true
}
