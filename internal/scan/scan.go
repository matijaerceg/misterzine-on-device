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
// prefix variant for Meathax-style names.
func (idx *Index) Lookup(core string) (Core, bool) {
	if idx == nil {
		return Core{}, false
	}
	return lookup(idx.Cores, core)
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
	return Core{}, false
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

// Alt is one alternative MRA under an _alternatives folder.
type Alt struct {
	Path    string   `json:"path"` // card-relative
	RBF     string   `json:"rbf"`  // lowercase <rbf> text
	Setname string   `json:"setname"`
	Zips    []string `json:"zips"` // lowercase zip names the rom index 0 references
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

// altCacheVersion is the format written now; version 2 entries (no skip
// records, written only for directories where every MRA parsed) stay valid.
const altCacheVersion = 3

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
	var problems []error
	var original []byte
	var err error
	cache := map[string]altDir{}
	if cachePath != "" {
		original, err = os.ReadFile(cachePath)
		if err == nil {
			if err = json.Unmarshal(original, &cache); err != nil {
				problems = append(problems, fmt.Errorf("alternatives cache: %w", err))
				cache = map[string]altDir{}
			}
		} else if !os.IsNotExist(err) {
			problems = append(problems, err)
		}
	}
	fresh := map[string]altDir{}
	var out []Alt
	var skipped []Skipped
	for _, root := range roots {
		dirs, err := os.ReadDir(filepath.Join(card, filepath.FromSlash(root.rel)))
		if err != nil {
			problems = append(problems, err)
			continue
		}
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			info, err := d.Info()
			if err != nil {
				problems = append(problems, err)
				continue
			}
			key := root.key + d.Name()
			mt := info.ModTime().UnixNano()
			if c, ok := cache[key]; ok && (c.Version == 2 || c.Version == altCacheVersion) && c.Mtime == mt && skippedUnchanged(card, c.Skipped) {
				fresh[key] = c
				out = append(out, c.Alts...)
				skipped = append(skipped, c.Skipped...)
				continue
			}
			var alts []Alt
			var skips []Skipped
			complete := true
			files, err := os.ReadDir(filepath.Join(card, filepath.FromSlash(root.rel), d.Name()))
			if err != nil {
				problems = append(problems, err)
				continue
			}
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".mra") {
					continue
				}
				rel := path.Join(root.rel, d.Name(), f.Name())
				a, s, readErr := parseMRAHeader(filepath.Join(card, filepath.FromSlash(rel)))
				if readErr != nil {
					complete = false
					problems = append(problems, fmt.Errorf("%s: %w", rel, readErr))
					continue
				}
				if s != nil {
					s.Path = rel
					skips = append(skips, *s)
					continue
				}
				a.Path = rel
				alts = append(alts, a)
			}
			if complete {
				fresh[key] = altDir{Version: altCacheVersion, Mtime: mt, Alts: alts, Skipped: skips}
			}
			out = append(out, alts...)
			skipped = append(skipped, skips...)
		}
	}
	if cachePath != "" {
		b, err := json.Marshal(fresh)
		if err == nil && !bytes.Equal(b, original) {
			err = store.WriteAtomic(cachePath, b)
		}
		if err != nil {
			problems = append(problems, fmt.Errorf("alternatives cache save: %w", err))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, skipped, errors.Join(problems...)
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

// ParseMRAHeader reads <rbf>, <setname> and the zip list of the first
// <rom index="0"> from an MRA, stopping before the bulky <part> data.
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

func parseMRAResult(r io.Reader) (Alt, bool, error) {
	var a Alt
	dec := xml.NewDecoder(&commentStripper{br: bufio.NewReader(r)})
	dec.Strict = false
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) { return input, nil }
	depth := 0
	want := ""
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				err = errNoXML
			}
			return a, false, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			name := strings.ToLower(t.Name.Local)
			switch name {
			case "rbf", "setname":
				want = name
			case "rom":
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
					if a.RBF != "" && a.Setname != "" {
						return a, true, nil
					}
				}
			}
		case xml.CharData:
			if want != "" {
				v := strings.TrimSpace(string(t))
				if v != "" {
					if want == "rbf" {
						a.RBF = strings.ToLower(v)
					} else {
						a.Setname = v
					}
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

// Alternatives lists the alternative MRAs for a row: same rbf, and a rom zip
// list that references the row's setname.
func Alternatives(alts []Alt, r *data.Row) []string {
	if r == nil || !r.IsArcade() || r.Core == "" {
		return nil
	}
	core := strings.ToLower(r.Core)
	if m := rbfName.FindStringSubmatch(core + ".rbf"); m != nil {
		core = m[1]
	}
	sn := strings.ToLower(r.SN)
	var out []string
	for _, a := range alts {
		rbf := a.RBF
		if m := rbfName.FindStringSubmatch(rbf + ".rbf"); m != nil {
			rbf = m[1]
		}
		if rbf != core && rbf != "arcade-"+core && "arcade-"+rbf != core {
			continue
		}
		if sn == "" {
			continue
		}
		hit := strings.EqualFold(a.Setname, sn)
		for _, z := range a.Zips {
			if z == sn+".zip" {
				hit = true
			}
		}
		if hit {
			out = append(out, a.Path)
		}
	}
	return out
}
