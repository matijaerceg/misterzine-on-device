package scan

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// storePath is the downloader's ledger of every file update_all installed,
// card-relative. It records the md5 the db carried at install time, which is
// exactly what the feed's bh field publishes for the current db.
const storePath = "Scripts/.config/downloader/downloader.json"

type storeFile struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// hashEntry is one md5 the app computed itself, remembered by size and
// mtime so a card of undated Jotego rbfs is hashed once, not every scan.
type hashEntry struct {
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	Hash  string `json:"hash"`
}

// hashes resolves the md5 of an installed rbf: the downloader store first
// (free), a cached own computation second, hashing the file last. It is
// created lazily so the launcher's Lookup-only ScanCores never pays for it.
type hashes struct {
	once  sync.Once
	store map[string]storeFile // lowercase card-relative path
	cache map[string]hashEntry // card-relative path as scanned
	dirty bool
	path  string // cache file, "" disables persistence
}

func (h *hashes) load(card string) {
	h.store = map[string]storeFile{}
	h.cache = map[string]hashEntry{}
	if raw, err := os.ReadFile(filepath.Join(card, filepath.FromSlash(storePath))); err == nil {
		var s struct {
			DBs map[string]struct {
				Files map[string]storeFile `json:"files"`
			} `json:"dbs"`
		}
		if json.Unmarshal(raw, &s) == nil {
			for _, db := range s.DBs {
				for p, f := range db.Files {
					// "|" prefixes mark files on external storage; the
					// path after it is still card-relative
					p = strings.ToLower(strings.TrimPrefix(p, "|"))
					if strings.HasSuffix(p, ".rbf") && f.Hash != "" {
						h.store[p] = f
					}
				}
			}
		}
	}
	if h.path != "" {
		if raw, err := os.ReadFile(h.path); err == nil {
			_ = json.Unmarshal(raw, &h.cache)
		}
	}
}

// of returns the rbf's md5 and modification time, "" when the file cannot be
// read. size/mtime guard the store answer too: a hand-copied file under the
// same name has a different size or a newer mtime than the ledger's install.
func (h *hashes) of(card, rel, cachePath string) (string, time.Time) {
	h.once.Do(func() { h.path = cachePath; h.load(card) })
	fi, err := os.Stat(filepath.Join(card, filepath.FromSlash(rel)))
	if err != nil {
		return "", time.Time{}
	}
	if e, ok := h.cache[rel]; ok && e.Size == fi.Size() && e.MTime == fi.ModTime().UnixNano() {
		return e.Hash, fi.ModTime()
	}
	if f, ok := h.store[strings.ToLower(rel)]; ok && f.Size == fi.Size() {
		return f.Hash, fi.ModTime()
	}
	f, err := os.Open(filepath.Join(card, filepath.FromSlash(rel)))
	if err != nil {
		return "", fi.ModTime()
	}
	defer f.Close()
	sum := md5.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", fi.ModTime()
	}
	hx := hex.EncodeToString(sum.Sum(nil))
	h.cache[rel] = hashEntry{Size: fi.Size(), MTime: fi.ModTime().UnixNano(), Hash: hx}
	h.dirty = true
	return hx, fi.ModTime()
}

// save persists own computations; a failed write only costs a rehash later.
func (h *hashes) save() {
	if !h.dirty || h.path == "" {
		return
	}
	if raw, err := json.Marshal(h.cache); err == nil {
		_ = store.WriteAtomic(h.path, raw)
	}
	h.dirty = false
}
