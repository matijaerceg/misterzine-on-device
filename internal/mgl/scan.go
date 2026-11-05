package mgl

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

const MaxEntries = 100000
const MaxFiles = 20000
const MaxDepth = 8

type Entry struct {
	Path     string    `json:"path"`
	Decision string    `json:"decision"` // included, candidate, excluded, invalid
	Reason   string    `json:"reason"`
	GameID   string    `json:"game_id,omitempty"`
	Document *Document `json:"document,omitempty"`
}

type Result struct {
	Entries      []Entry             `json:"entries"`
	Rows         []data.Row          `json:"rows"`
	Alternatives map[string][]string `json:"alternatives"`
	Problems     []string            `json:"problems"`
}

// Scan reads only menu folders (underscore-prefixed at every level), plus
// root-level MGLs. It never crawls games/ or follows symlinks. All paths in
// the result are card-relative; file targets remain uninterpreted XML data.
// An identity match is a naming match, not ROM or BIOS validation.
func Scan(card string, catalogue *Catalogue) (Result, error) {
	result := Result{Entries: []Entry{}, Rows: []data.Row{}, Alternatives: map[string][]string{}, Problems: []string{}}
	root, err := os.OpenRoot(card)
	if err != nil {
		return result, err
	}
	defer root.Close()
	visited, files := 0, 0
	seen := map[string]int{}
	err = fs.WalkDir(root.FS(), ".", func(p string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Problems = append(result.Problems, fmt.Sprintf("%s: %v", p, walkErr))
			return nil
		}
		visited++
		if visited > MaxEntries {
			return errors.New("menu scan entry limit reached")
		}
		if e.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if e.IsDir() {
			if p == "." {
				return nil
			}
			if !strings.HasPrefix(e.Name(), "_") {
				return fs.SkipDir
			}
			if strings.Count(p, "/") >= MaxDepth {
				result.Problems = append(result.Problems, p+": depth limit reached")
				return fs.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(path.Ext(p), ".mgl") {
			return nil
		}
		files++
		if files > MaxFiles {
			return errors.New("MGL file limit reached")
		}
		entry := Entry{Path: p, Decision: "invalid"}
		info, err := e.Info()
		if err != nil {
			entry.Reason = err.Error()
		} else if !info.Mode().IsRegular() {
			entry.Reason = "not a regular file"
		} else {
			f, openErr := root.Open(p)
			if openErr != nil {
				entry.Reason = openErr.Error()
			} else {
				d, parseErr := Parse(f)
				f.Close()
				if parseErr != nil {
					entry.Reason = parseErr.Error()
				} else {
					entry.Document = &d
					ok, reason := d.Cartridge()
					entry.Decision, entry.Reason = "excluded", reason
					if ok {
						entry.Decision, entry.Reason = "candidate", "Neo Geo cartridge; arcade identity not in the supplied catalogue"
						if g, matched := catalogue.Match(d); matched {
							entry.Decision, entry.GameID = "included", g.ID
							entry.Reason = "arcade identity matched by target name; ROMs, BIOS and core not checked"
							// Keep different setnames, cores or load instructions
							// as separate entries: they can have different configs.
							key := identity(g.ID, d)
							if _, exists := seen[key]; exists {
								result.Alternatives[key] = append(result.Alternatives[key], p)
							} else {
								seen[key] = len(result.Rows)
								result.Rows = append(result.Rows, data.Row{K: key, Title: g.Title, Base: "Arcade", Src: data.SrcLocal, Core: path.Base(d.RBF), MRA: p, SN: g.ID})
							}
						}
					}
				}
			}
		}
		result.Entries = append(result.Entries, entry)
		return nil
	})
	if err != nil {
		result.Problems = append(result.Problems, err.Error())
	}
	return result, err
}

// A renamed shortcut keeps its key. Only byte-identical shortcuts coalesce:
// even unrecognised options/reset instructions can change launch behaviour.
func identity(game string, d Document) string {
	return "local:neogeo:" + game + ":" + d.Fingerprint
}
