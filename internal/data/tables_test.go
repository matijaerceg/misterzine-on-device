package data

import (
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/gen"
)

const fetchedApp = `{
 "version": 1,
 "sources": {
  "newdev": {"name": "newdev (New Dev)", "section": "NewDev/MiSTer_NewDev", "db_url": "https://raw.githubusercontent.com/NewDev/MiSTer_NewDev/db/db.json.zip"},
  "jtbindb": {"name": "Jotego (JTcores)", "section": "jtcores", "db_url": ""},
  "meathax": {"name": "Meathax (renamed)", "section": "meathax/meatcores", "db_url": "https://raw.githubusercontent.com/meathax/meatcores/db/db.json.zip"}
 },
 "core_names": {"NewBoard": "New Dev Board", "TaitoGNET": "Taito G-NET (fetched)"},
 "later_field": "ignored"
}`

func useTables(t *testing.T, raw string) {
	t.Helper()
	tb, err := DecodeTables([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	SetTables(tb)
	t.Cleanup(func() { SetTables(nil) })
}

// A source the site adds after this build: the fetched table names it,
// labels its cores and lets Installed only hide it, with no release.
func TestFetchedTablesAddSource(t *testing.T) {
	if SrcFull("newdev") != "newdev" || CoreLabel("NewBoard", nil) != "New Board" {
		t.Fatal("the built-in tables already know the test source")
	}
	if HiddenSources([]DB{{ID: "distribution_mister"}})["newdev"] {
		t.Fatal("an unknown source must never be hidden")
	}
	useTables(t, fetchedApp)
	if SrcFull("newdev") != "newdev (New Dev)" || SrcShort("newdev") != "newdev" {
		t.Fatalf("labels: %q %q", SrcFull("newdev"), SrcShort("newdev"))
	}
	if CoreLabel("NewBoard", nil) != "New Dev Board" {
		t.Fatalf("core label %q", CoreLabel("NewBoard", nil))
	}
	if !HiddenSources([]DB{{ID: "distribution_mister"}})["newdev"] {
		t.Fatal("a fetched source the card lacks must be hidden")
	}
	for _, db := range []DB{
		{ID: "NewDev/MiSTer_NewDev"},
		{ID: "custom", URL: "https://raw.githubusercontent.com/newdev/mister_newdev/db/db.json.zip"},
	} {
		if HiddenSources([]DB{db})["newdev"] {
			t.Fatalf("installed source hidden: %+v", db)
		}
	}
}

// Fetched entries win; built-in ones the file lacks stay; nil goes back.
func TestFetchedTablesMerge(t *testing.T) {
	builtGNET := CoreLabel("TaitoGNET", nil)
	useTables(t, fetchedApp)
	if SrcFull("meathax") != "Meathax (renamed)" || CoreLabel("TaitoGNET", nil) != "Taito G-NET (fetched)" {
		t.Fatalf("fetched entries must win: %q %q", SrcFull("meathax"), CoreLabel("TaitoGNET", nil))
	}
	if SrcFull("shmupfan") != gen.SrcNames["shmupfan"] || CoreLabel("dec8", nil) != gen.CoreNames["dec8"] {
		t.Fatal("built-in entries the file lacks must stay")
	}
	// an update_all-toggled database keeps its db_url from sourceExtras
	if HiddenSources([]DB{{ID: "x", URL: "https://raw.githubusercontent.com/jotego/jtcores_mister/main/jtbindb.json.zip"}})["jtbindb"] {
		t.Fatal("jtbindb lost its sourceExtras db_url")
	}
	if HiddenSources([]DB{{ID: "atrac17/coin-op_collection"}})["coinop"] {
		t.Fatal("coinop lost its former section name")
	}
	SetTables(nil)
	if SrcFull("newdev") != "newdev" || CoreLabel("TaitoGNET", nil) != builtGNET {
		t.Fatal("SetTables(nil) must restore the built-in tables")
	}
}

func TestDecodeTablesRejects(t *testing.T) {
	good := `{"version":1,"sources":{"a":{"name":"A","section":"a/a","db_url":""}},"core_names":{"c":"C"}}`
	if _, err := DecodeTables([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string]string{
		"not json":      `<html>`,
		"version":       strings.Replace(good, `"version":1`, `"version":2`, 1),
		"no version":    strings.Replace(good, `"version":1,`, ``, 1),
		"no sources":    `{"version":1,"sources":{},"core_names":{}}`,
		"no name":       strings.Replace(good, `"name":"A"`, `"name":" "`, 1),
		"no section":    strings.Replace(good, `"section":"a/a"`, `"section":""`, 1),
		"plain http":    strings.Replace(good, `"db_url":""`, `"db_url":"http://x/db.json.zip"`, 1),
		"local id":      strings.Replace(good, `"a":{`, `"`+SrcLocal+`":{`, 1),
		"empty core":    strings.Replace(good, `"c":"C"`, `"c":""`, 1),
		"wrong types":   `{"version":1,"sources":{"a":"A"},"core_names":{}}`,
		"null sources":  `{"version":1,"sources":null}`,
		"empty core id": strings.Replace(good, `"c":"C"`, `"":"C"`, 1),
	} {
		if _, err := DecodeTables([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
