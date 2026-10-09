package scan

import (
	"path"
	"slices"
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
//
// The board runs this twice per scan over thousands of files, so titles are
// compared only where setnames already point the way (a file's other
// archives against the games named so), and without allocating (sameTitle).
func InferParents(alts []Alt, more []Alt, rows []data.Row) {
	// the files that could take a parent, with the other archives they load
	cand := map[int][]string{}
	wanted := map[string]bool{}
	for i := range alts {
		a := &alts[i]
		if a.Parent != "" || a.Name == "" || !underAlternatives(a.Path) {
			continue
		}
		sn := identity(a.Setname)
		var stems []string
		for _, z := range a.Zips {
			stems = appendStem(stems, z, sn)
		}
		for _, z := range a.OtherZips {
			stems = appendStem(stems, z, sn)
		}
		if len(stems) > 0 {
			cand[i] = stems
			for _, s := range stems {
				wanted[s] = true
			}
		}
	}
	if len(cand) == 0 {
		return
	}
	// coreStem runs a regexp, slow on the board: once per core name
	stems := map[string]string{}
	stemOf := func(core string) string {
		s, ok := stems[core]
		if !ok {
			s = coreStem(core)
			stems[core] = s
		}
		return s
	}
	type game struct{ core, title string }
	known := map[string][]game{} // setname -> the games it names, by core and title
	add := func(core, title, sn string) {
		if sn = identity(sn); wanted[sn] && core != "" && title != "" {
			known[sn] = append(known[sn], game{stemOf(core), title})
		}
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
	for i, sets := range cand {
		a := &alts[i]
		core := stemOf(a.RBF)
		parent := ""
		for _, stem := range sets {
			match := false
			for _, g := range known[stem] {
				if sameStem(g.core, core) && sameTitle(g.title, a.Name) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
			if parent != "" {
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

// sameTitle is titleKey(a) == titleKey(b) without building either key when
// both are plain printable ASCII, as nearly every MRA name is: then the keys
// are the words lowercased, so the words compare case-insensitively. Plain
// byte loops, as the general string routines cost tens of microseconds a
// call on the board's ARM cores.
func sameTitle(a, b string) bool {
	if !plainASCII(a) || !plainASCII(b) {
		return titleKey(a) == titleKey(b)
	}
	i, j := 0, 0
	for {
		for i < len(a) && asciiSpace(a[i]) {
			i++
		}
		for j < len(b) && asciiSpace(b[j]) {
			j++
		}
		if i == len(a) || j == len(b) {
			return i == len(a) && j == len(b)
		}
		// one word each, compared case-insensitively to its end
		for i < len(a) && !asciiSpace(a[i]) {
			if j == len(b) || asciiSpace(b[j]) || lowerASCII(a[i]) != lowerASCII(b[j]) {
				return false
			}
			i++
			j++
		}
		if j < len(b) && !asciiSpace(b[j]) {
			return false
		}
	}
}

// asciiSpace reports what data.ASCII turns into a space and strings.Fields
// splits on, within ASCII.
func asciiSpace(c byte) bool {
	return c == ' ' || c >= '\t' && c <= '\r'
}

func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// plainASCII reports whether data.ASCII leaves s as it is, apart from spacing.
func plainASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c < 0x20 || c >= 0x7f) && !asciiSpace(c) {
			return false
		}
	}
	return true
}

// appendStem adds the setname archive z stands for to stems, unless it is
// the file's own set (sn), no setname, or already there.
func appendStem(stems []string, z, sn string) []string {
	if s := zipStem(z); s != "" && s != sn && !slices.Contains(stems, s) {
		return append(stems, s)
	}
	return stems
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
