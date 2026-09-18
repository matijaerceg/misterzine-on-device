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
