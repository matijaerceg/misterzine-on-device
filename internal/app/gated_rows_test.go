package app

import (
	"strings"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// A gated row's Details line and chip follow the feed's gate term: Jotego's
// betas keep the jtbeta.zip wording (also for an older export with no term),
// Coin-Op's alphas and betas name the licence key and read their stage, and
// an unknown source gets the generic line.
func TestGatedRowsDetailsAndChips(t *testing.T) {
	rows := []data.Row{
		{K: "jt", Title: "Jotego Beta", Base: "Arcade", Src: "jtbindb", Beta: true, Gate: "jtbeta", Updated: "2026-09-01"},
		{K: "old", Title: "Old Export", Base: "Arcade", Src: "jtbindb", Beta: true, Updated: "2026-09-01"},
		{K: "cob", Title: "Coin-Op Beta", Base: "Arcade", Src: "coinop", Beta: true, Gate: "coinop-collection-beta", Updated: "2026-09-01"},
		{K: "coa", Title: "Coin-Op Alpha", Base: "Arcade", Src: "coinop", Beta: true, Gate: "coinop-collection-alpha", Updated: "2026-09-01"},
		{K: "oth", Title: "Other Gate", Base: "Arcade", Src: "meathax", Beta: true, Gate: "meat-early", Updated: "2026-09-01"},
		{K: "pub", Title: "Public", Base: "Arcade", Src: "coinop", Updated: "2026-09-01"},
	}
	a := New(Config{PhysW: 320, PhysH: 240, Status: func(int) data.Status { return data.StatusFoundUndated }}, data.Ingest(rows, "h", time.Now()), nil)
	want := map[string]struct{ line, chip string }{
		"jt":  {"Patreon beta: needs Jotego's jtbeta.zip", "beta"},
		"old": {"Patreon beta: needs Jotego's jtbeta.zip", "beta"},
		"cob": {"Patreon beta: needs a Coin-Op licence key", "beta"},
		"coa": {"Patreon alpha: needs a Coin-Op licence key", "alpha"},
		"oth": {"Patreon beta: early access, needs a key from the author", "beta"},
		"pub": {"", ""},
	}
	for k, w := range want {
		i := a.ds.Index(k)
		if i < 0 {
			t.Fatalf("%s not indexed", k)
		}
		row, d := &a.ds.Rows[i], &a.ds.Der[i]
		var texts []string
		for _, l := range a.detailLines(row, d, i) {
			texts = append(texts, l.text)
		}
		joined := strings.Join(texts, "\n")
		if w.line == "" {
			if strings.Contains(joined, "Patreon") {
				t.Errorf("%s: public row has a gate line:\n%s", k, joined)
			}
		} else if !strings.Contains(joined, w.line) {
			t.Errorf("%s: details lack %q:\n%s", k, w.line, joined)
		}
		ch := chips(row, d)
		if w.chip == "" {
			if len(ch) != 0 {
				t.Errorf("%s: chips %v", k, ch)
			}
		} else if len(ch) != 1 || ch[0] != w.chip {
			t.Errorf("%s: chips %v, want [%s]", k, ch, w.chip)
		}
		if got := data.BetaKind(row); (got == "beta") != row.Beta {
			t.Errorf("%s: BetaKind %q", k, got)
		}
	}
}
