package scan

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Walk limits: deeper trees and bigger sets are almost certainly organiser
// output or a mirrored ROM archive, not games to list.
const localMaxDepth = 6

var localMaxFiles = 20000 // a variable so a test can lower it

// ErrTooManyMRAs stops the walk once localMaxFiles headers have been read.
var ErrTooManyMRAs = errors.New("too many MRA files under _Arcade; list cut short")

// SlotLocalSnap is the picture slot of a local row whose shot comes from the
// image service rather than the catalogue (fetch.SlotLocalSnap).
const SlotLocalSnap = "lsnap"

// LocalResult is what DiscoverLocal hands the host to merge.
type LocalResult struct {
	Rows    []data.Row          // one per unmatched setname, sorted by K
	Alts    map[string][]string // local K -> further copies of the same set
	Files   int                 // MRAs the walk read, matched or not
	Skipped []Skipped
	// SkippedDirs are the folders under _Arcade the walk did not enter, each
	// with the rule that kept it out (the cores and _alternatives folders,
	// read elsewhere, are not listed).
	SkippedDirs []Skipped
	// Accounted are the files the walk read that make no row of their own,
	// each with the reason: the catalogue's file, a version of a catalogue
	// or local game, or a catalogue game this card cannot run either way.
	Accounted []Accounted
	// OwnFiles and VersionFiles count the files that are in the list through
	// the catalogue: its own MRAs, and the versions tied to its games.
	OwnFiles, VersionFiles int
	// Versions are further card files a catalogue game that runs here
	// offers as versions, by catalogue key: its own sets outside the
	// _alternatives folders, and sets for another core that is on the card.
	// MergeVersions adds them to the family resolver's result.
	Versions map[string][]string
	// VersionCores names each of those files' own core, so the picker can
	// drop one whose core leaves the card; VersionOwner is the first row
	// that offers it, where a stand-in that gives way hands its records.
	VersionCores, VersionOwner map[string]string
	Err                        error
}

// Accounted is one card file that is no row of its own, for the diagnostic
// report: why, and the row it belongs to when there is one.
type Accounted struct {
	Path   string
	Reason string
	K      string
}

// skipDir names the folders the walk never enters: cores, anything an
// organiser wrote (arcade_organizer's _Organized tree duplicates every MRA
// many times over), the _alternatives folders ScanAlternatives already
// reads, and hidden folders.
func skipDir(name string) bool {
	return skipDirReason(name) != "" || strings.EqualFold(name, "cores") || strings.EqualFold(name, "_alternatives")
}

// skipDirReason is why skipDir keeps a folder out, for the folders worth
// telling a player about.
func skipDirReason(name string) string {
	l := strings.ToLower(name)
	switch {
	case strings.Contains(l, "organized"):
		return "an organiser's folder (names containing \"organized\" are not read)"
	case strings.HasPrefix(l, "."):
		return "a hidden folder"
	}
	return ""
}

// ScanArcadeMRAs reads the header of every MRA under _Arcade outside the
// folders skipDir names, symlinks excluded, through the same per-directory
// cache scheme as the alternatives scan (cachePath, "" to disable).
func ScanArcadeMRAs(card, cachePath string) ([]Alt, []Skipped, error) {
	alts, skipped, _, err := scanArcadeMRAs(card, cachePath)
	return alts, skipped, err
}

// scanArcadeMRAs is ScanArcadeMRAs that also names the folders it left out.
func scanArcadeMRAs(card, cachePath string) ([]Alt, []Skipped, []Skipped, error) {
	dc := openDirCache(cachePath)
	var out []Alt
	var skipped, dirs []Skipped
	files := 0
	var walk func(rel string, depth int) bool
	walk = func(rel string, depth int) bool {
		abs := filepath.Join(card, filepath.FromSlash(rel))
		st, err := os.Stat(abs)
		if err != nil {
			if rel != "_Arcade" || !os.IsNotExist(err) {
				dc.problems = append(dc.problems, err)
			}
			return true
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			dc.problems = append(dc.problems, err)
			return true
		}
		alts, skips := dc.dir(card, rel, rel, st.ModTime().UnixNano(), entries)
		out = append(out, alts...)
		skipped = append(skipped, skips...)
		files += len(alts) + len(skips)
		if files > localMaxFiles {
			dc.problems = append(dc.problems, ErrTooManyMRAs)
			return false
		}
		for _, e := range entries {
			sub := path.Join(rel, e.Name())
			// e.IsDir() is false for a symlink whatever it points at.
			if e.Type()&os.ModeSymlink != 0 {
				if st, err := os.Stat(filepath.Join(abs, e.Name())); err == nil && st.IsDir() {
					dirs = append(dirs, Skipped{Path: sub, Reason: "a symbolic link (links to folders are not followed)"})
				}
				continue
			}
			if !e.IsDir() {
				continue
			}
			if why := skipDirReason(e.Name()); why != "" {
				dirs = append(dirs, Skipped{Path: sub, Reason: why})
				continue
			}
			if skipDir(e.Name()) {
				continue
			}
			if depth >= localMaxDepth {
				dirs = append(dirs, Skipped{Path: sub, Reason: fmt.Sprintf("more than %d folders deep", localMaxDepth)})
				continue
			}
			if !walk(sub, depth+1) {
				return false
			}
		}
		return true
	}
	walk("_Arcade", 0)
	dc.save("local scan cache")
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, skipped, dirs, errors.Join(dc.problems...)
}

// DiscoverLocal turns the MRAs the catalogue does not account for into rows.
// alts are the alternatives already scanned; attached holds every path the
// family resolver tied to a catalogue row; status is what the list shows for
// each catalogue row (nil: a row runs when its core is on the card). A file
// is accounted for when its path is a catalogue MRA or attached, or when it
// belongs to a catalogue game that runs here (owners.of): a game the
// catalogue knows is never listed twice. Such a file whose own core is on the
// card, another core or the row's own, is one more version of every owner
// that runs (Versions), which MergeVersions adds to the resolver's lists.
//
// A game the catalogue knows only through cores the card has not got is the
// exception. Its row is greyed and refuses to launch, so a file that plays
// it on a core the card does have (another author's, or one from a database
// the tracker does not follow) would vanish between the two rules. Such a
// file becomes a standin row instead, and idx is what decides it: without a
// core index nothing is on the card and the old rule stands.
//
// The rest are grouped by setname; the copy outside any _alternatives folder
// with the shortest path becomes the row and the others its alternatives. Set
// snap when the image service is available: such rows then ask it for a shot
// by setname.
func DiscoverLocal(card, cachePath string, catalogue []data.Row, idx *Index, status []data.Status, alts []Alt, attached map[string]bool, snap bool) LocalResult {
	walked, skipped, dirs, err := scanArcadeMRAs(card, cachePath)
	res := LocalResult{Alts: map[string][]string{}, Files: len(walked), Skipped: skipped, SkippedDirs: dirs, Err: err,
		Versions: map[string][]string{}, VersionCores: map[string]string{}, VersionOwner: map[string]string{}}
	own := catalogueOwners(catalogue)
	cores := map[string]bool{}
	has := func(core string) bool { // coreOnCard, remembered per name: a miss scans the index
		v, ok := cores[core]
		if !ok {
			v = coreOnCard(idx, core)
			cores[core] = v
		}
		return v
	}
	// A row runs when the list shows it on the card: its core and its own
	// MRA are there. Without statuses, its core alone decides.
	runs := func(i int) bool {
		if status != nil {
			return i < len(status) && status[i].Found()
		}
		return has(catalogue[i].Core)
	}
	// version offers file a to every row among rows that runs here; a file
	// the family resolver already attached skips the rows on its own core,
	// which that resolver has covered. It reports whether any row took it.
	version := func(a Alt, rows []int, attached bool) bool {
		took := false
		for _, i := range rows {
			if !runs(i) || attached && sameCore(a.RBF, catalogue[i].Core) {
				continue
			}
			k := catalogue[i].K
			res.Versions[k] = append(res.Versions[k], a.Path)
			res.VersionCores[a.Path] = a.RBF
			if res.VersionOwner[a.Path] == "" {
				res.VersionOwner[a.Path] = k
			}
			took = true
		}
		return took
	}
	var loose []Alt
	present := map[string]bool{}
	standins := map[string]bool{} // path -> the catalogue lists this game, unrunnably
	for _, a := range append(append([]Alt{}, walked...), alts...) {
		// the catalogue's own files are its rows
		if own.paths[a.Path] {
			res.OwnFiles++
			continue
		}
		bios := isBIOS(a)
		rows, ambiguous := own.of(a)
		if attached[a.Path] {
			// already a version of a catalogue game; another row of the same
			// game that runs here on another core offers it too
			res.VersionFiles++
			if !bios && !ambiguous && has(a.RBF) {
				version(a, rows, true)
			}
			continue
		}
		if bios {
			res.Skipped = append(res.Skipped, Skipped{Path: a.Path, Reason: "BIOS, not a game", Size: a.Size, Mtime: a.Mtime})
			continue
		}
		if len(rows) > 0 {
			runnable := false
			for _, i := range rows {
				runnable = runnable || runs(i)
			}
			k := catalogue[rows[0]].K
			switch {
			case runnable && ambiguous:
				// several catalogue games share this family: offering the
				// file under any of them could be the wrong title
				res.Accounted = append(res.Accounted, Accounted{Path: a.Path, K: k,
					Reason: "several catalogue games share this set's family, so it is not offered as a version of any"})
				continue
			case runnable && has(a.RBF):
				version(a, rows, false)
				continue
			case runnable:
				res.Accounted = append(res.Accounted, Accounted{Path: a.Path, K: k,
					Reason: "the catalogue runs this game here; this file's core (" + a.RBF + ") is not in _Arcade/cores"})
				continue
			case !has(a.RBF):
				// neither copy runs: the greyed catalogue row says so
				res.Accounted = append(res.Accounted, Accounted{Path: a.Path, K: k,
					Reason: "neither the catalogue's core nor this file's core (" + a.RBF + ") is in _Arcade/cores"})
				continue
			}
			standins[a.Path] = true
		}
		sn := identity(a.Setname)
		if sn == "" {
			sn = identity(strings.TrimSuffix(path.Base(a.Path), path.Ext(a.Path)))
			if sn == "" {
				res.Skipped = append(res.Skipped, Skipped{Path: a.Path, Reason: "no setname", Size: a.Size, Mtime: a.Mtime})
				continue
			}
		}
		a.Setname = sn
		present[sn] = true
		loose = append(loose, a)
	}
	// A clone whose parent is itself on the card files under the parent, so
	// one game makes one row however many sets of it are installed.
	groups := map[string][]Alt{}
	for _, a := range loose {
		key := a.Setname
		if a.Parent != "" && a.Parent != key && present[a.Parent] {
			key = a.Parent
		}
		groups[key] = append(groups[key], a)
	}
	for sn, g := range groups {
		sort.Slice(g, func(i, j int) bool {
			// the parent set itself, then main folders, then short paths
			pi, pj := g[i].Setname == sn, g[j].Setname == sn
			if pi != pj {
				return pi
			}
			ai, aj := underAlternatives(g[i].Path), underAlternatives(g[j].Path)
			if ai != aj {
				return !ai
			}
			if len(g[i].Path) != len(g[j].Path) {
				return len(g[i].Path) < len(g[j].Path)
			}
			return g[i].Path < g[j].Path
		})
		r := rowFromAlt(g[0])
		for _, a := range g {
			r.Standin = r.Standin || standins[a.Path]
		}
		if snap {
			r.Img, r.ImgSlots = sn, []string{SlotLocalSnap}
			if r.RotGroup() == "v" {
				r.ImgW, r.ImgH = 3, 4
			} else {
				r.ImgW, r.ImgH = 4, 3
			}
		}
		res.Rows = append(res.Rows, r)
		for _, a := range g[1:] {
			res.Alts[r.K] = append(res.Alts[r.K], a.Path)
		}
	}
	sort.Slice(res.Rows, func(i, j int) bool { return res.Rows[i].K < res.Rows[j].K })
	return res
}

// owners indexes which catalogue rows a card file belongs to.
type owners struct {
	catalogue []data.Row
	paths     map[string]bool  // the catalogue's own MRA paths
	bySN      map[string][]int // exact release: the rows whose setname it is
	byFamily  map[string][]int // the rows naming it as family root or known clone
}

func catalogueOwners(catalogue []data.Row) owners {
	o := owners{catalogue: catalogue, paths: map[string]bool{}, bySN: map[string][]int{}, byFamily: map[string][]int{}}
	for i := range catalogue {
		r := &catalogue[i]
		if !r.IsArcade() {
			continue
		}
		if r.MRA != "" {
			o.paths[r.MRA] = true
		}
		if id := identity(r.SN); id != "" {
			o.bySN[id] = append(o.bySN[id], i)
		}
		seen := map[string]bool{}
		for _, v := range append([]string{r.Family}, r.FamilySets...) {
			if id := identity(v); id != "" && !seen[id] {
				seen[id] = true
				o.byFamily[id] = append(o.byFamily[id], i)
			}
		}
	}
	return o
}

// of names the catalogue rows a file belongs to. The rows whose own setname
// it is own it outright. Otherwise the rows claiming it through its parent or
// a family own it, but only when they are one release (the same setname in
// several implementations or variants): families in the catalogue span
// different titles (Sprint 1 and 2, River Patrol and Silver Land), and a
// family sibling must never stand in for the wrong game. ambiguous reports
// claimants of different releases; rows then lists them all.
func (o owners) of(a Alt) (rows []int, ambiguous bool) {
	sn, parent := identity(a.Setname), identity(a.Parent)
	if sn != "" && len(o.bySN[sn]) > 0 {
		return o.bySN[sn], false
	}
	seen := map[int]bool{}
	add := func(list []int) {
		for _, i := range list {
			if !seen[i] {
				seen[i] = true
				rows = append(rows, i)
			}
		}
	}
	if sn != "" {
		add(o.byFamily[sn])
	}
	if parent != "" {
		add(o.bySN[parent])
		add(o.byFamily[parent])
	}
	sort.Ints(rows)
	for _, i := range rows {
		if identity(o.catalogue[i].SN) != identity(o.catalogue[rows[0]].SN) {
			return rows, true
		}
	}
	return rows, false
}

// MergeVersions adds the extra versions DiscoverLocal found (catalogue key ->
// card paths) to a family resolver result, each list sorted and without
// duplicates. It returns how many row-to-file offers were new and how many
// distinct files they came from.
func MergeVersions(resolved map[string][]string, extra map[string][]string) (offers, files int) {
	fileSet := map[string]bool{}
	for k, ps := range extra {
		have := map[string]bool{}
		for _, p := range resolved[k] {
			have[p] = true
		}
		merged := append([]string(nil), resolved[k]...)
		for _, p := range ps {
			if !have[p] {
				have[p] = true
				merged = append(merged, p)
				offers++
				fileSet[p] = true
			}
		}
		sort.Strings(merged)
		resolved[k] = merged
	}
	return offers, len(fileSet)
}

// coreOnCard reports whether an rbf name resolves to an arcade core the
// index holds. No index (a card scan that failed, or a caller with none)
// means nothing is on the card.
func coreOnCard(idx *Index, core string) bool {
	if idx == nil || core == "" {
		return false
	}
	_, ok := idx.lookupFor(true, core)
	return ok
}

// isBIOS recognises a system BIOS MRA (MiSTer ships one per platform core,
// e.g. "PGM (Polygame Master) System BIOS") by the word in its name or file
// name; it is not a game to list.
func isBIOS(a Alt) bool {
	for _, s := range []string{a.Name, path.Base(a.Path)} {
		for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
		}) {
			if w == "bios" {
				return true
			}
		}
	}
	return false
}

// AttachedPaths flattens a family resolver result into the set of paths it
// tied to catalogue rows.
func AttachedPaths(resolved map[string][]string) map[string]bool {
	out := map[string]bool{}
	for _, ps := range resolved {
		for _, p := range ps {
			out[p] = true
		}
	}
	return out
}

func underAlternatives(p string) bool {
	return strings.Contains(strings.ToLower(p), "/_alternatives/")
}

// String makes the result readable in the audit tool's output.
func (r LocalResult) String() string {
	return fmt.Sprintf("%d files, %d local rows, %d skipped", r.Files, len(r.Rows), len(r.Skipped))
}

// rowFromAlt builds the row for an MRA the catalogue does not list. Every
// field comes from the header; what the header lacks stays empty so the
// facets file it under "unknown". A local row carries no release dates: the
// file's own stamp says nothing about the game, so date sorts place it last
// and the last-look marker ignores it.
func rowFromAlt(a Alt) data.Row {
	sn := identity(a.Setname)
	title := strings.TrimSpace(a.Name)
	if title == "" {
		title = strings.TrimSuffix(path.Base(a.Path), path.Ext(a.Path))
	}
	r := data.Row{
		Title:        title,
		Base:         "Arcade",
		Src:          data.SrcLocal,
		K:            data.LocalKey(sn),
		SN:           sn,
		Family:       a.Parent,
		Core:         a.RBF,
		MRA:          a.Path,
		Year:         strings.TrimSpace(a.Year),
		Manufacturer: strings.TrimSpace(a.Manufacturer),
		Reg:          strings.TrimSpace(a.Region),
		Rot:          mraRotation(a.Rotation),
		Plr:          players(a.Players),
	}
	if c := strings.TrimSpace(a.Category); c != "" {
		r.Note = "MRA category: " + c
	}
	joy := strings.TrimSpace(a.Joystick)
	if n, ok := buttonCount(a); ok {
		r.Buttons = &n
		if joy != "" {
			r.Ctl = joy + " · " + buttonsWord(n)
		} else {
			r.Ctl = buttonsWord(n)
		}
	} else {
		r.Ctl = joy
	}
	return r
}

// mraRotation maps an MRA <rotation> onto the catalogue's strings, so the
// rotation facet and the rotation chip read a local row like any other.
func mraRotation(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "vertical") && strings.Contains(l, "ccw"):
		return "Vertical (CCW)"
	case strings.Contains(l, "vertical") && strings.Contains(l, "cw"):
		return "Vertical (CW)"
	case strings.Contains(l, "vertical"):
		return "Vertical"
	case strings.Contains(l, "horizontal") && strings.Contains(l, "180"):
		return "Horizontal (180)"
	case strings.Contains(l, "horizontal"):
		return "Horizontal"
	}
	return ""
}

// players keeps the leading number of "<players>2 (alternating)</players>".
func players(s string) string {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i]
}

// buttonCount prefers <num_buttons>; otherwise it counts <buttons names>
// entries that are game buttons rather than start, coin or service inputs.
func buttonCount(a Alt) (int, bool) {
	if a.NumButtons > 0 {
		return a.NumButtons, true
	}
	if a.ButtonNames == nil {
		return 0, false
	}
	n := 0
	for _, name := range a.ButtonNames {
		l := strings.ToLower(name)
		switch {
		case l == "" || l == "-" || l == "n/a" || l == "none":
		case strings.HasPrefix(l, "start"), strings.HasPrefix(l, "coin"),
			l == "pause", l == "service", l == "test", l == "tilt", l == "select":
		default:
			n++
		}
	}
	return n, true
}

func buttonsWord(n int) string {
	if n == 1 {
		return "1 button"
	}
	return strconv.Itoa(n) + " buttons"
}
