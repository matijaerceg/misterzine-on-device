package data

import (
	"reflect"
	"testing"
)

func TestLocalTakeovers(t *testing.T) {
	local := []Row{
		localRow("cuebrickj", "jttwin16", "_Arcade/_alternatives/_Cuebrick/Cue Brick (Japan).mra"),
		localRow("phantasm", "megasys1_a", "_Arcade/_alternatives/_Avenging Spirit/Phantasm (Japan).mra"),
		localRow("orphan", "defender", "_Arcade/_Extra/Orphan.mra"),
		{Title: "not local", Base: "Arcade", Src: "coinop", K: "x", SN: "orphan"},
	}
	catalogue := []Row{
		{Title: "Cue Brick", Base: "Arcade", Src: "jtbindb", K: "cuebrick", SN: "cuebrick", Family: "cuebrick", FamilySets: []string{"cuebrickj"}},
		{Title: "Avenging Spirit", Base: "Arcade", Src: "coinop", K: "avspirit", SN: "avspirit", Family: "avspirit", FamilySets: []string{"phantasm"}},
		{Title: "Phantasm (Japan)", Base: "Arcade", Src: "coinop", K: "phantasm-row", SN: "phantasm", Family: "avspirit"},
		{Title: "Orphan core", Base: "Computer", Src: "distribution_mister", K: "C64", SN: "orphan"},
	}
	got := LocalTakeovers(local, catalogue)
	// the row whose own setname matches wins over a family membership
	want := map[string]string{"local:cuebrickj": "cuebrick", "local:phantasm": "phantasm-row"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("takeovers %v, want %v", got, want)
	}
	if LocalTakeovers(nil, catalogue) != nil || LocalTakeovers(local, nil) != nil {
		t.Fatal("nothing to move must be nil")
	}
}

// A catalogue that lists a local game before the run's first card scan
// leaves the game's star under a key no row carries.
func TestStrandedLocalKeys(t *testing.T) {
	catalogue := []Row{
		{Title: "Cue Brick", Base: "Arcade", Src: "jtbindb", K: "cuebrickj", SN: "cuebrickj", Family: "cuebrick", FamilySets: []string{"cuebrickj"}},
		{Title: "Gigandes", Base: "Arcade", Src: "jtbindb", K: "gigandes", SN: "gigandes"},
		{Title: "Orphan core", Base: "Computer", Src: "distribution_mister", K: "C64", SN: "orphan"},
	}
	stand := localRow("gigandes", "taitox", "_Arcade/_Extra/Gigandes.mra")
	stand.Standin = true
	rows := append(append([]Row{}, catalogue...), stand, localRow("kept", "defender", "_Arcade/_Extra/Kept.mra"))
	keys := []string{
		"local:cuebrickj", // no row: the catalogue's row takes it
		"local:cuebrick",  // the family root: same row
		"local:gigandes",  // a standin still carries it
		"local:kept",      // a local row the scan keeps
		"local:orphan",    // only a non-arcade row names it
		"local:gone",      // nothing covers it: left alone
		"colony7",         // not a local key
	}
	got := StrandedLocalKeys(keys, rows, catalogue)
	want := map[string]string{"local:cuebrickj": "cuebrickj", "local:cuebrick": "cuebrickj"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stranded %v, want %v", got, want)
	}
	if StrandedLocalKeys(nil, rows, catalogue) != nil || StrandedLocalKeys([]string{"local:kept", "colony7"}, rows, catalogue) != nil {
		t.Fatal("nothing to move must be nil")
	}
}
