package data

import "strings"

// Same-game groups mirror the site's umbrella rows. The site's pipeline
// marks every arcade row that is the same game as a row on another core
// with a shared G (the group's MAME parent setname), and the list folds
// the members it shows into one umbrella line after filtering and search.
// The groups are the pipeline's call alone: title or family matching here
// would make the false friends the site rejects (Tetris on three boards),
// so nothing below decides who belongs together, only how a group shows.

// GroupKeyPrefix starts an umbrella's key, the site's #g:<g> permalink: a
// member's K can equal the group's root setname (bublbobl), so the bare G
// would name two lines.
const GroupKeyPrefix = "g:"

// GroupKey is the key the cursor remembers an umbrella by.
func GroupKey(g string) string { return GroupKeyPrefix + g }

// Folds reports whether the order folds same-game rows into umbrellas: the
// date orders and the browsing orders that run over the whole catalogue.
// The Core order lists the games under the cores that run them, and the
// members are different cores by definition; Favorites and Recents are the
// player's own lists, where each star and launch is one core's.
func (m SortMode) Folds() bool {
	switch m {
	case SortUpdated, SortDebut, SortAlphabetical, SortMaker, SortYear:
		return true
	}
	return false
}

// GroupOf is the group row i folds under, "" for none. A standin folds with
// the catalogue row it stands in for (SortRow), which is where it sorts.
func (ds *Dataset) GroupOf(i int) string { return ds.Rows[ds.SortRow(i)].G }

// Group is one umbrella line of a folded view.
type Group struct {
	G       string
	Members []int // the member rows the view shows, in its order, standins included
	Cores   int   // the catalogue members among them, the "2 cores" chip
	// Anchor is the member whose place in the order the umbrella takes: the
	// first one, which holds the group's newest build under Updated, except
	// under Debut, where the umbrella dates from the group's first debut.
	Anchor int
	Open   bool // the members follow the umbrella
}

// Fold turns a filtered view into the list's lines: under an order that
// folds, each group the view shows two members or more of becomes one
// umbrella, numbered len(ds.Rows)+n for groups[n], followed by its members
// while open says it is expanded. A group left with one member shows it as
// a plain row. Other orders get the view back untouched.
func (ds *Dataset) Fold(view []int, mode SortMode, open func(g string) bool) ([]int, []Group) {
	if !mode.Folds() {
		return view, nil
	}
	// Only catalogue members count toward the two a group needs: a standin
	// is one more file for a member's game, not another core, so a member
	// and its own standin stay a plain row and the standin under it.
	count := map[string]int{}
	for _, i := range view {
		if g := ds.GroupOf(i); g != "" && ds.SortRow(i) == i {
			count[g]++
		}
	}
	at := map[string]int{}
	var groups []Group
	for _, i := range view {
		g := ds.GroupOf(i)
		if count[g] < 2 {
			continue
		}
		n, ok := at[g]
		if !ok {
			n = len(groups)
			at[g] = n
			groups = append(groups, Group{G: g, Anchor: i, Open: open != nil && open(g)})
		}
		gr := &groups[n]
		gr.Members = append(gr.Members, i)
		if ds.SortRow(i) == i {
			gr.Cores++
		}
		if mode == SortDebut && earlierDate(ds.Rows[ds.SortRow(i)].Date, ds.Rows[ds.SortRow(gr.Anchor)].Date) {
			gr.Anchor = i
		}
	}
	if len(groups) == 0 {
		return view, nil
	}
	out := make([]int, 0, len(view))
	for _, i := range view {
		n, ok := at[ds.GroupOf(i)]
		switch {
		case !ok:
			out = append(out, i)
		case i == groups[n].Anchor:
			out = append(out, len(ds.Rows)+n)
			if groups[n].Open {
				out = append(out, groups[n].Members...)
			}
		}
	}
	return out, groups
}

// earlierDate reports whether ISO date a is before b, a blank date being
// later than any other, as the date orders sink it.
func earlierDate(a, b string) bool { return a != "" && (b == "" || a < b) }

// Title is the umbrella's name: the shortest title among the members that
// debuted first, without the trailing "(Core)" that tells members apart.
func (g *Group) Title(ds *Dataset) string {
	best := -1
	for _, i := range g.Members {
		if best < 0 {
			best = i
			continue
		}
		d, bd := ds.Rows[ds.SortRow(i)].Date, ds.Rows[ds.SortRow(best)].Date
		if earlierDate(d, bd) || d == bd && len(ds.Der[i].Title) < len(ds.Der[best].Title) {
			best = i
		}
	}
	t := ds.Der[best].Title
	if strings.HasSuffix(t, ")") {
		if cut := strings.LastIndex(t, " ("); cut > 0 {
			t = t[:cut]
		}
	}
	return t
}

// Updated is the newest build among the members, Date their first debut.
func (g *Group) Updated(ds *Dataset) string {
	out := ""
	for _, i := range g.Members {
		out = max(out, ds.Rows[i].Updated)
	}
	return out
}

func (g *Group) Date(ds *Dataset) string {
	out := ""
	for _, i := range g.Members {
		if d := ds.Rows[ds.SortRow(i)].Date; earlierDate(d, out) {
			out = d
		}
	}
	return out
}

// Preview is the member whose screenshot stands for the umbrella: the
// first one with a picture, else the first.
func (g *Group) Preview(ds *Dataset) int {
	for _, i := range g.Members {
		if ds.Rows[i].Img != "" {
			return i
		}
	}
	return g.Members[0]
}
