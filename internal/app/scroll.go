package app

// Keep selection on the middle row where possible, without blank space at
// either end. With an even number of rows, use the lower of the middle pair.
func centeredTop(row, total, visible int) int {
	return max(0, min(row-max(visible, 1)/2, total-visible))
}

// jumpPanelSection moves the cursor a section back or on (L and R, as
// the list's letter jumps): in Filters to the section's heading, in
// Options, whose headings are not selectable, to the section's first
// row. From inside a section, back goes to the previous section, not to
// the top of this one. Jumps stop at either end.
func (a *App) jumpPanelSection(direction int) bool {
	p := &a.panel
	var sections []int
	current := -1
	for i, e := range p.entries {
		heading := e.header && !e.info && e.kind != ""
		if a.screen == ScreenOptions {
			heading = e.header && e.glyph != ""
		}
		if heading {
			sections = append(sections, i)
			if i <= p.cursor {
				current = len(sections) - 1
			}
		}
	}
	if len(sections) == 0 {
		return false
	}
	target := max(0, min(current+direction, len(sections)-1))
	i := sections[target]
	for a.screen == ScreenOptions && i+1 < len(p.entries) && p.entries[i].info {
		i++
	}
	p.cursor = i
	a.all = true
	return true
}

func (a *App) expandFilterSection(open bool) bool {
	// Years are the inner level: close an open decade before its section.
	if decade := a.selectedDecade(); decade != "" && (open || a.panel.yearOpen[decade]) {
		return a.expandYears(open)
	}
	p := &a.panel
	for i := p.cursor; i >= 0; i-- {
		e := p.entries[i]
		if e.header && !e.info && e.kind != "" {
			if p.sectionClosed == nil {
				p.sectionClosed = map[string]bool{}
			}
			p.sectionClosed[e.kind] = !open
			if !open {
				p.cursor = i
			}
			a.buildPanel()
			return true
		}
	}
	return false
}
