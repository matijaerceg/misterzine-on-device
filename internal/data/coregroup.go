package data

// The Core view (SortCore) puts the games under the core that runs them.
// An arcade core that runs two games or more is a group of its own, named
// by its core label (Derived.CoreLabel, as Details shows it: "Capcom
// CPS-1", "Sega System 16B"), and two rbfs that share a label share the
// group. More than half the arcade cores run a single game, and most of
// them are labelled with its title, so a header each would mostly repeat
// the row under it: they are gathered in one group instead. Console,
// computer and other cores are one row each, so they group by kind.

// The Core view's gathered groups, as their headers name them.
const (
	CoreSingleGame = "Single-game cores"
	CoreUnknown    = "Core unknown"
)

// coreGroupOf names the group a row belongs to, given how many games its
// core label runs, and ranks the group's kind: the multi-game arcade cores
// (A-Z by label among themselves), the single-game arcade cores, the arcade
// rows with no core, then the other kinds in the order the catalogue lists
// them.
func coreGroupOf(r *Row, games int, label string) (rank int, group string) {
	switch {
	case !r.IsArcade():
		switch r.Base {
		case "Console":
			return 3, "Console cores"
		case "Computer":
			return 4, "Computer cores"
		case "", "Other":
			return 5, "Other cores"
		}
		return 5, r.Base + " cores"
	case label == "":
		return 2, CoreUnknown
	case games < 2: // a standin counts none of its own
		return 1, CoreSingleGame
	}
	return 0, label
}

// coreGroups sets each row's Core view group. A game counts toward its own
// core, except a standin: it sorts and groups under the catalogue row it
// stands in for (SortRow), so it leaves that core's count alone.
func (ds *Dataset) coreGroups() {
	games := map[string]int{}
	for i := range ds.Rows {
		if ds.Der[i].anchor == 0 && ds.Rows[i].IsArcade() {
			games[ds.Der[i].CoreLabel]++
		}
	}
	keys := map[string][]elem{}
	for i := range ds.Rows {
		d := &ds.Der[i]
		d.coreRank, d.CoreGroup = coreGroupOf(&ds.Rows[i], games[d.CoreLabel], d.CoreLabel)
		if d.coreRank == 0 {
			d.coreGroupKey = d.coreKey
			continue
		}
		k, ok := keys[d.CoreGroup]
		if !ok {
			k = Key(d.CoreGroup)
			keys[d.CoreGroup] = k
		}
		d.coreGroupKey = k
	}
}
