package data

import "strings"

// DB is one database section of Downloader's downloader.ini: the section
// name and its db_url.
type DB struct {
	ID  string
	URL string
}

// sourceDBs names the Downloader database behind each feed source: the
// db_url the site's SOURCES table records and the section names update_all
// has written for it, older ones included. A source missing here is never
// hidden, so a source the feed gains later stays visible until it is added.
var sourceDBs = map[string]struct {
	url string
	ids []string
}{
	"distribution_mister": {"https://raw.githubusercontent.com/MiSTer-devel/Distribution_MiSTer/main/db.json.zip", []string{"distribution_mister"}},
	"jtbindb":             {"https://raw.githubusercontent.com/jotego/jtcores_mister/main/jtbindb.json.zip", []string{"jtcores"}},
	"coinop":              {"https://raw.githubusercontent.com/Coin-OpCollection/Distribution-MiSTerFPGA/db/db.json.zip", []string{"coin-opcollection/distribution-misterfpga", "atrac17/coin-op_collection"}},
	"meathax":             {"https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip", []string{"meathax/meatcores"}},
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
			have[strings.ToLower(db.URL)] = true
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
