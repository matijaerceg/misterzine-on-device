// Package data holds the release rows and every rule the website applies to
// them: sorting, labels, relative times, the last-look marker, notices and
// filters. It is pure: no I/O beyond decoding JSON, no platform code, so the
// same package runs on the device, in the PC harness and in tests.
package data

import (
	"encoding/json"
	"fmt"
	"io"
)

// Row is one release row exactly as docs/releases/data.json emits it. Field
// names follow the JSON keys; see the site's _web_row in misterzine.py for
// the authority on what each means.
type Row struct {
	Title        string   `json:"title"`
	Base         string   `json:"base"` // Arcade, Console, Computer, Other
	Genre        string   `json:"genre"`
	Date         string   `json:"date"`      // ISO debut date
	DateKind     string   `json:"date_kind"` // debut (build is dormant)
	Year         string   `json:"year"`
	Manufacturer string   `json:"manufacturer"`
	Core         string   `json:"core"` // rbf base name
	Deprecated   bool     `json:"deprecated"`
	Updated      string   `json:"updated"` // ISO date of the newest shipped build
	BD           string   `json:"bd"`      // ISO build date of the shipped rbf itself; "" when undated or unknown
	BH           string   `json:"bh"`      // md5 of the shipped rbf, the value update_all's db carries
	B            int      `json:"b"`       // refresh run that first shipped this Updated value
	K            string   `json:"k"`       // stable deep-link key
	Src          string   `json:"src"`
	FT           string   `json:"ft"`
	Repo         string   `json:"repo"`
	MRA          string   `json:"mra"` // card-relative path, arcade only
	Img          string   `json:"img"` // screenshot file stem
	ImgSlots     []string `json:"img_slots"`
	ImgW         int      `json:"img_w"`
	ImgH         int      `json:"img_h"`
	SN           string   `json:"sn"` // MAME setname
	Rot          string   `json:"rot"`
	Plr          string   `json:"plr"`
	Ctl          string   `json:"ctl"`
	Buttons      *int     `json:"buttons,omitempty"` // nil = unknown; zero is an explicit count
	Reg          string   `json:"reg"`
	Res          string   `json:"res"`
	Act          string   `json:"act"` // latest commit, only when != updated
	Flip         string   `json:"flip"`
	MT           string   `json:"mt"`
	Spc          string   `json:"spc"`
	Prov         []string `json:"prov"`
	Beta         bool     `json:"beta"`
	Brot         string   `json:"brot"`
	Note         string   `json:"note"`
	Scr          int      `json:"scr"`
}

// Meta is docs/releases/meta.json.
type Meta struct {
	Updated string `json:"updated"` // "2026-09-08T00:59Z"
	Hash    string `json:"hash"`    // sha256 of data.json
	Rows    int    `json:"rows"`
}

// DecodeRows reads a data.json array.
func DecodeRows(r io.Reader) ([]Row, error) {
	var rows []Row
	dec := json.NewDecoder(r)
	if err := dec.Decode(&rows); err != nil {
		return nil, fmt.Errorf("data.json: %w", err)
	}
	return rows, nil
}

// DecodeMeta reads a meta.json object.
func DecodeMeta(r io.Reader) (Meta, error) {
	var m Meta
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return Meta{}, fmt.Errorf("meta.json: %w", err)
	}
	return m, nil
}

// HasProv reports whether the named field is provisional on this row.
func (r *Row) HasProv(field string) bool {
	for _, p := range r.Prov {
		if p == field {
			return true
		}
	}
	return false
}

// IsArcade reports whether the row is an arcade game (as opposed to a core).
func (r *Row) IsArcade() bool { return r.Base == "Arcade" }

// RotGroup buckets the MAD rotation string for the rotation filter:
// "h" horizontal, "v" vertical, "" unknown.
func (r *Row) RotGroup() string {
	switch {
	case len(r.Rot) >= 8 && r.Rot[:8] == "Vertical":
		return "v"
	case len(r.Rot) >= 10 && r.Rot[:10] == "Horizontal":
		return "h"
	}
	return ""
}
