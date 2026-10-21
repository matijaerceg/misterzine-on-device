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
// date orders and the browsing orders that run over the whole catalogue,
// the genre views among them (one genre of it, A-Z). The Core order lists
// the games under the cores that run them, and the members are different
// cores by definition; Favorites and Recents are the player's own lists,
// where each star and launch is one core's.
func (m SortMode) Folds() bool {
	switch m {
	case SortUpdated, SortDebut, SortAlphabetical, SortMaker, SortYear:
		return true
	}
	return m.Genre() != ""
}

// GroupOf is the group row i folds under, "" for none. A standin folds with
// the catalogue row it stands in for (SortRow), which is where it sorts.
func (ds *Dataset) GroupOf(i int) string { return ds.Rows[ds.SortRow(i)].G }

// Group is one umbrella line of a folded view: a same-game group, or a
// core batch (batches.go) when Core is set.
type Group struct {
	G        string
	Members  []int // the member rows the view shows, in its order, standins included
	Versions int   // the catalogue members among them: a group's "2 versions", a batch's games
	// Anchor is the member whose place in the order the umbrella takes: the
	// first one, which holds the group's newest build under Updated, except
	// under Debut, where a same-game umbrella dates from the group's first
	// debut. A batch's members share the date its order sorts by.
	Anchor int
	Open   bool // the members follow the umbrella
	// Core and Day name a core batch: the rbf its games share and the date
	// they share under the order (the build under Updated, the debut under
	// Debut). Total is the batch's games in the whole catalogue, more than
	// Versions when a search shows only some of them.
	Core, Day string
	Total     int
}

// Batch reports whether the umbrella is a core batch.
func (g *Group) Batch() bool { return g.Core != "" }

// Key is the key the cursor remembers the umbrella by.
func (g *Group) Key() string {
	if g.Batch() {
		return BatchKey(g.G)
	}
	return GroupKey(g.G)
}

// FoldOpts says which umbrellas Fold makes: each kind is a beta feature of
// its own. Open says which umbrellas are expanded.
type FoldOpts struct {
	Groups  bool // same-game groups
	Batches bool // core batches, under the date orders only
	// Search is on while a title search narrows the view: a batch then
	// gathers even one matching game under its header (batches.go).
	Search bool
	Open   func(g string) bool
}

// Fold turns a filtered view into the list's lines: under an order that
// folds, each same-game group the view shows two members or more of becomes
// one umbrella, and under the date orders the plain rows left over that
// share a core and a date fold into core batches (batches.go). Umbrellas are
// numbered len(ds.Rows)+n for groups[n], each followed by its members while
// Open says it is expanded. A group left with one member shows it as a plain
// row. Other orders get the view back untouched.
func (ds *Dataset) Fold(view []int, mode SortMode, o FoldOpts) ([]int, []Group) {
	groups, batches := o.Groups && mode.Folds(), o.Batches && mode.Batches()
	if !groups && !batches {
		return view, nil
	}
	// Only catalogue members count toward the two a group needs: a standin
	// is one more file for a member's game, not another core, so a member
	// and its own standin stay a plain row and the standin under it.
	count := map[string]int{}
	if groups {
		for _, i := range view {
			if g := ds.GroupOf(i); g != "" && ds.SortRow(i) == i {
				count[g]++
			}
		}
	}
	// key is the umbrella row i folds under, "" for none: its same-game
	// group, else its core batch
	key := func(i int) string {
		if g := ds.GroupOf(i); count[g] >= 2 {
			return g
		}
		return ""
	}
	if batches {
		bcount := map[string]int{}
		for _, i := range view {
			if key(i) == "" && ds.SortRow(i) == i {
				if b := ds.batchOf(i, mode, o.Search); b != "" {
					bcount[b]++
				}
			}
		}
		grouped := key
		key = func(i int) string {
			if g := grouped(i); g != "" {
				return g
			}
			// a search keeps even one match under a batch the whole
			// catalogue has; otherwise the view must show the batch
			b := ds.batchOf(i, mode, o.Search)
			if b != "" && (bcount[b] >= BatchMin || o.Search && bcount[b] > 0 && ds.batchTotal(b, mode) >= BatchMin) {
				return b
			}
			return ""
		}
	}
	at := map[string]int{}
	var out []Group
	for _, i := range view {
		g := key(i)
		if g == "" {
			continue
		}
		n, ok := at[g]
		if !ok {
			n = len(out)
			at[g] = n
			u := Group{G: g, Anchor: i, Open: o.Open != nil && o.Open(g)}
			if count[g] < 2 {
				r := ds.SortRow(i)
				u.Core, u.Day, u.Total = ds.Rows[r].Core, batchDate(&ds.Rows[r], mode), ds.batchTotal(g, mode)
			}
			out = append(out, u)
		}
		gr := &out[n]
		gr.Members = append(gr.Members, i)
		if ds.SortRow(i) == i {
			gr.Versions++
		}
		if !gr.Batch() && mode == SortDebut && earlierDate(ds.Rows[ds.SortRow(i)].Date, ds.Rows[ds.SortRow(gr.Anchor)].Date) {
			gr.Anchor = i
		}
	}
	if len(out) == 0 {
		return view, nil
	}
	return ds.lines(view, out, at, key), out
}

// lines lays the folded view out: each umbrella at its anchor's place,
// followed by its members while open, every other row as it was.
func (ds *Dataset) lines(view []int, groups []Group, at map[string]int, key func(int) string) []int {
	out := make([]int, 0, len(view))
	for _, i := range view {
		n, ok := at[key(i)]
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
	return out
}

// earlierDate reports whether ISO date a is before b, a blank date being
// later than any other, as the date orders sink it.
func earlierDate(a, b string) bool { return a != "" && (b == "" || a < b) }

// Title is the umbrella's name: the shortest title among the members that
// debuted first, without the trailing "(Core)" that tells members apart. A
// core batch is named after its core.
func (g *Group) Title(ds *Dataset) string {
	if g.Batch() {
		if l := ds.Der[ds.SortRow(g.Anchor)].CoreLabel; l != "" {
			return l
		}
		return g.Core
	}
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
