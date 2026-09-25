package scan

import (
	"os"
	"path/filepath"
	"sort"
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
	for _, rel := range []string{"downloader.ini", "Scripts/downloader.ini"} {
		path := filepath.Join(card, filepath.FromSlash(rel))
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		dbs = parseDownloaderINI(string(raw))
		for _, f := range dropInFiles(filepath.Dir(path)) {
			if raw, err := os.ReadFile(f); err == nil {
				dbs = append(dbs, parseINISections(string(raw))...)
			}
		}
		return dbs, true
	}
	return nil, false
}

// dropInFiles lists the drop-in database files Downloader reads from the
// folder holding downloader.ini, in its order: downloader/*.ini, then
// downloader_*.ini, each sorted, dotfiles skipped.
func dropInFiles(dir string) []string {
	var out []string
	for _, pattern := range []string{filepath.Join(dir, "downloader", "*.ini"), filepath.Join(dir, "downloader_*.ini")} {
		m, _ := filepath.Glob(pattern)
		sort.Strings(m)
		for _, f := range m {
			if !strings.HasPrefix(filepath.Base(f), ".") {
				out = append(out, f)
			}
		}
	}
	return out
}

// parseDownloaderINI reads downloader.ini: its database sections, or MiSTer
// Distribution alone when it has none.
func parseDownloaderINI(s string) []data.DB {
	dbs := parseINISections(s)
	if len(dbs) == 0 {
		dbs = append(dbs, data.DB{ID: "distribution_mister"})
	}
	return dbs
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
