package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func writeSourceINI(t *testing.T, card, rel, content string) {
	t.Helper()
	path := filepath.Join(card, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func sourceFile(t *testing.T, result DownloaderScan, path string) data.SourceFile {
	t.Helper()
	for _, f := range result.Files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("no check for %s: %+v", path, result.Files)
	return data.SourceFile{}
}

func TestSourceFilesKeepDatabaseOriginsAndPrecedence(t *testing.T) {
	card := t.TempDir()
	writeSourceINI(t, card, "downloader.ini", "[mister]\n[distribution_mister]\n[jtcores]\n")
	writeSourceINI(t, card, "downloader/renamed.ini", "[my database]\ndb_url=https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip\n")
	writeSourceINI(t, card, "downloader_extra.ini", "[jtcores]\n[misterzine]\ndb_url=https://example.test/private?key=SECRET\n")
	writeSourceINI(t, card, "Scripts/downloader.ini", "[coin-opcollection/distribution-misterfpga]\n")
	writeSourceINI(t, card, "Scripts/downloader_ignored.ini", "[blahm1d]\n")
	writeSourceINI(t, card, "downloader/.hidden.ini", "[blahm1d]\n")
	r := ScanDownloader(card)
	if !r.Found || len(r.DBs) != 5 {
		t.Fatalf("configuration: %+v", r)
	}
	if f := sourceFile(t, r, "downloader/renamed.ini"); f.State != data.SourceFileUsed || len(f.Sections) != 1 || !reflect.DeepEqual(f.Sections[0].Sources, []string{"meathax"}) {
		t.Fatalf("renamed section did not resolve by URL: %+v", f)
	}
	if f := sourceFile(t, r, "downloader_extra.ini"); len(f.Sections) != 2 || len(f.Sections[1].Sources) != 0 || f.Sections[1].ID != "misterzine" {
		t.Fatalf("unknown database not retained: %+v", f)
	}
	for _, path := range []string{"Scripts/downloader.ini", "Scripts/downloader_ignored.ini"} {
		f := sourceFile(t, r, path)
		if f.State != data.SourceFileSkipped || len(f.Sections) != 0 {
			t.Fatalf("unused file contributed databases: %+v", f)
		}
	}
	hidden := data.HiddenSources(r.DBs)
	if hidden["jtbindb"] || hidden["meathax"] || !hidden["coinop"] || !hidden["blahm1d"] {
		t.Fatalf("provenance changed filtering: %v", hidden)
	}
	encoded, err := json.Marshal(r.Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"SECRET", "https://", "db_url", ".hidden.ini"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("%q leaked into the display/report snapshot", forbidden)
		}
	}
}

func TestSourceFilesMissingDefaultFallbackAndRescan(t *testing.T) {
	card := t.TempDir()
	writeSourceINI(t, card, "downloader_extra.ini", "[jtcores]\n")
	r := ScanDownloader(card)
	if r.Found || r.DBs != nil || sourceFile(t, r, "downloader.ini").State != data.SourceFileMissing || sourceFile(t, r, "downloader_extra.ini").State != data.SourceFileSkipped {
		t.Fatalf("drop-in alone enabled a source: %+v", r)
	}
	if err := os.Mkdir(filepath.Join(card, "downloader.ini"), 0755); err != nil {
		t.Fatal(err)
	}
	writeSourceINI(t, card, "Scripts/downloader.ini", "[mister]\nfilter=arcade\n")
	writeSourceINI(t, card, "Scripts/downloader_extra.ini", "[blahm1d]\n")
	writeSourceINI(t, card, "Scripts/downloader/empty.ini", "[mister]\n")
	r = ScanDownloader(card)
	if !r.Found || sourceFile(t, r, "downloader.ini").State != data.SourceFileUnreadable || len(r.DBs) != 2 {
		t.Fatalf("unreadable root should fall back: %+v", r)
	}
	f := sourceFile(t, r, "Scripts/downloader.ini")
	if f.State != data.SourceFileUsed || len(f.Sections) != 1 || !f.Sections[0].Default || f.Sections[0].Sources[0] != "distribution_mister" {
		t.Fatalf("implicit default missing: %+v", f)
	}
	if f := sourceFile(t, r, "Scripts/downloader/empty.ini"); f.State != data.SourceFileUsed || f.Note != "No database sections" {
		t.Fatalf("empty drop-in: %+v", f)
	}
	if err := os.Remove(filepath.Join(card, "downloader.ini")); err != nil {
		t.Fatal(err)
	}
	writeSourceINI(t, card, "downloader.ini", "[jtcores]\n")
	r = ScanDownloader(card)
	if sourceFile(t, r, "Scripts/downloader.ini").State != data.SourceFileSkipped || !data.HiddenSources(r.DBs)["blahm1d"] {
		t.Fatalf("rescan retained the old source: %+v", r)
	}
}

func TestSourceFilesDistinguishUnreadableDropInLocation(t *testing.T) {
	card := t.TempDir()
	writeSourceINI(t, card, "downloader.ini", "[mister]\n")
	writeSourceINI(t, card, "downloader", "a file, not a directory")
	r := ScanDownloader(card)
	if f := sourceFile(t, r, "downloader/*.ini"); f.State != data.SourceFileUnreadable || f.Note != "Could not list this location" {
		t.Fatalf("unreadable location reported as missing: %+v", f)
	}
	if !r.Found || len(r.DBs) != 1 || r.DBs[0].ID != "distribution_mister" {
		t.Fatalf("unreadable drop-in directory changed defaults: %+v", r.DBs)
	}
}
