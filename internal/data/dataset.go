package data

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
)

// Derived is what Ingest computes once per row so painting and sorting never
// recompute labels or collation keys.
type Derived struct {
	CoreLabel  string // site coreLabel(core)
	TypeLabel  string // "Arcade" or "<base> core"
	SrcShort   string // MiSTer / Jotego / Coin-Op / Meathax / rmCores
	Title      string // ASCII-folded title for the bitmap font
	Ctl        string // ASCII-folded controls
	RotGroup   string // "h", "v" or ""
	Directions string
	Buttons    string // numeric count including zero; empty = unknown
	Year       string // exact original release year; empty = unknown/ambiguous
	Maker      string // the manufacturer group's name (maker.go), ASCII-folded; empty = unknown
	BatchN     int    // rows sharing this core's updated stamp (0 = not a batch stamp)

	titleKey   []elem
	coreKey    []elem
	updatedKey []elem
	dateKey    []elem
	makerKey   []elem
}

// Facets are the distinct filterable values with row counts.
type Facets struct {
	Res        map[string]int // raw resolution, "" = unknown
	Base       map[string]int
	Src        map[string]int
	Rot        map[string]int // "h", "v", ""
	Plr        map[string]int // raw plr strings, "" = unknown
	Genre      map[string]int // raw genre strings, "" = no genre
	Directions map[string]int
	Buttons    map[string]int
	Year       map[string]int
}

// Dataset is one ingested data.json with every derived index.
type Dataset struct {
	Rows    []Row
	Der     []Derived
	Hash    string    // sha256 hex from meta.json, "" when unknown; catalogue only
	Gen     string    // Hash plus a digest of the local rows: what positional scan results index into
	NCat    int       // Rows[:NCat] are catalogue rows, Rows[NCat:] local rows
	Updated time.Time // meta.json updated stamp, zero when unknown
	ByKey   map[string]int
	Sole    map[string]string
	cluster map[string]int
	Facets  Facets

	orderMu sync.Mutex
	orders  map[SortMode][]int // Order's result per mode, sorted on first use
}

// Ingest mirrors the site's ingest(): it rebuilds (never merges) every
// derived map from the rows. Local rows, if any, follow the catalogue rows
// (MergeLocal keeps that order).
func Ingest(rows []Row, hash string, updated time.Time) *Dataset {
	ncat := len(rows)
	for i := range rows {
		if rows[i].IsLocal() {
			ncat = i
			break
		}
	}
	ds := &Dataset{
		Rows:    rows,
		Der:     make([]Derived, len(rows)),
		Hash:    hash,
		Gen:     Generation(hash, rows[ncat:]),
		NCat:    ncat,
		Updated: updated,
		ByKey:   make(map[string]int, len(rows)),
		Sole:    SoleTitles(rows),
		cluster: map[string]int{},
		Facets: Facets{
			Base: map[string]int{}, Src: map[string]int{}, Rot: map[string]int{},
			Plr: map[string]int{}, Genre: map[string]int{},
			Directions: map[string]int{}, Buttons: map[string]int{}, Res: map[string]int{}, Year: map[string]int{},
		},
	}
	for i := range rows {
		r := &rows[i]
		if r.K != "" {
			ds.ByKey[r.K] = i
		}
		if r.Core != "" && r.Updated != "" && r.Updated != r.Date {
			ds.cluster[r.Core+"\x00"+r.Updated]++
		}
		ds.Facets.Base[r.Base]++
		ds.Facets.Src[r.Src]++
		if r.IsArcade() {
			ds.Facets.Rot[r.RotGroup()]++
			ds.Facets.Plr[r.Plr]++
			ds.Facets.Genre[r.Genre]++
			ds.Facets.Res[r.Res]++
		}
	}
	makers := makerLabels(rows)
	for i := range rows {
		r := &rows[i]
		d := &ds.Der[i]
		d.CoreLabel = CoreLabel(r.Core, ds.Sole)
		d.Maker = makers[MakerKey(r.Manufacturer)]
		d.makerKey = Key(d.Maker)
		d.TypeLabel = TypeLabel(r)
		d.SrcShort = SrcShort(r.Src)
		d.Title = ASCII(r.Title)
		d.Ctl = ASCII(r.Ctl)
		d.Directions, d.Buttons = r.ControlFacets()
		d.Year = ReleaseYear(r.Year)
		if r.IsArcade() {
			ds.Facets.Year[d.Year]++
			ds.Facets.Directions[d.Directions]++
			ds.Facets.Buttons[d.Buttons]++
		}
		d.RotGroup = r.RotGroup()
		d.BatchN = ds.ClusterN(i)
		d.titleKey = Key(r.Title)
		d.coreKey = Key(d.CoreLabel)
		d.updatedKey = Key(r.Updated)
		d.dateKey = Key(r.Date)
	}
	return ds
}

// ClusterN mirrors clusterN(d): 0 for rows with no core, no updated date or
// a debut-floored date; otherwise how many rows share that core+updated stamp
// (1 = shipped alone, 2+ = shipped together).
func (ds *Dataset) ClusterN(i int) int {
	r := &ds.Rows[i]
	if r.Core == "" || r.Updated == "" || r.Updated == r.Date {
		return 0
	}
	return ds.cluster[r.Core+"\x00"+r.Updated]
}

// Catalogue is the rows that came from data.json.
func (ds *Dataset) Catalogue() []Row { return ds.Rows[:ds.NCat] }

// MergeLocal appends the local rows to the catalogue rows, dropping any local
// row whose game the catalogue now names (by K, setname, family root or a
// known clone setname). A standin row is kept: the catalogue names its game
// only for cores the card has not got, so dropping it would hide the only
// copy that runs. The catalogue keeps its order; local rows follow sorted by
// K, so the merge is deterministic and Generation is stable.
func MergeLocal(catalogue, local []Row) []Row {
	if len(local) == 0 {
		return catalogue
	}
	known := make(map[string]bool, len(catalogue)*2)
	for i := range catalogue {
		r := &catalogue[i]
		if r.K != "" {
			known[r.K] = true
		}
		if !r.IsArcade() {
			continue
		}
		for _, id := range append([]string{r.SN, r.Family}, r.FamilySets...) {
			if id != "" {
				known[LocalKey(id)] = true
			}
		}
	}
	kept := make([]Row, 0, len(local))
	seen := map[string]bool{}
	for i := range local {
		r := &local[i]
		if !r.IsLocal() || r.K == "" || seen[r.K] || (known[r.K] && !r.Standin) {
			continue
		}
		seen[r.K] = true
		kept = append(kept, *r)
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].K < kept[j].K })
	out := make([]Row, 0, len(catalogue)+len(kept))
	out = append(out, catalogue...)
	return append(out, kept...)
}

// LocalTakeovers maps the K of each local row that a new catalogue now covers
// (by setname, family root or known clone) to the catalogue row's K, so a
// favorite, remembered version or launch record follows the game when the
// catalogue catches up with the card. Coverage on paper is all this can see,
// so callers hand it the rows that are actually giving way: a standin row
// keeps its key until a scan finds the game runnable from the catalogue.
func LocalTakeovers(local, catalogue []Row) map[string]string {
	if len(local) == 0 {
		return nil
	}
	byID := map[string]string{}
	for i := range catalogue {
		r := &catalogue[i]
		if !r.IsArcade() || r.K == "" {
			continue
		}
		// a row's own setname wins over a family membership
		if r.SN != "" {
			byID[LocalKey(r.SN)] = r.K
		}
		for _, id := range append([]string{r.Family}, r.FamilySets...) {
			if id != "" {
				if _, ok := byID[LocalKey(id)]; !ok {
					byID[LocalKey(id)] = r.K
				}
			}
		}
	}
	out := map[string]string{}
	for i := range local {
		r := &local[i]
		if k, ok := byID[r.K]; ok && r.IsLocal() && k != r.K {
			out[r.K] = k
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// LocalDigest summarises a set of local rows: it changes only when a row is
// added, removed, or launches a different file or core.
func LocalDigest(local []Row) string {
	if len(local) == 0 {
		return ""
	}
	parts := make([]string, 0, len(local))
	for i := range local {
		r := &local[i]
		parts = append(parts, r.K+"|"+r.MRA+"|"+r.Core)
	}
	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(h[:8])
}

// Generation is the dataset identity the scanner's positional results are
// checked against: the catalogue hash alone when there are no local rows.
func Generation(hash string, local []Row) string {
	d := LocalDigest(local)
	if d == "" {
		return hash
	}
	return hash + ":" + d
}

// Index returns the row index for a key, -1 when absent.
func (ds *Dataset) Index(k string) int {
	if i, ok := ds.ByKey[k]; ok {
		return i
	}
	return -1
}
