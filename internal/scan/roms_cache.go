package scan

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// The sweep's verdicts outlive the run in cache/roms.json: each MRA's
// result with the size and time of the MRA and of every archive its parts
// name. The next sweep reuses a verdict whose files are all as they were,
// and reads only the MRAs and zips that changed, so a card that saw no
// update is swept in the time its files take to stat rather than the
// minute their parsing takes on a DE10.
//
// A verdict that depends on more than the files is not reused: one made
// under a mame folder at the card's root, or for an MRA wanting Jotego's
// beta key, whose answer follows the catalogue's word on the core.
const romCacheVersion = 1

type romCacheFile struct {
	V        int                      `json:"v"`
	Root     string                   `json:"root"`      // where Main looks for the zips (arcadeROMRoot)
	RootMame bool                     `json:"root_mame"` // a mame folder at the card's root: nothing loads
	MRAs     map[string]romCacheEntry `json:"mras"`
}

type romCacheEntry struct {
	Size  int64      `json:"size"`
	MTime int64      `json:"mtime"` // unix nanoseconds
	Zips  []zipStamp `json:"zips,omitempty"`
	Text  string     `json:"text,omitempty"`
	Block bool       `json:"block,omitempty"`
	About string     `json:"about,omitempty"` // ROMResult.Zips: the archives the text is about
	// Key marks a verdict that asked the core's beta status for a missing
	// jtbeta.zip: the status can change between runs, so it is checked again.
	Key bool `json:"key,omitempty"`
}

// zipStamp is one archive as the verdict saw it: Size -1 when it was not
// on the card, since its appearing changes the answer too.
type zipStamp struct {
	Path  string `json:"p"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime,omitempty"`
}

func (e romCacheEntry) result() ROMResult {
	return ROMResult{Text: e.Text, Block: e.Block, Zips: e.About}
}

// loadCache reads the file once per ROMCheck; a file for another ROM root
// or format is ignored, and the sweep rewrites it.
func (c *ROMCheck) loadCache(root string, rootMame bool) {
	c.cacheOnce.Do(func() {
		c.cache = map[string]romCacheEntry{}
		if c.CachePath == "" {
			return
		}
		b, err := os.ReadFile(c.CachePath)
		if err != nil {
			return
		}
		var f romCacheFile
		if json.Unmarshal(b, &f) != nil || f.V != romCacheVersion || f.Root != root || f.RootMame != rootMame {
			return
		}
		c.cacheRaw = b
		if f.MRAs != nil {
			c.cache = f.MRAs
		}
	})
}

// cached returns the saved verdict for rel when the MRA and every archive
// it names are as the verdict saw them. stats memoises archive stats across
// one sweep, since many MRAs share their zips.
func (c *ROMCheck) cached(rel string, stats map[string]zipStamp) (romCacheEntry, bool) {
	e, ok := c.cache[rel]
	if !ok || e.Key {
		return romCacheEntry{}, false
	}
	st, err := os.Stat(c.cardPath(rel))
	if err != nil || st.Size() != e.Size || st.ModTime().UnixNano() != e.MTime {
		return romCacheEntry{}, false
	}
	for _, z := range e.Zips {
		now, seen := stats[z.Path]
		if !seen {
			now = zipStamp{Path: z.Path, Size: -1}
			if st, err := os.Stat(z.Path); err == nil {
				now.Size, now.MTime = st.Size(), st.ModTime().UnixNano()
			}
			stats[z.Path] = now
		}
		if now.Size != z.Size || now.Size >= 0 && now.MTime != z.MTime {
			return romCacheEntry{}, false
		}
	}
	return e, true
}

// stampVerdict builds the cache entry for a verdict the check just made,
// from the parsed MRA and the archives it looked at; ok is false when the
// MRA was never parsed (a replaced check in tests). The caller holds work.
func (c *ROMCheck) stampVerdict(root string, rel string, res ROMResult) (romCacheEntry, bool) {
	m, ok := c.mras[rel]
	if !ok {
		return romCacheEntry{}, false
	}
	e := romCacheEntry{Size: m.size, MTime: m.mtime.UnixNano(), Text: res.Text, Block: res.Block, About: res.Zips}
	seen := map[string]bool{}
	for _, s := range m.sections {
		for _, p := range s.parts {
			if keyPart(p) {
				e.Key = true
			}
			if p.name == "" {
				continue
			}
			for _, z := range p.zips {
				archive, _, ok := zipPath(root, z, p.name)
				if !ok || seen[archive] {
					continue
				}
				seen[archive] = true
				stamp := zipStamp{Path: archive, Size: -1}
				if ix, ok := c.zips[archive]; ok {
					stamp.Size, stamp.MTime = ix.size, ix.mtime.UnixNano()
				} else if st, err := os.Stat(archive); err == nil {
					stamp.Size, stamp.MTime = st.Size(), st.ModTime().UnixNano()
				}
				e.Zips = append(e.Zips, stamp)
			}
		}
	}
	return e, true
}

// saveCache writes the verdicts this run confirmed or made for paths, when
// they differ from what was read; MRAs no longer on the card drop out.
func (c *ROMCheck) saveCache(root string, rootMame bool, paths []string) {
	if c.CachePath == "" {
		return
	}
	f := romCacheFile{V: romCacheVersion, Root: root, RootMame: rootMame, MRAs: map[string]romCacheEntry{}}
	c.mu.Lock()
	for _, rel := range paths {
		if e, ok := c.stamps[rel]; ok {
			f.MRAs[rel] = e
		}
	}
	c.mu.Unlock()
	b, err := json.Marshal(f)
	if err != nil || bytes.Equal(b, c.cacheRaw) {
		return
	}
	if store.WriteAtomic(c.CachePath, b) == nil {
		c.cacheRaw = b
	}
}

// romCacheSaveEvery is how often a running sweep writes what it has, so a
// run cut short by a launch still leaves most of its work for the next.
const romCacheSaveEvery = 10 * time.Second
