package data

import "sort"

// PreferenceKey keeps a Neo Geo family's star and version choice even if
// its primary cartridge disappears while another edition remains.
func (row *Row) PreferenceKey() string {
	if row.IsNeoGeoShortcut() && row.SN != "" {
		return "neo-family:" + row.SN
	}
	return row.K
}

// NeoMembers returns the discovered targets of a recognised Neo Geo family.
// Individual rows retain their keys, core status and launch paths.
func (ds *Dataset) NeoMembers(row *Row) []int {
	if row.IsNeoGeoShortcut() {
		return ds.neoFamilies[row.SN]
	}
	return nil
}

func (ds *Dataset) indexNeoFamilies() {
	ds.neoFamilies = make(map[string][]int)
	for i := ds.NCat; i < len(ds.Rows); i++ {
		r := &ds.Rows[i]
		if r.IsNeoGeoShortcut() && r.SN != "" {
			ds.neoFamilies[r.SN] = append(ds.neoFamilies[r.SN], i)
		}
	}
	for _, members := range ds.neoFamilies {
		sort.Slice(members, func(i, j int) bool {
			a, b := &ds.Rows[members[i]], &ds.Rows[members[j]]
			if (a.NeoSet == a.SN) != (b.NeoSet == b.SN) {
				return a.NeoSet == a.SN
			}
			if a.IsNeoGeoFile() != b.IsNeoGeoFile() {
				return !a.IsNeoGeoFile()
			}
			if a.NeoSet != b.NeoSet {
				return a.NeoSet < b.NeoSet
			}
			return a.MRA < b.MRA
		})
		for rank, i := range members {
			ds.Der[i].NeoRank = rank
		}
		ds.ByKey[ds.Rows[members[0]].PreferenceKey()] = members[0]
	}
}
