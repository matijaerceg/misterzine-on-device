// mzaccess exports the exact feature register used by the app.
package main

import (
	"encoding/json"
	"github.com/matijaerceg/misterzine-on-device/internal/access"
	"os"
)

func main() {
	if err := json.NewEncoder(os.Stdout).Encode(access.Catalog()); err != nil {
		panic(err)
	}
}
