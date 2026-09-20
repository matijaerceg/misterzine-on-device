package data

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
)

// DB is one database section of Downloader's downloader.ini: the section
// name and its db_url.
type DB struct {
	ID  string
	URL string
}

// sourceDB is the Downloader database behind one feed source: its db_url and
// the section names update_all has written for it, older ones included,
// lower-cased for matching.
type sourceDB struct {
	url string
	ids []string
}

// sourceExtras holds what the site's tables do not carry: the db_url of the
// databases update_all has a settings toggle for (the site records URLs for
// the opt-in ones only) and the former section names update_all still
// writes. A source with neither entry needs nothing here.
var sourceExtras = map[string]sourceDB{
	"distribution_mister":                {url: "https://raw.githubusercontent.com/MiSTer-devel/Distribution_MiSTer/main/db.json.zip"},
	"jtbindb":                            {url: "https://raw.githubusercontent.com/jotego/jtcores_mister/main/jtbindb.json.zip"},
	"coinop":                             {url: "https://raw.githubusercontent.com/Coin-OpCollection/Distribution-MiSTerFPGA/db/db.json.zip", ids: []string{"atrac17/coin-op_collection"}},
	"theypsilon_unofficial_distribution": {url: "https://raw.githubusercontent.com/theypsilon/Unofficial_Distribution_MiSTer/main/unofficialdb.json.zip"},
}

// sourceDBs names the Downloader database behind each feed source, built
// from the site's tables (gen.SrcDBs, via cmd/mzgen) plus sourceExtras. A
// source missing here is never hidden, so a source the feed gains before
// the app is regenerated stays visible until then.
//
// Adding a source the site starts tracking: run cmd/mzgen. The full name,
// section and db_url arrive with it; the short chip derives from the full
// name unless srcShort in labels.go says otherwise. TestSourceTablesComplete
// fails when a source still lacks a db_url.
var sourceDBs = buildSourceDBs()

func buildSourceDBs() map[string]sourceDB {
	m := map[string]sourceDB{}
	for src, g := range gen.SrcDBs {
		s := sourceDB{url: g.URL, ids: []string{strings.ToLower(g.Section)}}
		if x, ok := sourceExtras[src]; ok {
			if x.url != "" {
				s.url = x.url
			}
			s.ids = append(s.ids, x.ids...)
		}
		m[src] = s
	}
	return m
}

// HiddenSources lists the feed sources whose Downloader database is not
// among dbs, matched by db_url or by section name, case-insensitively. A nil
// dbs means the configuration is unknown: nothing is hidden.
func HiddenSources(dbs []DB) map[string]bool {
	if dbs == nil {
		return nil
	}
	have := map[string]bool{}
	for _, db := range dbs {
		have[strings.ToLower(db.ID)] = true
		if db.URL != "" {
			url := strings.ToLower(db.URL)
			// Update All still writes the repository's former name.
			url = strings.Replace(url, "/theypsilon/distribution_unofficial_mister/", "/theypsilon/unofficial_distribution_mister/", 1)
			have[url] = true
		}
	}
	hidden := map[string]bool{}
	for src, s := range sourceDBs {
		if have[strings.ToLower(s.url)] {
			continue
		}
		found := false
		for _, id := range s.ids {
			if have[id] {
				found = true
				break
			}
		}
		if !found {
			hidden[src] = true
		}
	}
	return hidden
}
