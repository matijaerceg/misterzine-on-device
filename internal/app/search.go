package app

import (
	"strings"
	"unicode"
)

// Search reports the session's keyboard title query.
func (a *App) Search() string { return a.query }

func searchText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}

func (a *App) typeSearch(ch rune) bool {
	// The bitmap UI and keyboard input use printable ASCII. Keep a bounded
	// query, with its tail visible when the heading cannot fit it all.
	if ch < 32 || ch > 126 || len(a.query) >= 80 {
		return false
	}
	a.setSearch(a.query + string(ch))
	return true
}

func (a *App) setSearch(query string) {
	k := a.CursorKey()
	a.query = query
	a.cursor, a.top = 0, 0
	a.rebuild()
	a.moveToKey(k)
}
