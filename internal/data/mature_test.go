package data

import (
	"encoding/json"
	"testing"
)

func TestMatureFlagDecodesAndHides(t *testing.T) {
	var rows []Row
	if err := json.Unmarshal([]byte(`[{"title":"Super Model","mat":1},{"title":"Rainbow Islands"}]`), &rows); err != nil {
		t.Fatal(err)
	}
	if rows[0].Mat != 1 || rows[1].Mat != 0 {
		t.Fatalf("mat = %d, %d", rows[0].Mat, rows[1].Mat)
	}
	f := Filters{HideMature: true}
	if !f.Active() {
		t.Fatal("hiding mature games is a narrowing")
	}
	if why := f.Why(&rows[0], &Derived{}, StatusCurrent, false, false, ROMState(0)); why == "" {
		t.Fatal("a mature row passed with mature games hidden")
	}
	if why := f.Why(&rows[1], &Derived{}, StatusCurrent, false, false, ROMState(0)); why != "" {
		t.Fatalf("an ordinary row was hidden: %s", why)
	}
}

// A card's MRA carries no mature mark, so a local copy of a mature
// catalogue game, by setname or by its MRA's parent, takes the game's.
func TestMergeLocalInheritsMature(t *testing.T) {
	cat := []Row{
		{Title: "Super Model", Base: "Arcade", Src: "bazset", K: "supmodel", SN: "supmodel", Family: "supmodel", Mat: 1},
		{Title: "Galaxian", Base: "Arcade", Src: "distribution_mister", K: "galaxian", SN: "galaxian", Family: "galaxian"},
	}
	stand := localRow("supmodel", "other", "_Arcade/Super Model.mra")
	stand.Standin = true
	hack := localRow("supmodlx", "other", "_Arcade/Super Model X.mra")
	hack.Family = "supmodel"
	plain := localRow("andromd", "galaxian", "_Arcade/Andromeda.mra")
	plain.Family = "andromd"
	got := map[string]int{}
	for _, r := range MergeLocal(cat, []Row{stand, hack, plain}) {
		got[r.K] = r.Mat
	}
	if got["local:supmodel"] != 1 || got["local:supmodlx"] != 1 || got["local:andromd"] != 0 {
		t.Fatalf("mature marks %v", got)
	}
}
