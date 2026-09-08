// Package snapshot embeds a copy of data.json so the very first run without
// a network still shows a list. Refresh it with tools/snapshot.sh before a
// release; the app replaces it with the live data on the first successful
// fetch.
package snapshot

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

//go:embed data.json.gz
var dataGz []byte

//go:embed meta.json
var metaJSON []byte

// Load decodes the embedded rows and meta.
func Load() ([]data.Row, data.Meta, error) {
	zr, err := gzip.NewReader(bytes.NewReader(dataGz))
	if err != nil {
		return nil, data.Meta{}, err
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, data.Meta{}, err
	}
	rows, err := data.DecodeRows(bytes.NewReader(raw))
	if err != nil {
		return nil, data.Meta{}, err
	}
	meta, err := data.DecodeMeta(bytes.NewReader(metaJSON))
	if err != nil {
		return nil, data.Meta{}, err
	}
	return rows, meta, nil
}

// Raw returns the embedded data.json bytes (uncompressed), to seed the cache.
func Raw() ([]byte, []byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(dataGz))
	if err != nil {
		return nil, nil, err
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, nil, err
	}
	return raw, metaJSON, nil
}
