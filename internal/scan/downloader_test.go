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
	if got := data.HiddenSources(dbs); !reflect.DeepEqual(got, map[string]bool{"jtbindb": true}) {
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
