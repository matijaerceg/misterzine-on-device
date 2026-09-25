package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func TestDownloaderDBs(t *testing.T) {
	card := t.TempDir()
	if dbs, found := DownloaderDBs(card); found || dbs != nil {
		t.Fatal("a card without downloader.ini must report not found")
	}
	ini := "\ufeff; MiSTer Pi\r\n\r\n[mister]\r\nfilter = arcade\r\n\r\n" +
		"[distribution_mister]\r\ndb_url = https://raw.githubusercontent.com/MiSTer-devel/Distribution_MiSTer/main/db.json.zip\r\n\r\n" +
		"[Coin-OpCollection/Distribution-MiSTerFPGA]\r\nDB_URL = \"https://raw.githubusercontent.com/Coin-OpCollection/Distribution-MiSTerFPGA/db/db.json.zip\"\r\n" +
		"# hand-added\r\n[meathax/meatcores]\r\ndb_url = https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip\r\nfilter = [mister] arcade\r\n"
	os.MkdirAll(filepath.Join(card, "Scripts"), 0755)
	if err := os.WriteFile(filepath.Join(card, "Scripts", "downloader.ini"), []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	dbs, found := DownloaderDBs(card)
	want := []data.DB{
		{ID: "distribution_mister", URL: "https://raw.githubusercontent.com/mister-devel/distribution_mister/main/db.json.zip"},
		{ID: "coin-opcollection/distribution-misterfpga", URL: "https://raw.githubusercontent.com/coin-opcollection/distribution-misterfpga/db/db.json.zip"},
		{ID: "meathax/meatcores", URL: "https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip"},
	}
	if !found || !reflect.DeepEqual(dbs, want) {
		t.Fatalf("Scripts/downloader.ini: found=%v dbs=%v", found, dbs)
	}
	if got := data.HiddenSources(dbs); !reflect.DeepEqual(got, map[string]bool{"blahm1d": true, "jtbindb": true, "kuzecores": true, "rmcores": true, "slopcore": true, "theypsilon_unofficial_distribution": true}) {
		t.Fatalf("hidden = %v", got)
	}
	// the card root's ini wins; without database sections Downloader's
	// default is MiSTer Distribution alone
	if err := os.WriteFile(filepath.Join(card, "downloader.ini"), []byte("[mister]\nfilter = arcade\n"), 0644); err != nil {
		t.Fatal(err)
	}
	dbs, found = DownloaderDBs(card)
	if !found || !reflect.DeepEqual(dbs, []data.DB{{ID: "distribution_mister"}}) {
		t.Fatalf("root downloader.ini: found=%v dbs=%v", found, dbs)
	}
}

func TestDownloaderDropIns(t *testing.T) {
	card := t.TempDir()
	write := func(rel, s string) {
		t.Helper()
		p := filepath.Join(card, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(s), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// a drop-in alone neither makes the configuration known nor counts
	write("downloader_blahm1d.ini", "[blahm1d]\ndb_url = https://mister.blahm1d.com/db.json.zip\n")
	if dbs, found := DownloaderDBs(card); found || dbs != nil {
		t.Fatalf("drop-in without downloader.ini: found=%v dbs=%v", found, dbs)
	}
	// Downloader's default database stands beside the drop-ins
	write("downloader.ini", "[mister]\nfilter = arcade\n")
	write("downloader/extra.ini", "[mister]\nfilter = x\n[rmonic79/rmcores]\ndb_url = https://raw.githubusercontent.com/rmonic79/rmcores/db/db.json.zip\n")
	write("downloader/.hidden.ini", "[kuzearcade/kuzecores]\n")
	write("downloader/notes.txt", "[TheJesusFish/Slop-Core]\n")
	dbs, found := DownloaderDBs(card)
	want := []data.DB{
		{ID: "distribution_mister"},
		{ID: "rmonic79/rmcores", URL: "https://raw.githubusercontent.com/rmonic79/rmcores/db/db.json.zip"},
		{ID: "blahm1d", URL: "https://mister.blahm1d.com/db.json.zip"},
	}
	if !found || !reflect.DeepEqual(dbs, want) {
		t.Fatalf("drop-ins: found=%v dbs=%v", found, dbs)
	}
	hidden := data.HiddenSources(dbs)
	if hidden["blahm1d"] || hidden["rmcores"] || hidden["distribution_mister"] || !hidden["kuzecores"] || !hidden["slopcore"] {
		t.Fatalf("hidden = %v", hidden)
	}
}
