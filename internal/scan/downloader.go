package scan

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// DownloaderDBs reads the databases Downloader is set up to fetch from the
// card's downloader.ini, which update_all rewrites on every run: every
// section but [mister], with its db_url. Downloader's own default applies
// when the file has no database section: MiSTer Distribution alone. The
// drop-in files Downloader reads beside it count too (downloader_*.ini and
// downloader/*.ini): blahm1d publishes his database as downloader_blahm1d.ini
// to copy next to downloader.ini. found is false when no ini exists, so the
// caller hides nothing.
func DownloaderDBs(card string) (dbs []data.DB, found bool) {
	r := ScanDownloader(card)
	return r.DBs, r.Found
}

// MisterZineEntries lists the db_url of every [misterzine] section
// Downloader reads on the card, lowercased, in its order: the database
// MisterZine's own updates come from, the free releases' or MisterZine
// Arcade's. A section without a db_url lists "", and a comment after the
// address is dropped. found is false when the card has no downloader.ini,
// so the caller knows nothing about it.
func MisterZineEntries(card string) (urls []string, found bool) {
	dbs, found := DownloaderDBs(card)
	for _, db := range dbs {
		if db.ID == "misterzine" {
			url := ""
			if f := strings.Fields(db.URL); len(f) > 0 {
				url = f[0]
			}
			urls = append(urls, url)
		}
	}
	return urls, found
}

// parseINISections lists an ini's database sections, [mister] aside, and
// lowercases section names and db_urls.
func parseINISections(s string) []data.DB {
	dbs := []data.DB{}
	cur := -1
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				continue
			}
			id := strings.ToLower(strings.TrimSpace(line[1:end]))
			if id == "mister" {
				cur = -1
				continue
			}
			dbs = append(dbs, data.DB{ID: id})
			cur = len(dbs) - 1
			continue
		}
		if cur < 0 {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.ToLower(strings.TrimSpace(k)) == "db_url" {
			dbs[cur].URL = strings.ToLower(strings.Trim(strings.TrimSpace(v), `"'`))
		}
	}
	return dbs
}
