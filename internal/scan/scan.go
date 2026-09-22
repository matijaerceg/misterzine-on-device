// Package scan reads the card: which cores are installed at which build
// date, whether a row's MRA is present, and which alternative MRAs exist for
// a game. It turns that into the per-row status the list shows (current,
// outdated, not found) and the launch picker's alternatives.
package scan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// coreDirs are where MiSTer keeps rbf files, card-relative.
var coreDirs = []string{"_Arcade/cores", "_Console", "_Computer", "_Other", "_Utility"}

// rbfName matches "<Core>_YYYYMMDD[_anything].rbf"; undated rbfs (Jotego
// ships e.g. jt1942.rbf) simply do not match and count as undated.
var rbfName = regexp.MustCompile(`^(?i)(.+?)_(\d{8})(?:_[^/]*)?\.rbf$`)

// Core is one installed core: its newest build date and file.
type Core struct {
	Date string `json:"date"` // YYYYMMDD, "" when undated
	Path string `json:"path"` // card-relative
}

// Index is the card's core inventory keyed by lowercase core name. Cores is
// the whole card (what the launcher resolves names against); arcade and
// system split it by folder, because an arcade MRA only ever loads from
// _Arcade/cores while a console/computer row's rbf lives elsewhere, and the
// two can share a name (Astrocade ships both).
type Index struct {
	Cores  map[string]Core
	arcade map[string]Core
	system map[string]Core
	At     time.Time
	Err    error // unreadable card/core folders; absent optional folders are fine
	// FeedAt is when the feed was generated. An undated rbf installed after
	// that cannot be behind the feed's hash, so a mismatch reads as current
	// rather than older. Zero disables the guard.
	FeedAt time.Time
	// HashCache persists the md5s the app computes itself; "" keeps them in
	// memory only.
	HashCache string
	hashes    hashes
}

// ScanCores lists the core folders. Missing folders are fine (an arcade-only
// card has no _Console).
func ScanCores(card string) *Index {
	idx := &Index{Cores: map[string]Core{}, arcade: map[string]Core{}, system: map[string]Core{}, At: time.Now()}
	if _, err := os.ReadDir(card); err != nil {
		idx.Err = err
		return idx
	}
	for _, dir := range coreDirs {
		entries, err := os.ReadDir(filepath.Join(card, dir))
		if err != nil {
			if !os.IsNotExist(err) {
				idx.Err = errors.Join(idx.Err, err)
			}
			continue
		}
		kind := idx.system
		if dir == "_Arcade/cores" {
			kind = idx.arcade
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".rbf") {
				continue
			}
			stem := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
			base, date := stem, ""
			if m := rbfName.FindStringSubmatch(name); m != nil {
				base, date = strings.ToLower(m[1]), m[2]
			}
			rel := path.Join(dir, name)
			for _, m := range []map[string]Core{idx.Cores, kind} {
				if cur, ok := m[base]; !ok || date > cur.Date {
					m[base] = Core{Date: date, Path: rel}
				}
				// the full stem too, for core values that carry a date suffix
				if stem != base {
					if cur, ok := m[stem]; !ok || date > cur.Date {
						m[stem] = Core{Date: date, Path: rel}
					}
				}
			}
		}
	}
	return idx
}

// Lookup finds a row's core in the index, trying the exact lowercase name,
// then the name with its own _YYYYMMDD suffix stripped, then an "arcade-"
// prefix variant for Meathax-style names, and last the rule MiSTer's own
// MRA loader applies: any rbf whose name starts with the core followed by
// "_" (Coin-Op ships blkheart_mister_20260909.rbf for <rbf>blkheart</rbf>).
func (idx *Index) Lookup(core string) (Core, bool) {
	if idx == nil {
		return Core{}, false
	}
	return lookup(idx.Cores, core)
}

// HasArcadeCore reports whether an MRA naming this core would find it in
// _Arcade/cores. A nil index has nothing.
func (idx *Index) HasArcadeCore(core string) bool {
	return coreOnCard(idx, core)
}

// lookupFor is Lookup restricted to the folders a row's kind loads from.
func (idx *Index) lookupFor(arcade bool, core string) (Core, bool) {
	if idx == nil {
		return Core{}, false
	}
	if arcade {
		return lookup(idx.arcade, core)
	}
	return lookup(idx.system, core)
}

func lookup(cores map[string]Core, core string) (Core, bool) {
	if core == "" {
		return Core{}, false
	}
	l := strings.ToLower(core)
	if c, ok := cores[l]; ok {
		return c, true
	}
	if m := rbfName.FindStringSubmatch(l + ".rbf"); m != nil {
		if c, ok := cores[m[1]]; ok {
			return c, true
		}
	}
	if c, ok := cores["arcade-"+l]; ok {
		return c, true
	}
	if strings.HasPrefix(l, "arcade-") {
		if c, ok := cores[strings.TrimPrefix(l, "arcade-")]; ok {
			return c, true
		}
	}
	return lookupPrefix(cores, l)
}

// lookupPrefix is MiSTer's get_rbf rule: an rbf matches a core name when
// its filename starts with the name followed by "_" (the exact name and
// the "." case are the early returns above), also with an "Arcade-" prefix,
// and the greatest matching filename wins when several do. A dated file is
// indexed under both its base and its full stem, both at the same path, so
// for one core either key gives the same answer; between two cores that
// share a prefix (foo_a, foo_b) the greatest wins, as on MiSTer.
func lookupPrefix(cores map[string]Core, l string) (Core, bool) {
	best, found := "", false
	for _, p := range []string{l + "_", "arcade-" + l + "_"} {
		for k := range cores {
			if strings.HasPrefix(k, p) && (!found || k > best) {
				best, found = k, true
			}
		}
	}
	if !found {
		return Core{}, false
	}
	return cores[best], true
}

// Status decides a row's card status. Arcade rows need their MRA on the
// card and their core in the index; others just the core. The shipped
// build's md5 (bh) decides when the card's file can be hashed; otherwise the
// shipped rbf's build date (bd, falling back to updated) is compared with
// the rbf filename date.
func Status(card string, idx *Index, r *data.Row) data.Status {
	if r.IsArcade() {
		if r.MRA == "" {
			return data.StatusUnknown
		}
		if _, err := os.Stat(filepath.Join(card, filepath.FromSlash(r.MRA))); err != nil {
			return data.StatusNotFound
		}
	}
	if r.Core == "" {
		if r.IsArcade() {
			return data.StatusFoundUndated // MRA present, core unknown to the data
		}
		return data.StatusUnknown
	}
	c, ok := idx.lookupFor(r.IsArcade(), r.Core)
	if !ok {
		return data.StatusNotFound
	}
	// An md5 match is current whatever the dates say. A mismatch on an
	// undated rbf is the evidence update_all itself acts on, so it reads as
	// likely older; a dated rbf still gets its answer from the dates.
	if r.BH != "" {
		if h, mtime := idx.hashes.of(card, c.Path, idx.HashCache); h != "" {
			if h == r.BH {
				return data.StatusCurrent
			}
			if c.Date == "" {
				if !idx.FeedAt.IsZero() && mtime.After(idx.FeedAt) {
					return data.StatusCurrent // installed after the feed was built: newer, not older
				}
				return data.StatusLikelyOutdated
			}
		}
	}
	if c.Date == "" {
		return data.StatusFoundUndated
	}
	// bd is the shipped rbf's own date; updated (the fallback for feeds
	// without bd) also moves on MRA-only fixes and debut-commit rollovers,
	// which read as "older build" when nothing newer ships.
	shipped := strings.ReplaceAll(r.BD, "-", "")
	if len(shipped) != 8 {
		shipped = strings.ReplaceAll(r.Updated, "-", "")
	}
	if len(shipped) != 8 {
		return data.StatusFoundUndated
	}
	if c.Date >= shipped {
		return data.StatusCurrent
	}
	return data.StatusOutdated
}

// Statuses computes every row's status and persists any md5s it had to
// compute itself.
func Statuses(card string, idx *Index, rows []data.Row) []data.Status {
	out := make([]data.Status, len(rows))
	for i := range rows {
		out[i] = Status(card, idx, &rows[i])
	}
	idx.hashes.save()
	return out
}

// Alt is one MRA's header: an alternative under an _alternatives folder, or
// any MRA the local walk found. The descriptive fields are what the header
// says, verbatim; they only matter for a row the catalogue cannot supply.
type Alt struct {
	Path    string   `json:"path"` // card-relative
	Size    int64    `json:"size,omitempty"`
	Mtime   int64    `json:"mtime,omitempty"`
	RBF     string   `json:"rbf"` // lowercase <rbf> text
	Setname string   `json:"setname"`
	Parent  string   `json:"parent,omitempty"`
	Zips    []string `json:"zips"` // lowercase zip names the rom index 0 references

	Name         string   `json:"name,omitempty"`
	Year         string   `json:"year,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Category     string   `json:"category,omitempty"`
	Rotation     string   `json:"rotation,omitempty"` // e.g. "vertical (cw)"
	Region       string   `json:"region,omitempty"`
	Players      string   `json:"players,omitempty"`
	Joystick     string   `json:"joystick,omitempty"`
	NumButtons   int      `json:"num_buttons,omitempty"` // <num_buttons>; 0 = absent
	ButtonNames  []string `json:"button_names,omitempty"`
	Homebrew     bool     `json:"homebrew,omitempty"`
	Bootleg      bool     `json:"bootleg,omitempty"`
}

// Skipped is an MRA under _Arcade/_alternatives whose header could not be
// read: no XML in it, cut short, or without an <rbf>. MiSTer could not load
// it either, so it is left out of the picker; the scan itself still counts
// as complete. Size and Mtime let a warm run notice an in-place rewrite of
// the file, which leaves the directory's mtime alone.
type Skipped struct {
	Path   string `json:"path"` // card-relative
	Reason string `json:"reason"`
	Size   int64  `json:"size"`
	Mtime  int64  `json:"mtime"`
	Fresh  bool   `json:"-"` // parsed on this run rather than read back from the cache
}

type altDir struct {
	Version int       `json:"version"`
	Mtime   int64     `json:"mtime"`
	Alts    []Alt     `json:"alts"`
	Skipped []Skipped `json:"skipped,omitempty"`
}

// Version 4 added parent metadata, version 5 the descriptive header fields;
// older entries must be reparsed.
const altCacheVersion = 5

// ScanAlternatives walks every _alternatives folder (altRoots), parsing
// only the header of each MRA. A per-directory cache keyed by mtime
// (cachePath, JSON) makes warm runs cheap; pass "" to disable the cache.
func ScanAlternatives(card, cachePath string) []Alt {
	alts, _, _ := ScanAlternativesWithError(card, cachePath)
	return alts
}

// ScanAlternativesWithError returns usable entries even if some reads or the
// optional cache write fail. MRAs that open but hold no usable header are
// skipped, not failures: they are listed, cached with their directory, and
// re-read only when the file itself changes. The error covers directory
// reads, opens and reads that failed, and the cache; a directory with such a
// failure is never cached as complete.
func ScanAlternativesWithError(card, cachePath string) ([]Alt, []Skipped, error) {
	roots := altRoots(card)
	if len(roots) == 0 {
		return nil, nil, nil
	}
	dc := openDirCache(cachePath)
	var out []Alt
	var skipped []Skipped
	for _, root := range roots {
		dirs, err := os.ReadDir(filepath.Join(card, filepath.FromSlash(root.rel)))
		if err != nil {
			dc.problems = append(dc.problems, err)
			continue
		}
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			info, err := d.Info()
			if err != nil {
				dc.problems = append(dc.problems, err)
				continue
			}
			files, err := os.ReadDir(filepath.Join(card, filepath.FromSlash(root.rel), d.Name()))
			if err != nil {
				dc.problems = append(dc.problems, err)
				continue
			}
			alts, skips := dc.dir(card, root.key+d.Name(), path.Join(root.rel, d.Name()), info.ModTime().UnixNano(), files)
			out = append(out, alts...)
			skipped = append(skipped, skips...)
		}
	}
	dc.save("alternatives cache")
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, skipped, errors.Join(dc.problems...)
}

// dirCache is the per-directory header cache both walks share: an entry is
// reused while the directory's mtime and every listed file's size and mtime
// hold; otherwise the directory's MRAs are parsed again.
type dirCache struct {
	path     string
	original []byte
	cache    map[string]altDir
	fresh    map[string]altDir
	problems []error
}

func openDirCache(cachePath string) *dirCache {
	dc := &dirCache{path: cachePath, cache: map[string]altDir{}, fresh: map[string]altDir{}}
	if cachePath == "" {
		return dc
	}
	original, err := os.ReadFile(cachePath)
	if err == nil {
		dc.original = original
		if err = json.Unmarshal(original, &dc.cache); err != nil {
			dc.problems = append(dc.problems, fmt.Errorf("scan cache: %w", err))
			dc.cache = map[string]altDir{}
		}
	} else if !os.IsNotExist(err) {
		dc.problems = append(dc.problems, err)
	}
	return dc
}

// dir returns the headers of the MRAs directly inside one directory (rel,
// card-relative; files is its listing), from the cache when it still holds.
// A directory with a failed read is never cached as complete.
func (dc *dirCache) dir(card, key, rel string, mtime int64, files []os.DirEntry) ([]Alt, []Skipped) {
	if c, ok := dc.cache[key]; ok && c.Version == altCacheVersion && c.Mtime == mtime && alternativesUnchanged(card, c.Alts) && skippedUnchanged(card, c.Skipped) {
		dc.fresh[key] = c
		return c.Alts, c.Skipped
	}
	var alts []Alt
	var skips []Skipped
	complete := true
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".mra") {
			continue
		}
		p := path.Join(rel, f.Name())
		a, s, readErr := parseMRAHeader(filepath.Join(card, filepath.FromSlash(p)))
		if readErr != nil {
			complete = false
			dc.problems = append(dc.problems, fmt.Errorf("%s: %w", p, readErr))
			continue
		}
		if s != nil {
			s.Path = p
			skips = append(skips, *s)
			continue
		}
		a.Path = p
		alts = append(alts, a)
	}
	if complete {
		dc.fresh[key] = altDir{Version: altCacheVersion, Mtime: mtime, Alts: alts, Skipped: skips}
	}
	return alts, skips
}

// save writes the entries this run confirmed, only when they differ from
// what was read; stale directories drop out.
func (dc *dirCache) save(what string) {
	if dc.path == "" {
		return
	}
	b, err := json.Marshal(dc.fresh)
	if err == nil && !bytes.Equal(b, dc.original) {
		err = store.WriteAtomic(dc.path, b)
	}
	if err != nil {
		dc.problems = append(dc.problems, fmt.Errorf("%s save: %w", what, err))
	}
}

// altRoot is one _alternatives folder: the top one, which the Distribution
// and Jotego fill, or the one inside an opt-in database's own folder
// (_Arcade/_MeatCores/_alternatives, _Arcade/_rmCores/_alternatives), so a
// database that keeps to that layout is covered without a list of names.
type altRoot struct {
	rel string // card-relative, forward slashes
	key string // prefix for its game folders' cache keys: "" for the top one
}

// altRoots lists the _alternatives folders present on the card: the top one
// first, then one per _Arcade/_<database> folder that has its own.
func altRoots(card string) []altRoot {
	var roots []altRoot
	top := "_Arcade/_alternatives"
	if st, err := os.Stat(filepath.Join(card, filepath.FromSlash(top))); err == nil && st.IsDir() {
		roots = append(roots, altRoot{rel: top})
	}
	entries, err := os.ReadDir(filepath.Join(card, "_Arcade"))
	if err != nil {
		return roots
	}
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() || !strings.HasPrefix(n, "_") || n == "_alternatives" {
			continue
		}
		rel := path.Join("_Arcade", n, "_alternatives")
		if st, err := os.Stat(filepath.Join(card, filepath.FromSlash(rel))); err == nil && st.IsDir() {
			roots = append(roots, altRoot{rel: rel, key: n + "/"})
		}
	}
	return roots
}

// Directory mtimes do not change when an existing MRA is rewritten.
func alternativesUnchanged(card string, alts []Alt) bool {
	for _, a := range alts {
		st, err := os.Stat(filepath.Join(card, filepath.FromSlash(a.Path)))
		if err != nil || st.Size() != a.Size || st.ModTime().UnixNano() != a.Mtime {
			return false
		}
	}
	return true
}

// skippedUnchanged reports whether every skipped file still has the size and
// mtime recorded with the cache entry, so a file rewritten in place (the
// directory's mtime does not move) is read again.
func skippedUnchanged(card string, skips []Skipped) bool {
	for _, s := range skips {
		st, err := os.Stat(filepath.Join(card, filepath.FromSlash(s.Path)))
		if err != nil || st.Size() != s.Size || st.ModTime().UnixNano() != s.Mtime {
			return false
		}
	}
	return true
}

// ParseMRAHeader reads an MRA's header: <rbf>, <setname>, the zip list of
// the first <rom index="0"> and the descriptive fields, skipping over the
// bulky <part> data. Only the first 64 KiB are read; a header that continues
// past that (ROM patches before <buttons>) simply loses its tail.
func ParseMRAHeader(p string) (Alt, bool) {
	a, s, err := parseMRAHeader(p)
	return a, s == nil && err == nil
}

// parseMRAHeader returns the header, or a Skipped (Fresh, without Path)
// describing why the file holds none. The error is for opening or reading
// the file, never for its content.
func parseMRAHeader(p string) (Alt, *Skipped, error) {
	f, err := os.Open(p)
	if err != nil {
		return Alt{}, nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Alt{}, nil, err
	}
	a, ok, err := parseMRAResult(io.LimitReader(f, 64<<10))
	if ok {
		a.Size, a.Mtime = st.Size(), st.ModTime().UnixNano()
		return a, nil, nil
	}
	var syntax *xml.SyntaxError
	switch {
	case err == nil:
		err = errors.New("no <rbf> element")
	case errors.Is(err, errNoXML), errors.As(err, &syntax):
	default:
		return Alt{}, nil, err // the read itself failed
	}
	return Alt{}, &Skipped{Reason: err.Error(), Size: st.Size(), Mtime: st.ModTime().UnixNano(), Fresh: true}, nil
}

func parseMRA(r io.Reader) (Alt, bool) {
	a, ok, _ := parseMRAResult(r)
	return a, ok
}

// errNoXML is what an empty file, or one without a single XML element,
// yields: the decoder reaches the end with nothing open.
var errNoXML = errors.New("no XML content")

// headerText names the root's children whose text is kept.
var headerText = map[string]bool{
	"rbf": true, "setname": true, "parent": true, "name": true, "year": true,
	"manufacturer": true, "category": true, "rotation": true, "region": true,
	"players": true, "joystick": true, "num_buttons": true, "homebrew": true, "bootleg": true,
}

// parseMRAResult reads the whole header. <rom> and <switches> subtrees are
// skipped rather than tokenised. Once a <rom> has been seen the header is
// complete enough: a read that fails after it (the 64 KiB limit landing in
// ROM data) still yields the header, while a file cut short before any
// <rom> is reported as unreadable, as before.
func parseMRAResult(r io.Reader) (Alt, bool, error) {
	var a Alt
	dec := xml.NewDecoder(&commentStripper{br: bufio.NewReader(r)})
	dec.Strict = false
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) { return input, nil }
	depth := 0
	want := ""
	sawROM := false
	for {
		tok, err := dec.Token()
		if err != nil {
			if sawROM && a.RBF != "" {
				return a, true, nil
			}
			if err == io.EOF {
				err = errNoXML
			}
			return a, false, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			name := strings.ToLower(t.Name.Local)
			if depth != 2 {
				continue
			}
			switch {
			case headerText[name]:
				want = name
			case name == "buttons":
				for _, at := range t.Attr {
					if strings.ToLower(at.Name.Local) == "names" {
						a.ButtonNames = splitNames(at.Value)
					}
				}
			case name == "rom" || name == "switches":
				if name == "rom" {
					idx, zip := "", ""
					for _, at := range t.Attr {
						switch strings.ToLower(at.Name.Local) {
						case "index":
							idx = at.Value
						case "zip":
							zip = at.Value
						}
					}
					if idx == "0" && a.Zips == nil {
						for _, z := range strings.Split(zip, "|") {
							z = strings.ToLower(strings.TrimSpace(z))
							if z != "" {
								a.Zips = append(a.Zips, z)
							}
						}
					}
					sawROM = true
				}
				if err := dec.Skip(); err != nil {
					if a.RBF != "" {
						return a, true, nil
					}
					return a, false, err
				}
				depth--
			}
		case xml.CharData:
			if want != "" {
				if v := strings.TrimSpace(string(t)); v != "" {
					a.setText(want, v)
				}
			}
		case xml.EndElement:
			depth--
			want = ""
			if depth <= 0 {
				return a, a.RBF != "", nil
			}
		}
	}
}

// setText stores one header element's text. Elements MiSTer reads
// case-insensitively are kept as written except where matching needs a
// canonical form.
func (a *Alt) setText(name, v string) {
	switch name {
	case "rbf":
		a.RBF = strings.ToLower(v)
	case "parent":
		a.Parent = identity(v)
	case "setname":
		a.Setname = v
	case "name":
		a.Name = v
	case "year":
		a.Year = v
	case "manufacturer":
		a.Manufacturer = v
	case "category":
		a.Category = v
	case "rotation":
		a.Rotation = v
	case "region":
		a.Region = v
	case "players":
		a.Players = v
	case "joystick":
		a.Joystick = v
	case "num_buttons":
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			a.NumButtons = n
		}
	case "homebrew":
		a.Homebrew = yes(v)
	case "bootleg":
		a.Bootleg = yes(v)
	}
}

func yes(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes", "true", "1":
		return true
	}
	return false
}

// splitNames splits a <buttons names="..."> list, keeping empty slots so
// the positions stay meaningful to a caller that wants them.
func splitNames(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// commentStripper drops <!-- ... --> spans before the XML decoder sees them.
// Go's decoder rejects "--" inside a comment whatever its Strict setting,
// and some MRA sets (Seibu SPI) write comments that way; MiSTer loads them
// fine. Newlines inside a comment pass through so line numbers in a later
// syntax error still point at the file.
type commentStripper struct {
	br *bufio.Reader
	in bool
}

func (c *commentStripper) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		b, err := c.br.ReadByte()
		if err != nil {
			return n, err
		}
		if c.in {
			if b == '\n' {
				p[n] = b
				n++
			} else if b == '-' {
				if peek, _ := c.br.Peek(2); string(peek) == "->" {
					c.br.Discard(2)
					c.in = false
				}
			}
			continue
		}
		if b == '<' {
			if peek, _ := c.br.Peek(3); string(peek) == "!--" {
				c.br.Discard(3)
				c.in = true
				continue
			}
		}
		p[n] = b
		n++
	}
	return n, nil
}

// coreStem is a core name as it compares: lowercase, without its own
// _YYYYMMDD suffix and without an "arcade-" prefix.
func coreStem(name string) string {
	l := strings.ToLower(name)
	if m := rbfName.FindStringSubmatch(l + ".rbf"); m != nil {
		l = m[1]
	}
	return strings.TrimPrefix(l, "arcade-")
}

// sameCore reports whether two core names load the same rbf under
// MiSTer's rules: equal, or one is the other followed by "_" and more (the
// row may say blkheart_mister, the file it was read from, while every
// alternative MRA still says blkheart, or the other way round on a cached
// feed), with or without an "arcade-" prefix and their own date suffixes.
func sameCore(a, b string) bool {
	a, b = coreStem(a), coreStem(b)
	return a == b || strings.HasPrefix(a, b+"_") || strings.HasPrefix(b, a+"_")
}

// AlternativeIndex groups explicit identities by compatible FPGA core.
// Build once in the background, rather than normalizing every file per row.
type AlternativeIndex map[string]map[string][]string

func IndexAlternatives(alts []Alt) AlternativeIndex {
	index := AlternativeIndex{}
	for _, a := range alts {
		core := coreStem(a.RBF)
		if core == "" {
			continue
		}
		ids := []string{a.Setname, a.Parent}
		for _, z := range a.Zips {
			z = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(z, "\\", "/")))
			if strings.HasSuffix(z, ".zip") {
				ids = append(ids, strings.TrimSuffix(path.Base(z), ".zip"))
			}
		}
		if index[core] == nil {
			index[core] = map[string][]string{}
		}
		for _, value := range ids {
			if id := identity(value); id != "" {
				index[core][id] = append(index[core][id], a.Path)
			}
		}
	}
	return index
}

// ForRow matches game identities, never arbitrary shared ROM dependencies.
func (index AlternativeIndex) ForRow(r *data.Row) []string {
	if r == nil || !r.IsArcade() || r.Core == "" {
		return nil
	}
	values := []string{r.SN, r.Family}
	if identity(r.Family) != "" {
		values = append(values, r.FamilySets...)
	}
	ids := map[string]bool{}
	for _, value := range values {
		if id := identity(value); id != "" {
			ids[id] = true
		}
	}
	seen := map[string]bool{}
	var out []string
	core := coreStem(r.Core)
	for candidate, byID := range index {
		// Both sides are already normalized, including dated core suffixes.
		if core != candidate && !strings.HasPrefix(core, candidate+"_") && !strings.HasPrefix(candidate, core+"_") {
			continue
		}
		for id := range ids {
			for _, p := range byID[id] {
				if !seen[p] && p != r.MRA {
					seen[p] = true
					out = append(out, p)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// Alternatives is the convenience matcher for a single row.
func Alternatives(alts []Alt, r *data.Row) []string { return IndexAlternatives(alts).ForRow(r) }

// identity accepts setnames, not paths or malformed parent text.
func identity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return ""
		}
	}
	return value
}
