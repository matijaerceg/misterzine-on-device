package app

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/report"
)

// ReportPart is the card report's view of the list, which only the UI has:
// the order, the filters in force and chosen, and every local game with the
// rule that keeps it out of the list, if one does. The rules are the list's
// own (data.Filters.Why with the same inputs as rebuild), so the report can
// never disagree with what the player sees.
func (a *App) ReportPart() report.AppPart {
	p := report.AppPart{
		View:      a.mode.Name(),
		Search:    a.query,
		Rotation:  a.Rotation().String(),
		Rows:      len(a.ds.Rows),
		Catalogue: a.ds.NCat,
		Shown:     len(a.view),
	}
	filters := a.effectiveFilters()
	if a.mode == data.SortFavorites {
		filters.FavOnly = true
	}
	p.Effective = fmt.Sprintf("arcade only %v, deprecated hidden %v, filter by rotation %q", filters.ArcadeOnly, filters.HideDeprecated, filters.MatchRotation)
	if b, err := json.Marshal(a.filters); err == nil && string(b) != "{}" {
		p.Saved = string(b)
	}
	for src := range filters.SrcHidden {
		p.Hidden = append(p.Hidden, src)
	}
	sort.Strings(p.Hidden)
	q := searchText(a.query)
	for i := a.ds.NCat; i < len(a.ds.Rows); i++ {
		r, d := &a.ds.Rows[i], &a.ds.Der[i]
		g := report.LocalGame{K: r.K, Title: r.Title, Core: r.Core, File: r.MRA}
		if j := a.ds.SortRow(i); j != i {
			g.StandsFor = a.ds.Rows[j].K
		}
		unseen := a.seen != nil && a.seen.Unseen(r)
		g.Hidden = filters.Why(r, d, a.status(i), a.cfg.Favorites[r.K], unseen)
		if g.Hidden == "" && q != "" && !strings.Contains(searchText(d.Title), q) {
			g.Hidden = fmt.Sprintf("the search %q", a.query)
		}
		p.Local = append(p.Local, g)
	}
	return p
}
