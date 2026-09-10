package app

// jumpLetter moves to the first visible title of the next/previous letter.
// Only filtered/search results count; empty groups are skipped and ends stop.
func (a *App) jumpLetter(direction int) {
	if len(a.view) == 0 {
		return
	}
	initial := func(i int) rune { return a.ds.Der[a.view[i]].TitleInitial() }
	current := initial(a.cursor)
	i := a.cursor + direction
	for i >= 0 && i < len(a.view) && initial(i) == current {
		i += direction
	}
	if i < 0 || i >= len(a.view) {
		return
	}
	group := initial(i)
	for i > 0 && initial(i-1) == group {
		i--
	}
	a.cursor = i
}
