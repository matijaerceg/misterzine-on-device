package data

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
)

// Tables is the site's releases/app.json: the source and multi-game core
// tables its page shows, which the pipeline writes from the page itself. The
// app fetches it after each catalogue check, so a source the site starts
// tracking gets its name, its Downloader section and db_url (Installed only
// hides it on a card without the database) and its core labels without an
// app release. The tables built in by cmd/mzgen stay underneath: a source or
// core the fetched file lacks keeps its built-in entry, and a file that is
// missing or does not decode changes nothing.
type Tables struct {
	Version   int                    `json:"version"`
	Sources   map[string]TableSource `json:"sources"`
	CoreNames map[string]string      `json:"core_names"`
}

// TableSource is one source in app.json. DBURL is empty for the databases
// update_all has a settings toggle for; sourceExtras supplies theirs.
type TableSource struct {
	Name    string `json:"name"`
	Section string `json:"section"`
	DBURL   string `json:"db_url"`
}

// TablesVersion is the app.json schema this build reads. The site adds
// fields without changing it; a file with another version is ignored.
const TablesVersion = 1

// DecodeTables parses and checks app.json. Any malformed entry rejects the
// whole file, so a half-broken table never mixes with the built-in one.
func DecodeTables(b []byte) (*Tables, error) {
	var t Tables
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("app.json: %v", err)
	}
	if t.Version != TablesVersion {
		return nil, fmt.Errorf("app.json: version %d, this build reads %d", t.Version, TablesVersion)
	}
	if len(t.Sources) == 0 {
		return nil, fmt.Errorf("app.json: no sources")
	}
	for id, s := range t.Sources {
		if id == "" || id == SrcLocal || strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Section) == "" {
			return nil, fmt.Errorf("app.json: source %q is incomplete", id)
		}
		if s.DBURL != "" && !strings.HasPrefix(s.DBURL, "https://") {
			return nil, fmt.Errorf("app.json: source %q: db_url %q", id, s.DBURL)
		}
	}
	for core, n := range t.CoreNames {
		if core == "" || strings.TrimSpace(n) == "" {
			return nil, fmt.Errorf("app.json: core name %q: %q", core, n)
		}
	}
	return &t, nil
}

// siteTables is the merged view the labels and HiddenSources read: built-in
// entries with the fetched ones laid over them. It is never changed once
// published, so a reader on any goroutine sees one whole table.
type siteTables struct {
	srcNames  map[string]string
	coreNames map[string]string
	dbs       map[string]sourceDB
}

var tables atomic.Pointer[siteTables]

func init() { tables.Store(mergeTables(nil)) }

// SetTables installs fetched tables over the built-in ones; nil goes back to
// the built-in tables alone. Rows ingested afterwards take the new labels,
// and the next HiddenSources call the new databases.
func SetTables(t *Tables) { tables.Store(mergeTables(t)) }

func mergeTables(t *Tables) *siteTables {
	m := &siteTables{srcNames: map[string]string{}, coreNames: map[string]string{}, dbs: map[string]sourceDB{}}
	for k, v := range gen.CoreNames {
		m.coreNames[k] = v
	}
	for k, v := range gen.SrcNames {
		m.srcNames[k] = v
	}
	sections := map[string]gen.SrcDB{}
	for k, v := range gen.SrcDBs {
		sections[k] = v
	}
	if t != nil {
		for k, v := range t.CoreNames {
			m.coreNames[k] = v
		}
		for k, s := range t.Sources {
			m.srcNames[k] = s.Name
			sections[k] = gen.SrcDB{Section: s.Section, URL: s.DBURL}
		}
	}
	for src, g := range sections {
		s := sourceDB{url: g.URL, ids: []string{strings.ToLower(g.Section)}}
		if x, ok := sourceExtras[src]; ok {
			if x.url != "" {
				s.url = x.url
			}
			s.ids = append(s.ids, x.ids...)
		}
		m.dbs[src] = s
	}
	return m
}
