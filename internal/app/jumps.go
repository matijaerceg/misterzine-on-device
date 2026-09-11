package app

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// jumpGroup moves to the first visible title of the next/previous letter or month.
// Only filtered/search results count; empty groups are skipped and ends stop.
func (a *App) jumpGroup(direction int) {
	if len(a.view) == 0 {
		return
	}
	current := a.jumpGroupKey(a.cursor)
	i := a.cursor + direction
	for i >= 0 && i < len(a.view) && a.jumpGroupKey(i) == current {
		i += direction
	}
	if i < 0 || i >= len(a.view) {
		return
	}
	group := a.jumpGroupKey(i)
	for i > 0 && a.jumpGroupKey(i-1) == group {
		i--
	}
	a.cursor = i
	a.top = a.screenLine(i)
	a.shortPage = true
	if a.mode != data.SortAlphabetical && a.mode != data.SortFavorites {
		label := "Date unknown"
		if a.mode == data.SortYear {
			label = "Year unknown"
			if group != "" {
				label = group
			}
		} else if month, err := time.Parse("2006-01", group); err == nil {
			label = month.Format("January 2006")
		}
		a.Notice(label, 2*time.Second)
	}
}

func (a *App) jumpGroupKey(pos int) string {
	i := a.view[pos]
	if a.mode == data.SortAlphabetical || a.mode == data.SortFavorites {
		return string(a.ds.Der[i].TitleInitial())
	}
	if a.mode == data.SortYear {
		return data.ReleaseYear(a.ds.Rows[i].Year) // "" gathers the unknown years at the end
	}
	date := a.ds.Rows[i].Updated
	if a.mode == data.SortDebut {
		date = a.ds.Rows[i].Date
	}
	if len(date) >= 7 {
		return date[:7] // year and month: different years remain separate groups
	}
	return ""
}
