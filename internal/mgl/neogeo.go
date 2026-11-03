package mgl

import (
	"bytes"
	_ "embed"
	"sync"
)

//go:embed neogeo-arcade.json
var neoGeoJSON []byte

var neoGeoCatalogue = sync.OnceValue(func() *Catalogue {
	c, err := ReadCatalogue(bytes.NewReader(neoGeoJSON))
	if err != nil {
		panic(err) // Invalid checked-in build data, never a user's input.
	}
	return c
})

// NeoGeoCatalogue is the shared arcade catalogue used by the device importer.
// The matching diagnostic JSON is shipped in the test package.
func NeoGeoCatalogue() *Catalogue { return neoGeoCatalogue() }
