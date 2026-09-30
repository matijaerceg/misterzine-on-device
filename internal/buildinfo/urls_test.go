package buildinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The scripts that switch a card between the builds, the free drop-in and
// the release check all name the same two databases as the app, which
// reads them to see which build a card's entry follows.
func TestDatabaseAddressesAgree(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, tc := range []struct {
		file string
		urls []string
	}{
		{"deploy/channel.py", []string{`"free": "` + FreeDBURL + `"`, `"beta": "` + BetaDBURL + `"`}},
		{"tools/verify_package.py", []string{`False: "` + FreeDBURL + `"`, `True: "` + BetaDBURL + `"`}},
		{"deploy/downloader_misterzine.ini", []string{"db_url = " + FreeDBURL + "\n"}},
	} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.file)))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(raw), "\r\n", "\n")
		for _, want := range tc.urls {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not have %s", tc.file, want)
			}
		}
	}
	if !strings.HasSuffix(BetaCatalogueURL, "/catalogue.json") || strings.TrimSuffix(BetaCatalogueURL, "catalogue.json") != strings.TrimSuffix(BetaDBURL, "beta.json.zip") {
		t.Errorf("the beta catalogue %s is not beside the beta database %s", BetaCatalogueURL, BetaDBURL)
	}
}
