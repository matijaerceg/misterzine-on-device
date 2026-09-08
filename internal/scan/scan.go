// Package scan reads the card: which cores are installed at which build
// date, whether a row's MRA is present, and which alternative MRAs exist for
// a game. It turns that into the per-row status the list shows (current,
// outdated, not found) and the launch picker's alternatives.
package scan

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
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

// Index is the card's core inventory keyed by lowercase core name.
type Index struct {
	Cores map[string]Core
	At    time.Time
}

// ScanCores lists the core folders. Missing folders are fine (an arcade-only
// card has no _Console).
func ScanCores(card string) *Index {
	idx := &Index{Cores: map[string]Core{}, At: time.Now()}
	for _, dir := range coreDirs {
		entries, err := os.ReadDir(filepath.Join(card, dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".rbf") {
				continue
			}
			stem := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
			base, date := stem, ""
			if m := rbfName.FindStringSubmatch(name); m != nil {
				base, date = strings.ToLower(m[1]), m[2]
			}
			rel := path.Join(dir, name)
			cur, ok := idx.Cores[base]
			if !ok || date > cur.Date {
				idx.Cores[base] = Core{Date: date, Path: rel}
			}
			// the full stem too, for core values that carry a date suffix
			if stem != base {
				if cur, ok := idx.Cores[stem]; !ok || date > cur.Date {
					idx.Cores[stem] = Core{Date: date, Path: rel}
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
	if idx == nil || core == "" {
		return Core{}, false
	}
	l := strings.ToLower(core)
	if c, ok := idx.Cores[l]; ok {
		return c, true
	}
	if m := rbfName.FindStringSubmatch(l + ".rbf"); m != nil {
		if c, ok := idx.Cores[m[1]]; ok {
			return c, true
		}
	}
	if c, ok := idx.Cores["arcade-"+l]; ok {
		return c, true
	}
	if strings.HasPrefix(l, "arcade-") {
		if c, ok := idx.Cores[strings.TrimPrefix(l, "arcade-")]; ok {
			return c, true
		}
	}
	return Core{}, false
}

// Status decides a row's card status. Arcade rows need their MRA on the
// card and their core in the index; others just the core. The shipped date
// in the row (updated, ISO) is compared with the rbf date.
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
	c, ok := idx.Lookup(r.Core)
	if !ok {
		return data.StatusNotFound
	}
	if c.Date == "" {
		return data.StatusFoundUndated
	}
	shipped := strings.ReplaceAll(r.Updated, "-", "")
	if len(shipped) != 8 {
		return data.StatusFoundUndated
	}
	if c.Date >= shipped {
		return data.StatusCurrent
	}
	return data.StatusOutdated
}

// Statuses computes every row's status.
func Statuses(card string, idx *Index, rows []data.Row) []data.Status {
	out := make([]data.Status, len(rows))
	for i := range rows {
		out[i] = Status(card, idx, &rows[i])
	}
	return out
}

// Alt is one alternative MRA under _Arcade/_alternatives.
type Alt struct {
	Path    string   `json:"path"` // card-relative
	RBF     string   `json:"rbf"`  // lowercase <rbf> text
	Setname string   `json:"setname"`
	Zips    []string `json:"zips"` // lowercase zip names the rom index 0 references
}

type altDir struct {
	Mtime int64 `json:"mtime"`
	Alts  []Alt `json:"alts"`
}

// ScanAlternatives walks _Arcade/_alternatives, parsing only the header of
// each MRA. A per-directory cache keyed by mtime (cachePath, JSON) makes
// warm runs cheap; pass "" to disable the cache.
func ScanAlternatives(card, cachePath string) []Alt {
	root := filepath.Join(card, "_Arcade", "_alternatives")
	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	cache := map[string]altDir{}
	if cachePath != "" {
		if b, err := os.ReadFile(cachePath); err == nil {
			json.Unmarshal(b, &cache)
		}
	}
	fresh := map[string]altDir{}
	var out []Alt
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		mt := info.ModTime().UnixNano()
		if c, ok := cache[d.Name()]; ok && c.Mtime == mt {
			fresh[d.Name()] = c
			out = append(out, c.Alts...)
			continue
		}
		var alts []Alt
		files, err := os.ReadDir(filepath.Join(root, d.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".mra") {
				continue
			}
			rel := path.Join("_Arcade", "_alternatives", d.Name(), f.Name())
			a, ok := ParseMRAHeader(filepath.Join(card, filepath.FromSlash(rel)))
			if !ok {
				continue
			}
			a.Path = rel
			alts = append(alts, a)
		}
		fresh[d.Name()] = altDir{Mtime: mt, Alts: alts}
		out = append(out, alts...)
	}
	if cachePath != "" {
		if b, err := json.Marshal(fresh); err == nil {
			os.MkdirAll(filepath.Dir(cachePath), 0755)
			os.WriteFile(cachePath, b, 0644)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ParseMRAHeader reads <rbf>, <setname> and the zip list of the first
// <rom index="0"> from an MRA, stopping before the bulky <part> data.
func ParseMRAHeader(p string) (Alt, bool) {
	f, err := os.Open(p)
	if err != nil {
		return Alt{}, false
	}
	defer f.Close()
	return parseMRA(io.LimitReader(f, 64<<10))
}

func parseMRA(r io.Reader) (Alt, bool) {
	var a Alt
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) { return input, nil }
	depth := 0
	want := ""
	for {
		tok, err := dec.Token()
		if err != nil {
			break
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
						return a, true
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
				return a, a.RBF != ""
			}
		}
	}
	return a, a.RBF != ""
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
