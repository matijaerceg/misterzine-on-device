package data

import "strings"

// Core batches mirror the site's batch rows: one shipped build stamps every
// game its core serves with the same Build Date, so a core update used to
// push 20-50 rows to the top of the list at once. Under the date orders,
// BatchMin or more plain rows sharing a core and the order's date (the build
// under Updated, the debut under Debut, where a new core's first wave lands
// together) fold under one umbrella titled after the core. Same-game
// umbrellas keep their members, so nothing nests two deep; a game whose
// same-game group did not fold in the view joins its batch as a plain row.
// While a search is on, a batch gathers even one matching game under its
// header, so a match keeps the context the search would otherwise drop.

// BatchMin is how many games on one core and date make a batch, the site's
// BATCH_MIN.
const BatchMin = 3

// BatchKeyPrefix starts a batch's key, as GroupKeyPrefix starts a same-game
// umbrella's.
const BatchKeyPrefix = "b:"

// BatchKey is the key the cursor remembers a batch by.
func BatchKey(g string) string { return BatchKeyPrefix + g }

// BatchG is a batch's G: its core and date, joined by an @ no setname
// carries, the site's batchG.
func BatchG(core, day string) string { return core + "@" + day }

// IsBatch reports whether umbrella G g is a core batch's.
func IsBatch(g string) bool { return strings.Contains(g, "@") }

// Batches reports whether the order folds core batches: the date orders,
// whose rows a batch shares its place with.
func (m SortMode) Batches() bool { return m == SortUpdated || m == SortDebut }

// batchDate is the date row r shares with its batch under the order.
func batchDate(r *Row, mode SortMode) string {
	if mode == SortDebut {
		return r.Date
	}
	return r.Updated
}

// batchOf is the batch row i can fold under, "" for none: its catalogue
// row's core and date. A local row is no catalogue game; a standin goes
// with the catalogue row it stands in for. During a search, members of a
// same-game group stay out of batches, as on the site.
func (ds *Dataset) batchOf(i int, mode SortMode, search bool) string {
	s := ds.SortRow(i)
	r := &ds.Rows[s]
	if s >= ds.NCat || r.Core == "" {
		return ""
	}
	day := batchDate(r, mode)
	if day == "" || search && ds.grouped(r) {
		return ""
	}
	return BatchG(r.Core, day)
}

// grouped reports whether row r belongs to a same-game group of two
// catalogue rows or more.
func (ds *Dataset) grouped(r *Row) bool { return r.G != "" && ds.groupSize[r.G] >= 2 }

// batchTotal is how many catalogue games batch g has under the order,
// filters and search ignored, same-game group members left out.
func (ds *Dataset) batchTotal(g string, mode SortMode) int {
	if mode == SortDebut {
		return ds.debutBatch[g]
	}
	return ds.buildBatch[g]
}

// countBatches fills the catalogue-wide counts batchTotal reads.
func (ds *Dataset) countBatches() {
	ds.groupSize = map[string]int{}
	for i := range ds.Rows[:ds.NCat] {
		if g := ds.Rows[i].G; g != "" {
			ds.groupSize[g]++
		}
	}
	ds.buildBatch, ds.debutBatch = map[string]int{}, map[string]int{}
	for i := range ds.Rows[:ds.NCat] {
		r := &ds.Rows[i]
		if r.Core == "" || ds.grouped(r) {
			continue
		}
		if r.Updated != "" {
			ds.buildBatch[BatchG(r.Core, r.Updated)]++
		}
		if r.Date != "" {
			ds.debutBatch[BatchG(r.Core, r.Date)]++
		}
	}
}
