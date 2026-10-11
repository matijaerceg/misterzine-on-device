package data

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// The local rows the last complete card scan made outlive the run, so the
// next start lists them from its first frame: on a big card the scan that
// finds them takes 15 seconds or more, and a list that grows by hundreds of
// games once it ends reads as one that forgot them. The scan still runs and
// replaces them when the card has changed.
//
// LocalRowsVersion changes whenever a saved row would no longer mean what it
// did; a file of another version is ignored and the next scan rewrites it.
const LocalRowsVersion = 1

type localRowsFile struct {
	V int `json:"v"`
	// Hash is the catalogue the rows were resolved against. Rows of another
	// catalogue still go through MergeLocal, which drops the games it lists.
	Hash string `json:"hash"`
	Rows []Row  `json:"rows"`
}

// EncodeLocalRows is the saved form of a scan's local rows.
func EncodeLocalRows(hash string, local []Row) ([]byte, error) {
	if local == nil {
		local = []Row{}
	}
	return json.Marshal(localRowsFile{V: LocalRowsVersion, Hash: hash, Rows: local})
}

// DecodeLocalRows reads what EncodeLocalRows wrote: the catalogue hash the
// rows were resolved against, and the rows. A file of another version is an
// error, as is one that does not decode.
func DecodeLocalRows(b []byte) (string, []Row, error) {
	var f localRowsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return "", nil, fmt.Errorf("local rows: %w", err)
	}
	if f.V != LocalRowsVersion {
		return "", nil, fmt.Errorf("local rows: version %d, this build reads %d", f.V, LocalRowsVersion)
	}
	return f.Hash, f.Rows, nil
}

// SameLocalRows reports whether two sets of local rows are alike field for
// field. Generation looks only at what positional statuses depend on, so
// rows read back from the last run can match a scan's by generation and
// still carry a title the MRA has changed since, or lack a field this build
// fills in.
func SameLocalRows(a, b []Row) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	ja, err := json.Marshal(a)
	if err != nil {
		return false
	}
	jb, err := json.Marshal(b)
	return err == nil && bytes.Equal(ja, jb)
}
