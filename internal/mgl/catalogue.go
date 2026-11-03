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
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Aliases []string `json:"aliases"`
}

type Catalogue struct{ names map[string]Game }

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
	c := &Catalogue{names: map[string]Game{}}
	ids := map[string]bool{}
	for _, g := range games {
		if g.ID == "" || g.Title == "" || ids[g.ID] || strings.ContainsAny(g.ID, "/\\: \t\r\n") {
			return nil, fmt.Errorf("invalid or repeated game identity %q", g.ID)
		}
		ids[g.ID] = true
		for _, name := range append([]string{g.ID, g.Title}, g.Aliases...) {
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
	name := path.Base(strings.TrimRight(d.Files[0].Path, "/"))
	name = strings.TrimSuffix(name, path.Ext(name))
	g, ok := c.names[strings.ToLower(name)]
	return g, ok
}
