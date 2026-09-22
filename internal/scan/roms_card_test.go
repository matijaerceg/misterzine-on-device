package scan

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestROMCheckCard runs the ROM check over every MRA under a real card's
// _Arcade and prints one line per MRA, the presence-only verdict v1.0.36
// shipped beside the new one, so two runs can be diffed, then the timings.
// It reads the card and writes nothing. On a board, from /tmp:
//
//	MZ_ROMCHECK_CARD=/media/fat ./scan.test -test.run '^TestROMCheckCard$' -test.v
func TestROMCheckCard(t *testing.T) {
	card := os.Getenv("MZ_ROMCHECK_CARD")
	if card == "" {
		t.Skip("set MZ_ROMCHECK_CARD to a card root")
	}
	var mras []string
	filepath.WalkDir(filepath.Join(card, "_Arcade"), func(p string, d fs.DirEntry, err error) error {
		// organiser trees repeat every MRA; the alternatives are wanted
		if err == nil && d.IsDir() && d.Name() != "_alternatives" && skipDir(d.Name()) {
			return filepath.SkipDir
		}
		if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".mra") {
			rel, _ := filepath.Rel(card, p)
			mras = append(mras, filepath.ToSlash(rel))
		}
		return nil
	})
	c := NewROMCheck(card)
	cold := make([]time.Duration, len(mras))
	kinds := map[string]int{}
	for i, rel := range mras {
		start := time.Now()
		res := c.Check(rel, true)
		cold[i] = time.Since(start)
		old := presenceOnly(card, rel)
		kind := "ok"
		if res.Text != "" {
			kind = strings.SplitN(res.Text, ":", 2)[0]
			if res.Block {
				kind += " (blocks)"
			} else {
				kind += " (warns)"
			}
		}
		kinds[kind]++
		if old != "" || res.Text != "" {
			t.Logf("mra %s\told=%q\tnew=%q\tblock=%v\t%v", rel, old, res.Text, res.Block, cold[i].Round(time.Microsecond))
		}
	}
	warm := make([]time.Duration, len(mras))
	for i, rel := range mras {
		start := time.Now()
		c.Check(rel, true)
		warm[i] = time.Since(start)
	}
	stats := func(d []time.Duration) string {
		if len(d) == 0 {
			return "none"
		}
		s := slices.Clone(d)
		slices.Sort(s)
		return fmt.Sprintf("p50 %v p95 %v max %v", s[len(s)/2].Round(time.Microsecond), s[len(s)*95/100].Round(time.Microsecond), s[len(s)-1].Round(time.Microsecond))
	}
	t.Logf("%d MRAs; first pass %s; second pass %s", len(mras), stats(cold), stats(warm))
	order := make([]int, len(mras))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(cold[b], cold[a]) })
	for _, i := range order[:min(10, len(order))] {
		t.Logf("slow %v %s", cold[i].Round(time.Microsecond), mras[i])
	}
	for k, n := range kinds {
		t.Logf("kind %s: %d", k, n)
	}
}

// presenceOnly is the v1.0.36 check, kept here only to compare against:
// every named part's zip list must have one file present, index-0
// sections being alternatives.
func presenceOnly(card, rel string) string {
	sections, issue := parseROMSections(filepath.Join(card, filepath.FromSlash(rel)), false)
	if issue != "" {
		return issue
	}
	root, _ := arcadeROMRoot(card)
	missing := func(s romSection) string {
		for _, p := range s.parts {
			found := false
			var names []string
			for _, z := range p.zips {
				z = strings.TrimSpace(z)
				names = append(names, z)
				path := filepath.Join(root, "mame", filepath.FromSlash(z))
				if strings.HasPrefix(z, "/") {
					path = filepath.Join(root, filepath.FromSlash(strings.TrimLeft(z, "/")))
				}
				if _, err := os.Stat(path); err == nil {
					found = true
					break
				}
			}
			if !found {
				return "Missing game ROM: " + strings.Join(names, " or ")
			}
		}
		return ""
	}
	zeroSeen, zeroOK, zeroIssue := false, false, ""
	for _, s := range sections {
		issue := missing(s)
		if s.index == 0 {
			zeroSeen = true
			if issue == "" {
				zeroOK = true
			} else if zeroIssue == "" {
				zeroIssue = issue
			}
		} else if issue != "" {
			return issue
		}
	}
	if zeroSeen && !zeroOK {
		return zeroIssue
	}
	return ""
}
