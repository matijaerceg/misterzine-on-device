package data

import (
	"strings"
	"testing"
)

func TestLocalRowsRoundTrip(t *testing.T) {
	stand := localRow("volfied", "taitox", "_Arcade/Volfied.mra")
	stand.Standin = true
	stand.FamilySets = []string{"volfiedu"}
	beta := localRow("zzz", "jtzzz", "_Arcade/zzz.mra")
	beta.Beta, beta.Gate = true, "jtbeta"
	local := []Row{localRow("orphan", "defender", "_Arcade/_Extra/Orphan.mra"), stand, beta}
	b, err := EncodeLocalRows("cat", local)
	if err != nil {
		t.Fatal(err)
	}
	hash, got, err := DecodeLocalRows(b)
	if err != nil || hash != "cat" {
		t.Fatalf("hash %q, err %v", hash, err)
	}
	if !SameLocalRows(local, got) || Generation("cat", got) != Generation("cat", local) {
		t.Fatalf("rows changed on the way through the file: %+v", got)
	}
	if !got[1].Standin || got[2].Gate != "jtbeta" {
		t.Fatalf("standin or gate lost: %+v", got)
	}

	// a card without local rows saves an empty set, which reads back empty
	b, _ = EncodeLocalRows("cat", nil)
	if _, got, err := DecodeLocalRows(b); err != nil || len(got) != 0 {
		t.Fatalf("empty set: %v, %v", got, err)
	}
}

func TestLocalRowsRejectsOtherVersionsAndJunk(t *testing.T) {
	for _, body := range []string{
		`{"v":0,"hash":"cat","rows":[{"k":"local:x","src":"local"}]}`,
		`{"v":2,"hash":"cat","rows":[{"k":"local:x","src":"local"}]}`,
		`[{"k":"local:x","src":"local"}]`,
		`{"v":1,"hash":"cat","rows":[{"k":"local:x"`,
		``,
	} {
		if _, rows, err := DecodeLocalRows([]byte(body)); err == nil || rows != nil {
			t.Errorf("%q read as %v", body, rows)
		} else if !strings.HasPrefix(err.Error(), "local rows: ") {
			t.Errorf("%q: error %q does not say which file", body, err)
		}
	}
}

func TestSameLocalRowsSeesWhatGenerationIgnores(t *testing.T) {
	a := []Row{localRow("orphan", "defender", "_Arcade/Orphan.mra")}
	b := []Row{a[0]}
	b[0].Title = "Orphan (World)"
	if Generation("cat", a) != Generation("cat", b) {
		t.Fatal("a title is not part of the generation; the test needs one that is not")
	}
	if SameLocalRows(a, b) || !SameLocalRows(a, []Row{a[0]}) || SameLocalRows(a, nil) || !SameLocalRows(nil, []Row{}) {
		t.Fatal("SameLocalRows")
	}
}
