package scan

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// ROMListName is the ROM problem list's file in MisterZine's folder.
const ROMListName = "rom_problems.txt"

// WriteList puts the stored answers for paths in the file at path, as the
// list a player takes to their ROM collection: the zip files to add or
// replace, one to a line, then each MRA with a problem and what it is.
// paths are the card-relative files the sweep was given; one without an
// answer, or other than an MRA, is left out. The file is written only when
// its text changes, and removed when no MRA was checked. A sweep calls it
// from its done function, so two may overlap; they take turns.
func (c *ROMCheck) WriteList(path string, paths []string) error {
	found := map[string]ROMResult{}
	checked := 0
	for _, rel := range paths {
		if !strings.EqualFold(filepath.Ext(rel), ".mra") {
			continue
		}
		if res, ok := c.Known(rel); ok {
			checked++
			if res.Text != "" {
				found[rel] = res
			}
		}
	}
	c.list.Lock()
	defer c.list.Unlock()
	if checked == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	text := romListText(found, checked)
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, text) {
		return nil
	}
	return store.WriteAtomic(path, text)
}

// romListText words the list for checked MRAs, of which found have a
// problem. The zip names are sorted and each given once, in the spelling
// met first: MiSTer's card ignores case.
func romListText(found map[string]ROMResult, checked int) []byte {
	var b strings.Builder
	if len(found) == 0 {
		b.WriteString("MisterZine ROM check: no problems in " + strconv.Itoa(checked) + " MRA files.\n")
		b.WriteString("Written after each card scan.\n")
		return []byte(b.String())
	}
	rels := make([]string, 0, len(found))
	for rel := range found {
		rels = append(rels, rel)
	}
	sort.Slice(rels, func(i, j int) bool {
		a, b := strings.ToLower(rels[i]), strings.ToLower(rels[j])
		return a < b || a == b && rels[i] < rels[j]
	})
	zips := map[string]string{} // lowercase name -> as first met
	for _, rel := range rels {
		for _, z := range strings.Split(found[rel].Zips, "|") {
			if k := strings.ToLower(z); z != "" && zips[k] == "" {
				zips[k] = z
			}
		}
	}
	keys := make([]string, 0, len(zips))
	for k := range zips {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b.WriteString("MisterZine ROM check: " + strconv.Itoa(len(found)) + " of " + strconv.Itoa(checked) + " MRA files have a ROM problem.\n")
	b.WriteString("Written after each card scan: the zip files first, then each MRA and its problem.\n")
	if len(keys) > 0 {
		b.WriteString("\nZip files to add or replace (" + strconv.Itoa(len(keys)) + "):\n")
		for _, k := range keys {
			b.WriteString(zips[k] + "\n")
		}
	}
	b.WriteString("\nMRA files (" + strconv.Itoa(len(rels)) + "):\n")
	for _, rel := range rels {
		b.WriteString(filepath.ToSlash(rel) + "\n  " + found[rel].Text + "\n")
	}
	return []byte(b.String())
}
