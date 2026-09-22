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
	Err                    error
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
// family resolver tied to a catalogue row. A file is accounted for when its
// path is a catalogue MRA or attached, or when its setname or parent is a
// catalogue release, family root or known clone that this card can run: a
// game the catalogue knows is never listed twice.
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
func DiscoverLocal(card, cachePath string, catalogue []data.Row, idx *Index, alts []Alt, attached map[string]bool, snap bool) LocalResult {
	walked, skipped, dirs, err := scanArcadeMRAs(card, cachePath)
	res := LocalResult{Alts: map[string][]string{}, Files: len(walked), Skipped: skipped, SkippedDirs: dirs, Err: err}
	ids := map[string]string{}  // setname, family root or clone -> a catalogue row naming it
	runs := map[string]string{} // ... and one of those rows that runs on this card
	paths := map[string]bool{}
	for i := range catalogue {
		r := &catalogue[i]
		if !r.IsArcade() {
			continue
		}
		if r.MRA != "" {
			paths[r.MRA] = true
		}
		onCard := coreOnCard(idx, r.Core)
		for _, v := range append([]string{r.SN, r.Family}, r.FamilySets...) {
			if id := identity(v); id != "" {
				if ids[id] == "" {
					ids[id] = r.K
				}
				if onCard && runs[id] == "" {
					runs[id] = r.K
				}
			}
		}
	}
	var loose []Alt
	present := map[string]bool{}
	standins := map[string]bool{} // path -> the catalogue lists this game, unrunnably
	for _, a := range append(append([]Alt{}, walked...), alts...) {
		// the catalogue's own files and its games' versions are in the list
		if paths[a.Path] {
			res.OwnFiles++
			continue
		}
		if attached[a.Path] {
			res.VersionFiles++
			continue
		}
		sn := identity(a.Setname)
		if k := firstOf(ids, sn, a.Parent); k != "" {
			if run := firstOf(runs, sn, a.Parent); run != "" {
				// the catalogue's own copy runs here
				res.Accounted = append(res.Accounted, Accounted{Path: a.Path, K: run,
					Reason: "the catalogue lists this game and its own core is on the card; this file's core (" + a.RBF + ") is not offered as a version"})
				continue
			}
			if !coreOnCard(idx, a.RBF) {
				// neither copy runs: the greyed catalogue row says so
				res.Accounted = append(res.Accounted, Accounted{Path: a.Path, K: k,
					Reason: "neither the catalogue's core nor this file's core (" + a.RBF + ") is in _Arcade/cores"})
				continue
			}
			standins[a.Path] = true
		}
		if sn == "" {
			sn = identity(strings.TrimSuffix(path.Base(a.Path), path.Ext(a.Path)))
			if sn == "" {
				res.Skipped = append(res.Skipped, Skipped{Path: a.Path, Reason: "no setname", Size: a.Size, Mtime: a.Mtime})
				continue
			}
		}
		if isBIOS(a) {
			res.Skipped = append(res.Skipped, Skipped{Path: a.Path, Reason: "BIOS, not a game", Size: a.Size, Mtime: a.Mtime})
			continue
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

// firstOf is m's value for the first of ids that has one.
func firstOf(m map[string]string, ids ...string) string {
	for _, id := range ids {
		if v := m[id]; id != "" && v != "" {
			return v
		}
	}
	return ""
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
