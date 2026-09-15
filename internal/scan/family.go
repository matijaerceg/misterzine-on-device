package scan

import (
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"os"
	"path/filepath"
)

type mainHeader struct {
	size, mtime int64
	alt         Alt
}

// FamilyCache belongs to the sequential background scanner, never the UI.
type FamilyCache struct{ headers map[string]mainHeader }

// Resolve builds a snapshot of launch paths for the supplied catalog.
// Missing metadata is optional: it never prevents existing setname matches.
func (c *FamilyCache) Resolve(card string, alts []Alt, rows []data.Row) map[string][]string {
	index := IndexAlternatives(alts)
	fresh := map[string]mainHeader{}
	out := map[string][]string{}
	for _, row := range rows {
		if !row.IsArcade() {
			continue
		}
		if identity(row.Family) == "" && row.MRA != "" {
			p := filepath.Join(card, filepath.FromSlash(row.MRA))
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				h, ok := c.headers[p]
				if !ok || h.size != st.Size() || h.mtime != st.ModTime().UnixNano() {
					a, valid := ParseMRAHeader(p)
					ok = valid
					h = mainHeader{st.Size(), st.ModTime().UnixNano(), a}
				}
				if ok {
					fresh[p] = h
					if sameCore(h.alt.RBF, row.Core) {
						row.Family = identity(h.alt.Parent)
					}
				}
			}
		}
		out[row.K] = index.ForRow(&row)
	}
	c.headers = fresh
	return out
}
