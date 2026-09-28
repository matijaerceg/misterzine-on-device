package scan

import (
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

// Access is what the catalogue says about an arcade core: whether it is a
// Patreon-gated early build and the download-filter term that gates it
// (jtbeta, coinop-collection-beta, coinop-collection-alpha), lowercase.
// Known is false when the catalogue does not list the core, or lists it
// both gated and not.
type Access struct {
	Known bool
	Beta  bool
	Gate  string
}

// JotegoBeta reports whether the core needs Jotego's jtbeta.zip. A beta
// row without a gate term comes from an older export and was Jotego's.
func (a Access) JotegoBeta() bool { return a.Beta && (a.Gate == "" || a.Gate == "jtbeta") }

// CoreAccess answers Access for the core an MRA names, from the catalogue's
// arcade rows. The feed flags beta per core: every row of a gated core
// carries it, so the rows of one core agree, and a core whose rows disagree
// is unknown rather than either. Built once per scan and never changed, so
// the local walk and the ROM check's workers can share one.
type CoreAccess struct {
	idx   *Index
	cores []listedCore
}

type listedCore struct {
	name, stem string
	acc        Access
}

// NewCoreAccess reads the access of every core the catalogue's arcade rows
// name. idx is the card's core index, which decides which of them an MRA
// loads; nil leaves only the names to go by.
func NewCoreAccess(catalogue []data.Row, idx *Index) *CoreAccess {
	ca := &CoreAccess{idx: idx}
	at := map[string]int{}
	for i := range catalogue {
		r := &catalogue[i]
		if !r.IsArcade() || r.Core == "" {
			continue
		}
		name := strings.ToLower(r.Core)
		acc := Access{Known: true, Beta: r.Beta}
		if r.Beta {
			acc.Gate = strings.ToLower(r.Gate)
		}
		j, ok := at[name]
		if !ok {
			at[name] = len(ca.cores)
			ca.cores = append(ca.cores, listedCore{name, coreStem(name), acc})
			continue
		}
		if ca.cores[j].acc != acc {
			ca.cores[j].acc = Access{}
		}
	}
	return ca
}

// Of is the access of the core an MRA naming rbf loads. With that core on
// the card, the catalogue cores that count are the ones resolving to the
// same file, as claimShared decides a shared MRA; without it, the ones
// named alike (sameCore). Several that disagree, or none, are unknown.
func (ca *CoreAccess) Of(rbf string) Access {
	if ca == nil || rbf == "" {
		return Access{}
	}
	stem := coreStem(rbf)
	file, installed := ca.idx.mainRBF(rbf)
	var out Access
	n := 0
	for _, c := range ca.cores {
		if !sameStem(c.stem, stem) {
			continue
		}
		if installed {
			if f, ok := ca.idx.lookupFor(true, c.name); !ok || f.Path != file.Path {
				continue
			}
		}
		if n > 0 && c.acc != out {
			return Access{}
		}
		out, n = c.acc, n+1
	}
	return out
}
