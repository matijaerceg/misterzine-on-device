// mzaltaudit compares legacy and family matching without changing the card.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/scan"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type finding struct {
	Title     string     `json:"title"`
	Key       string     `json:"key"`
	Family    string     `json:"family,omitempty"`
	Before    int        `json:"before"`
	After     int        `json:"after"`
	Recovered []string   `json:"recovered,omitempty"`
	Excluded  []scan.Alt `json:"excluded,omitempty"`
}

func main() {
	card := flag.String("card", "", "card root (read only)")
	catalog := flag.String("data", "", "catalog JSON")
	cache := flag.String("cache", "", "optional disposable scan cache; never use the live app cache")
	flag.Parse()
	f, err := os.Open(*catalog)
	if err != nil {
		panic(err)
	}
	rows, err := data.DecodeRows(f)
	f.Close()
	if err != nil {
		panic(err)
	}
	start := time.Now()
	alts, skipped, err := scan.ScanAlternativesWithError(*card, *cache)
	if err != nil {
		panic(err)
	}
	var families scan.FamilyCache
	resolved := families.Resolve(*card, alts, rows)
	cold := time.Since(start)
	start = time.Now()
	again, _, err := scan.ScanAlternativesWithError(*card, *cache)
	if err != nil {
		panic(err)
	}
	families.Resolve(*card, again, rows)
	warm := time.Since(start)

	// Local discovery: the MRAs no catalogue row accounts for. Timed apart
	// from the alternatives pass; a warm run should read no files.
	attached := scan.AttachedPaths(resolved)
	localCache := ""
	if *cache != "" {
		localCache = *cache + ".local"
	}
	start = time.Now()
	local := scan.DiscoverLocal(*card, localCache, rows, alts, attached, false)
	localCold := time.Since(start)
	start = time.Now()
	scan.DiscoverLocal(*card, localCache, rows, alts, attached, false)
	localWarm := time.Since(start)
	type localRow struct {
		Key   string   `json:"key"`
		Title string   `json:"title"`
		Path  string   `json:"path"`
		Core  string   `json:"core"`
		Alts  []string `json:"alts,omitempty"`
	}
	locals := make([]localRow, 0, len(local.Rows))
	for _, r := range local.Rows {
		locals = append(locals, localRow{r.K, r.Title, r.MRA, r.Core, local.Alts[r.K]})
	}
	localErr := ""
	if local.Err != nil {
		localErr = local.Err.Error()
	}
	findings := []finding{}
	changes := []finding{}
	changed := 0
	added := 0
	for _, r := range rows {
		if !r.IsArcade() {
			continue
		}
		old := map[string]bool{}
		now := map[string]bool{}
		for _, p := range resolved[r.K] {
			now[p] = true
		}
		for _, a := range alts {
			hit := strings.EqualFold(a.Setname, r.SN)
			for _, z := range a.Zips {
				if z == strings.ToLower(r.SN)+".zip" {
					hit = true
				}
			}
			if !hit || r.SN == "" {
				continue
			}
			// Delegate compatible-core semantics to the production matcher.
			probe := a
			probe.Parent = ""
			probe.Zips = nil
			probe.Setname = r.SN
			rr := r
			rr.Family = ""
			if len(scan.Alternatives([]scan.Alt{probe}, &rr)) > 0 {
				old[a.Path] = true
			}
		}
		x := finding{Title: r.Title, Key: r.K, Family: r.Family, Before: len(old), After: len(now)}
		flagged := false
		for _, a := range alts {
			if now[a.Path] && !old[a.Path] {
				x.Recovered = append(x.Recovered, a.Path)
			}
			if !strings.EqualFold(strings.TrimLeft(path.Base(path.Dir(a.Path)), "_"), r.Title) {
				continue
			}
			probe := a
			probe.Parent = ""
			probe.Zips = nil
			probe.Setname = r.SN
			rr := r
			rr.Family = ""
			if len(scan.Alternatives([]scan.Alt{probe}, &rr)) == 0 {
				continue
			}
			if !old[a.Path] {
				flagged = true
			}
			if !now[a.Path] {
				x.Excluded = append(x.Excluded, a)
			}
		}
		if len(x.Recovered) > 0 {
			changes = append(changes, x)
			changed++
			added += len(x.Recovered)
		}
		if flagged {
			findings = append(findings, x)
		}
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Title < findings[j].Title })
	result := struct {
		Alternatives int
		Skipped      int
		Cold         string
		Warm         string
		ChangedRows  int
		AddedMatches int
		Changes      []finding
		Findings     []finding
		LocalFiles   int    // MRAs the local walk read (outside _alternatives)
		LocalRows    int    // games no catalogue row accounts for
		LocalSkipped int    // unreadable headers or files without a setname
		LocalCold    string // discovery walk, files read
		LocalWarm    string // discovery walk, from the cache
		LocalError   string `json:",omitempty"`
		Locals       []localRow
	}{len(alts), len(skipped), cold.String(), warm.String(), changed, added, changes, findings,
		local.Files, len(local.Rows), len(local.Skipped), localCold.String(), localWarm.String(), localErr, locals}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
