package scan

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// For launchers discovered in a top-level menu directory, Main takes the
// first path component as the arcade root, regardless of nested menus.
func arcadeRoot(mra string) string {
	root, _, _ := strings.Cut(mra, "/")
	if strings.HasPrefix(root, "_") {
		return root
	}
	return "_Arcade"
}

type arcadeFiles struct {
	names map[string]Core // newest build by name, for matching catalogue core names
	rbfs  []Core          // every filename, for Main's prefix and boundary rules
}

func (idx *Index) arcadeFor(mra string) *arcadeFiles {
	if idx == nil {
		return nil
	}
	return idx.arcades[arcadeRoot(mra)]
}

func (files *arcadeFiles) lookup(name string) (Core, bool) {
	if files == nil {
		return Core{}, false
	}
	return lookup(files.names, name)
}

func (idx *Index) HasMRACore(mra, name string) bool {
	if f := idx.launcher(mra); f != nil && f.rbf == name {
		return f.core.Path != ""
	}
	if arcadeRoot(mra) == "_Arcade" {
		return idx.HasArcadeCore(name)
	}
	_, ok := idx.arcadeFor(mra).mainRBF(name)
	return ok
}

// Probe only top-level menu directories. The _Arcade marker also finds a
// partially installed collection whose cores folder hasn't arrived yet.
func collectionRoots(card string, top []os.DirEntry) ([]string, error) {
	var roots []string
	var problems []error
	for _, e := range top {
		n := e.Name()
		if !e.IsDir() || !strings.HasPrefix(n, "_") || strings.EqualFold(n, "_Arcade") || skipDir(n) {
			continue
		}
		for _, marker := range []string{"cores", "_Arcade"} {
			st, err := os.Lstat(filepath.Join(card, n, marker))
			if err == nil && st.IsDir() {
				roots = append(roots, n)
				break
			}
			if err != nil && !os.IsNotExist(err) {
				problems = append(problems, err)
			}
		}
	}
	return roots, errors.Join(problems...)
}

type collectionScan struct {
	games, alternatives  []Alt
	skipped, skippedDirs []Skipped
	err                  error
	launchers            map[string]*collectionLauncher
	versions             map[string][]string
}

// Each physical file gets its own status even when another collection ships
// the same game and the same core filename.
type collectionLauncher struct {
	rbf     string
	core    Core
	matched bool // this file was assigned to a catalogue row
}

func (idx *Index) launcher(p string) *collectionLauncher {
	if idx == nil {
		return nil
	}
	return idx.collections.launchers[p]
}

func (idx *Index) CollectionLauncher(p string) bool {
	return arcadeRoot(p) != "_Arcade" || idx.launcher(p) != nil
}

func (idx *Index) VersionCore(p string) (name string, installed bool) {
	if f := idx.launcher(p); f != nil {
		return f.rbf, f.core.Path != ""
	}
	return "", false
}

func (idx *Index) recordLauncher(a Alt) *collectionLauncher {
	core, _ := idx.arcadeFor(a.Path).mainRBF(a.RBF)
	f := &collectionLauncher{rbf: a.RBF, core: core}
	idx.collections.launchers[a.Path] = f
	return f
}

// ResolveCollections is run once on a fresh core index before publishing it.
// Only the scan's copy of the rows changes. Keys, metadata and the published
// path survive a rescan, removal or relocation of the collection.
func (idx *Index) ResolveCollections(card, cache string, rows []data.Row) ([]data.Row, bool) {
	c := &idx.collections
	matches := map[int][]Alt{}
	if len(idx.collectionRoots) > 0 {
		files, skipped, dirs, err := scanArcadeMRAs(card, cache, idx.collectionRoots...)
		c.skipped, c.skippedDirs, c.err = skipped, dirs, errors.Join(c.err, err)
		c.launchers = map[string]*collectionLauncher{}
		c.versions = map[string][]string{}
		owners := catalogueOwners(rows)
		for _, a := range files {
			file := idx.recordLauncher(a)
			if underAlternatives(a.Path) {
				c.alternatives = append(c.alternatives, a)
				continue
			}
			c.games = append(c.games, a)
			if i := idx.matchCollection(a, rows, owners.bySN[identity(a.Setname)]); i >= 0 && !isBIOS(a) {
				matches[i] = append(matches[i], a)
				file.matched = true
			}
		}
	}
	out := rows
	changed := false
	for i, r := range rows {
		canonical := publishedMRA(r)
		primary := canonical
		if copies := matches[i]; len(copies) > 0 {
			// Sorted scan paths give a stable default. A playable regular
			// copy comes first, then a playable collection copy, then a
			// present launcher whose own core is missing.
			if a, ok := ParseMRAHeader(filepath.Join(card, filepath.FromSlash(canonical))); ok &&
				identity(a.Setname) == identity(r.SN) && loadsCore(idx.arcadeFor(canonical), a.RBF, r.Core) {
				a.Path = canonical
				copies = append([]Alt{a}, copies...)
				idx.recordLauncher(a).matched = true
			}
			primary = copies[0].Path
			for _, a := range copies {
				if idx.HasMRACore(a.Path, a.RBF) {
					primary = a.Path
					break
				}
			}
			for _, a := range copies {
				if a.Path != primary {
					c.versions[r.K] = append(c.versions[r.K], a.Path)
				}
			}
		}
		if primary != r.MRA {
			if !changed {
				out = slices.Clone(rows)
				changed = true
			}
			out[i].MRA, out[i].CatalogueMRA = primary, ""
			if primary != canonical {
				out[i].CatalogueMRA = canonical
			}
		}
	}
	return out, changed
}

func publishedMRA(r data.Row) string {
	if r.CatalogueMRA != "" {
		return r.CatalogueMRA
	}
	return r.MRA
}

// An exact game set and compatible core identify the build. The path and
// Ongo's known source break ties between mirrors. Leave ambiguous files to
// the ordinary local-game rules rather than claiming multiple releases.
func (idx *Index) matchCollection(a Alt, rows []data.Row, candidates []int) int {
	if a.RBF == "" {
		return -1
	}
	root, rel, _ := strings.Cut(a.Path, "/")
	rel = strings.TrimPrefix(rel, "_Arcade/")
	best, score := -1, -1
	for _, i := range candidates {
		r := rows[i]
		if r.IsLocal() || r.MRA == "" || !loadsCore(idx.arcadeFor(a.Path), a.RBF, r.Core) {
			continue
		}
		rank := 0
		if strings.EqualFold(strings.TrimPrefix(publishedMRA(r), "_Arcade/"), rel) {
			rank++
		}
		if strings.EqualFold(root, "_Ongo") && r.Src == "ongo" {
			rank += 2
		}
		if rank > score {
			best, score = i, rank
		} else if rank == score {
			best = -1
		}
	}
	return best
}

func (idx *Index) HasCollections() bool { return len(idx.collectionRoots) > 0 }

// CollectionAlternatives includes the isolated sets preserved by an adapter.
func (idx *Index) CollectionAlternatives(alts []Alt) []Alt {
	if idx.HasCollections() {
		for _, a := range alts {
			idx.recordLauncher(a)
		}
	}
	return append(alts, idx.collections.alternatives...)
}

func (idx *Index) MergeCollectionVersions(resolved map[string][]string) {
	MergeVersions(resolved, idx.collections.versions)
}

// CheckMRACore rechecks the actual launcher and just its own cores directory
// at Start, so removal after a scan cannot hand Main a broken launch.
func CheckMRACore(card, rel string) error {
	a, ok := ParseMRAHeader(filepath.Join(card, filepath.FromSlash(rel)))
	if !ok || a.RBF == "" {
		return errors.New("cannot read launcher core")
	}
	dir := path.Join(arcadeRoot(rel), "cores")
	entries, err := os.ReadDir(filepath.Join(card, dir))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.EqualFold(path.Ext(name), ".rbf") && (rbfFor(name, a.RBF) || rbfFor(name, "Arcade-"+a.RBF)) {
			return nil
		}
	}
	return errors.New(a.RBF + " core missing from " + dir)
}
