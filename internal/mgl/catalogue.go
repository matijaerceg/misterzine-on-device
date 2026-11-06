package mgl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// Game is a reviewed arcade identity. Aliases are exact target basenames,
// without extensions; the shortcut filename and setname are not evidence.
type Game struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Aliases  []string `json:"aliases"`
	SetNames []string `json:"set_names,omitempty"` // additional MAME and MiSTer set identifiers
}

type Catalogue struct {
	names map[string]Game
	sets  map[string]Game
}

func ReadCatalogue(r io.Reader) (*Catalogue, error) {
	b, err := io.ReadAll(io.LimitReader(r, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 2<<20 {
		return nil, errors.New("arcade catalogue exceeds 2 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var games []Game
	if err := d.Decode(&games); err != nil {
		return nil, err
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return nil, errors.New("unexpected trailing catalogue data")
	}
	c := &Catalogue{names: map[string]Game{}, sets: map[string]Game{}}
	ids := map[string]bool{}
	for _, g := range games {
		if g.ID == "" || g.Title == "" || ids[g.ID] || strings.ContainsAny(g.ID, "/\\: \t\r\n") {
			return nil, fmt.Errorf("invalid or repeated game identity %q", g.ID)
		}
		ids[g.ID] = true
		sets := append([]string{g.ID}, g.SetNames...)
		for _, name := range sets {
			c.sets[strings.ToLower(name)] = g
		}
		names := append(sets, g.Title)
		for _, name := range append(names, g.Aliases...) {
			key := strings.ToLower(name)
			if name == "" || strings.ContainsAny(name, "/\\\r\n\x00") {
				return nil, fmt.Errorf("invalid alias %q", name)
			}
			if old, ok := c.names[key]; ok && old.ID != g.ID {
				return nil, fmt.Errorf("ambiguous game alias %q", name)
			}
			c.names[key] = g
		}
	}
	return c, nil
}

func (c *Catalogue) Match(d Document) (Game, bool) {
	if c == nil || len(d.Files) != 1 {
		return Game{}, false
	}
	return c.matchTarget(d.Files[0].Path)
}

func (c *Catalogue) matchTarget(target string) (Game, bool) {
	name := strings.ToLower(path.Base(strings.TrimRight(target, "/")))
	// Extracted-set directory names can contain periods (Neo Mr. Do!,
	// SNK vs. Capcom). Strip only a supported file extension.
	switch ext := path.Ext(name); ext {
	case ".neo", ".rom", ".zip":
		name = strings.TrimSuffix(name, ext)
	default:
		g, ok := c.names[name]
		return g, ok
	}
	if g, ok := c.names[name]; ok {
		return g, true
	}
	// In "Title (set)", only the final group can identify the cartridge.
	stem, closed := strings.CutSuffix(name, ")")
	i := strings.LastIndex(stem, " (")
	if !closed || i <= 0 || strings.TrimSpace(stem[:i]) == "" {
		return Game{}, false
	}
	g, ok := c.sets[stem[i+2:]]
	return g, ok
}
