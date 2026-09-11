package scan

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// DownloaderDBs reads the databases Downloader is set up to fetch from the
// card's downloader.ini, which update_all rewrites on every run: every
// section but [mister], with its db_url. Downloader's own default applies
// when the file has no database section: MiSTer Distribution alone. found
// is false when no ini exists, so the caller hides nothing.
func DownloaderDBs(card string) (dbs []data.DB, found bool) {
	for _, rel := range []string{"downloader.ini", "Scripts/downloader.ini"} {
		raw, err := os.ReadFile(filepath.Join(card, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		return parseDownloaderINI(string(raw)), true
	}
	return nil, false
}

// parseDownloaderINI lowercases section names and db_urls.
func parseDownloaderINI(s string) []data.DB {
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
	if len(dbs) == 0 {
		dbs = append(dbs, data.DB{ID: "distribution_mister"})
	}
	return dbs
}
