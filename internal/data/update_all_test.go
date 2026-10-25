package data

import "testing"

// update_all's built-in selection is the site's record: the four databases
// it ships toggles for are in, a hand-added one is out, and an unknown
// source (the local one included) is not known either way.
func TestInUpdateAllBuiltIn(t *testing.T) {
	for _, src := range []string{"distribution_mister", "jtbindb", "coinop", "theypsilon_unofficial_distribution"} {
		if in, known := InUpdateAll(src); !in || !known {
			t.Errorf("%s: in=%v known=%v, want in", src, in, known)
		}
	}
	for _, src := range []string{"meathax", "blahm1d", "rmcores"} {
		if in, known := InUpdateAll(src); in || !known {
			t.Errorf("%s: in=%v known=%v, want known and out", src, in, known)
		}
	}
	for _, src := range []string{"nobody", SrcLocal, ""} {
		if _, known := InUpdateAll(src); known {
			t.Errorf("%q: must not be known", src)
		}
	}
	if SrcSection("blahm1d") != "blahm1d" || SrcSection("coinop") != "Coin-OpCollection/Distribution-MiSTerFPGA" || SrcSection("nobody") != "" {
		t.Errorf("sections: %q %q %q", SrcSection("blahm1d"), SrcSection("coinop"), SrcSection("nobody"))
	}
}

// A fetched table sets the flag for a source it carries; an entry without
// the field (an older export) keeps the built-in answer; nil goes back.
func TestInUpdateAllFetched(t *testing.T) {
	if _, known := InUpdateAll("newdev"); known {
		t.Fatal("the built-in tables already know the test source")
	}
	useTables(t, fetchedApp)
	if in, known := InUpdateAll("newdev"); !in || !known {
		t.Errorf("newdev: in=%v known=%v, want in", in, known)
	}
	if in, _ := InUpdateAll("jtbindb"); !in {
		t.Error("jtbindb: an entry without in_update_all must keep the built-in answer")
	}
	if in, _ := InUpdateAll("meathax"); in {
		t.Error("meathax: an entry without in_update_all must keep the built-in answer")
	}
	if SrcSection("newdev") != "NewDev/MiSTer_NewDev" {
		t.Errorf("newdev section %q", SrcSection("newdev"))
	}
	useTables(t, `{"version": 1, "sources": {"jtbindb": {"name": "Jotego", "section": "jtcores", "db_url": "", "in_update_all": false}}, "core_names": {}}`)
	if in, _ := InUpdateAll("jtbindb"); in {
		t.Error("a fetched false must win over the built-in true")
	}
	SetTables(nil)
	if in, _ := InUpdateAll("jtbindb"); !in {
		t.Error("nil must go back to the built-in tables")
	}
}
