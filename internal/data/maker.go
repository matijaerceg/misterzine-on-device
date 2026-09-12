package data

import "strings"

// makerSuffixes are the words dropped from the end of a manufacturer
// credit, one after another, so that "Taito Corporation Japan", "Taito
// America Corporation" and "Taito" are one maker. A credit that is only
// such a word keeps it.
var makerSuffixes = map[string]bool{
	"corporation": true, "corp": true, "co": true, "company": true, "ltd": true, "limited": true,
	"inc": true, "enterprises": true, "entertainment": true, "industries": true, "electronics": true,
	"games": true, "gmbh": true, "srl": true, "s.a": true,
	"japan": true, "usa": true, "u.s.a": true, "america": true, "europe": true,
}

// MakerStem reduces a catalogue manufacturer credit to the company it
// names first: joint credits ("Sega / Westone") and licences ("Cave (Capcom
// license)") keep only the first name, then corporate and regional suffix
// words go. The result keeps the credit's own spelling and case; MakerKey
// folds it for grouping. Blank stays blank.
func MakerStem(raw string) string {
	s := strings.TrimSpace(raw)
	if i := strings.IndexAny(s, "/("); i >= 0 {
		s = s[:i]
	}
	words := strings.Fields(s)
	for len(words) > 1 && makerSuffixes[strings.ToLower(strings.Trim(words[len(words)-1], ".,"))] {
		words = words[:len(words)-1]
	}
	stem := strings.TrimRight(strings.Join(words, " "), " .,")
	if stem == "" {
		return strings.TrimSpace(raw)
	}
	return stem
}

// MakerKey is the group a credit belongs to: its stem, accent-folded and
// lower-cased, so "CAVE" and "Cave (Capcom license)" share one.
func MakerKey(raw string) string {
	return strings.ToLower(ASCII(MakerStem(raw)))
}

// makerLabels picks each group's name: the stem most rows use, spelled as
// they spell it (ties go to the first in byte order, so a capital wins).
func makerLabels(rows []Row) map[string]string {
	counts := map[string]map[string]int{}
	for i := range rows {
		key := MakerKey(rows[i].Manufacturer)
		if key == "" {
			continue
		}
		if counts[key] == nil {
			counts[key] = map[string]int{}
		}
		counts[key][MakerStem(rows[i].Manufacturer)]++
	}
	labels := make(map[string]string, len(counts))
	for key, stems := range counts {
		best, n := "", 0
		for stem, c := range stems {
			if c > n || (c == n && stem < best) {
				best, n = stem, c
			}
		}
		labels[key] = ASCII(best)
	}
	return labels
}
