//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/mgl"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

const pilotRoot = "/media/fat/misterzine-neogeo-test"

func pilotCatalogue() *mgl.Catalogue {
	return mgl.NeoGeoCatalogue()
}

func (h *host) scanPilot() (mgl.Result, error) {
	result, err := mgl.Scan(h.card, pilotCatalogue())
	if len(result.Problems) > 0 {
		err = errors.Join(err, errors.New(strings.Join(result.Problems, "; ")))
	}
	b, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr == nil {
		marshalErr = store.WriteAtomic(filepath.Join(h.root, "mgl-report.json"), b)
	}
	return result, errors.Join(err, marshalErr)
}

// Re-read the shortcut before handing the unchanged file to Main. A shortcut
// edited since discovery must still satisfy the pilot's core and title rules.
func validatePilotLaunch(card, target string) error {
	root, err := os.OpenRoot(card)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(target)
	if err != nil {
		return err
	}
	defer f.Close()
	d, err := mgl.Parse(f)
	if err != nil {
		return err
	}
	if ok, reason := d.Cartridge(); !ok {
		return errors.New(reason)
	}
	if _, ok := pilotCatalogue().Match(d); !ok {
		return errors.New("game target is not recognised in the Neo Geo arcade catalogue")
	}
	if !mgl.CorePresent(card, d) {
		return errors.New("Neo Geo core missing at shortcut's core path")
	}
	return nil
}
