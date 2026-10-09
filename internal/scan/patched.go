package scan

import (
	"path"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// InferParents fills in the parent of a patched set that names none, so it
// joins its game as a version rather than standing as a game of its own. A
// trainer MRA (Insert-Coin's Cave pack, patched with kasaski's cv1k_patcher)
// keeps the game's <name> and core and loads the game's own archive for all
// but its program ROM, which comes from an archive of its own under a new
// setname; nothing in the file says which game it is.
//
// The rule stays narrow, as games on one core do share archives (a BIOS,
// sound samples): the file sits under an _alternatives folder, and of the
// archives it loads besides its own set's, exactly one is the setname of a
// game on the same core with the same name (a program archive listed with
// the game's as a fallback, futaribljt.zip|futariblj.zip, counts too). That
// game is one of rows (the catalogue) or of the scanned files themselves,
// alts included. Files with a parent, or with no name, are left alone.
func InferParents(alts []Alt, more []Alt, rows []data.Row) {
	type game struct{ core, title string }
	known := map[game]map[string]bool{} // -> setnames
	add := func(core, title, sn string) {
		if sn = identity(sn); sn == "" || core == "" || title == "" {
			return
		}
		g := game{coreStem(core), titleKey(title)}
		if known[g] == nil {
			known[g] = map[string]bool{}
		}
		known[g][sn] = true
	}
	for _, r := range rows {
		if r.IsArcade() {
			add(r.Core, r.Title, r.SN)
		}
	}
	for _, list := range [][]Alt{alts, more} {
		for _, a := range list {
			add(a.RBF, a.Name, a.Setname)
		}
	}
	for i := range alts {
		a := &alts[i]
		if a.Parent != "" || a.Name == "" || !underAlternatives(a.Path) {
			continue
		}
		sn := identity(a.Setname)
		parent := ""
		for _, z := range append(append([]string{}, a.Zips...), a.OtherZips...) {
			stem := zipStem(z)
			if stem == "" || stem == sn {
				continue
			}
			var sets map[string]bool
			for g, s := range known {
				if g.title == titleKey(a.Name) && sameStem(g.core, coreStem(a.RBF)) {
					sets = s
					break
				}
			}
			if !sets[stem] {
				continue
			}
			if parent != "" && parent != stem {
				parent = "" // two games' archives: no telling which it is
				break
			}
			parent = stem
		}
		if parent != "" {
			a.Parent = parent
		}
	}
}

// zipStem is the setname an archive name stands for ("akatana.zip" ->
// akatana), or "" when it is no setname.
func zipStem(z string) string {
	z = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(z, "\\", "/")))
	if !strings.HasSuffix(z, ".zip") {
		return ""
	}
	return identity(strings.TrimSuffix(path.Base(z), ".zip"))
}
