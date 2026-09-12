package data

import (
	"sync"
	"time"
)

// Derived is what Ingest computes once per row so painting and sorting never
// recompute labels or collation keys.
type Derived struct {
	CoreLabel  string // site coreLabel(core)
	TypeLabel  string // "Arcade" or "<base> core"
	SrcShort   string // MiSTer / Jotego / Coin-Op / Meathax
	Title      string // ASCII-folded title for the bitmap font
	Ctl        string // ASCII-folded controls
	RotGroup   string // "h", "v" or ""
	Directions string
	Buttons    string // numeric count including zero; empty = unknown
	Year       string // exact original release year; empty = unknown/ambiguous
	BatchN     int    // rows sharing this core's updated stamp (0 = not a batch stamp)

	titleKey   []elem
	coreKey    []elem
	updatedKey []elem
	dateKey    []elem
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
	Hash    string    // sha256 hex from meta.json, "" when unknown
	Updated time.Time // meta.json updated stamp, zero when unknown
	ByKey   map[string]int
	Sole    map[string]string
	cluster map[string]int
	Facets  Facets

	orderMu sync.Mutex
	orders  map[SortMode][]int // Order's result per mode, sorted on first use
}

// Ingest mirrors the site's ingest(): it rebuilds (never merges) every
// derived map from the rows.
func Ingest(rows []Row, hash string, updated time.Time) *Dataset {
	ds := &Dataset{
		Rows:    rows,
		Der:     make([]Derived, len(rows)),
		Hash:    hash,
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
	for i := range rows {
		r := &rows[i]
		d := &ds.Der[i]
		d.CoreLabel = CoreLabel(r.Core, ds.Sole)
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

// Index returns the row index for a key, -1 when absent.
func (ds *Dataset) Index(k string) int {
	if i, ok := ds.ByKey[k]; ok {
		return i
	}
	return -1
}
