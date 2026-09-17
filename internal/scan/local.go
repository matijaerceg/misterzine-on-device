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
// image service rather than the catalogue.
const SlotLocalSnap = "lsnap"

// LocalResult is what DiscoverLocal hands the host to merge.
type LocalResult struct {
	Rows    []data.Row          // one per unmatched setname, sorted by K
	Alts    map[string][]string // local K -> further copies of the same set
	Files   int                 // MRAs the walk read, matched or not
	Skipped []Skipped
	Err     error
}

// skipDir names the folders the walk never enters: cores, anything an
// organiser wrote (arcade_organizer's _Organized tree duplicates every MRA
// many times over), the _alternatives folders ScanAlternatives already
// reads, and hidden folders.
func skipDir(name string) bool {
	l := strings.ToLower(name)
	return l == "cores" || l == "_alternatives" || strings.Contains(l, "organized") || strings.HasPrefix(l, ".")
}

// ScanArcadeMRAs reads the header of every MRA under _Arcade outside the
// folders skipDir names, symlinks excluded, through the same per-directory
// cache scheme as the alternatives scan (cachePath, "" to disable).
func ScanArcadeMRAs(card, cachePath string) ([]Alt, []Skipped, error) {
	dc := openDirCache(cachePath)
	var out []Alt
	var skipped []Skipped
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
		if depth >= localMaxDepth {
			return true
		}
		for _, e := range entries {
			// e.IsDir() is false for a symlink whatever it points at.
			if !e.IsDir() || skipDir(e.Name()) {
				continue
			}
			if !walk(path.Join(rel, e.Name()), depth+1) {
				return false
			}
		}
		return true
	}
	walk("_Arcade", 0)
	dc.save("local scan cache")
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, skipped, errors.Join(dc.problems...)
}

// DiscoverLocal turns the MRAs the catalogue does not account for into rows.
// alts are the alternatives already scanned; attached holds every path the
// family resolver tied to a catalogue row. A file is accounted for when its
// path is a catalogue MRA or attached, or when its setname or parent is a
// catalogue release, family root or known clone, whatever core it names:
// a game the catalogue knows is never listed twice. The rest are grouped by
// setname; the copy outside any _alternatives folder with the shortest path
// becomes the row and the others its alternatives. Set snap when the image
// service is available: such rows then ask it for a shot by setname.
func DiscoverLocal(card, cachePath string, catalogue []data.Row, alts []Alt, attached map[string]bool, snap bool) LocalResult {
	walked, skipped, err := ScanArcadeMRAs(card, cachePath)
	res := LocalResult{Alts: map[string][]string{}, Files: len(walked), Skipped: skipped, Err: err}
	ids := map[string]bool{}
	paths := map[string]bool{}
	for i := range catalogue {
		r := &catalogue[i]
		if !r.IsArcade() {
			continue
		}
		if r.MRA != "" {
			paths[r.MRA] = true
		}
		for _, v := range append([]string{r.SN, r.Family}, r.FamilySets...) {
			if id := identity(v); id != "" {
				ids[id] = true
			}
		}
	}
	var loose []Alt
	present := map[string]bool{}
	for _, a := range append(append([]Alt{}, walked...), alts...) {
		if paths[a.Path] || attached[a.Path] {
			continue
		}
		sn := identity(a.Setname)
		if (sn != "" && ids[sn]) || (a.Parent != "" && ids[a.Parent]) {
			continue
		}
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
