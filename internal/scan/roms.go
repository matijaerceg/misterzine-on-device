package scan

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NewROMCheck checks archive presence, not archive contents or playability.
// Details may reuse a recent result; pressing Start always checks again.
func NewROMCheck(card string) func(string, bool) string {
	type result struct {
		at    time.Time
		issue string
	}
	cache := map[string]result{}
	return func(rel string, fresh bool) string {
		if !strings.EqualFold(filepath.Ext(rel), ".mra") {
			return ""
		}
		if r, ok := cache[rel]; ok && !fresh && time.Since(r.at) < 3*time.Second {
			return r.issue
		}
		issue := checkROMs(card, rel)
		cache[rel] = result{time.Now(), issue}
		return issue
	}
}

// arcadeROMRoot follows Main's findGamesDir precedence. Main selects one
// directory; it does not combine archives from different storage devices.
func arcadeROMRoot(card string) string {
	var roots []string
	for i := 0; i < 6; i++ {
		roots = append(roots, filepath.Join(filepath.Dir(card), fmt.Sprintf("usb%d", i)))
	}
	roots = append(roots, filepath.Join(filepath.Dir(card), "network"), filepath.Join(card, "cifs"), card)
	for _, root := range roots {
		for _, p := range []string{root, filepath.Join(root, "games")} {
			if st, err := os.Stat(filepath.Join(p, "mame")); err == nil && st.IsDir() {
				return p
			}
		}
	}
	return filepath.Join(card, "_Arcade")
}

func checkROMs(card, rel string) string {
	f, err := os.Open(filepath.Join(card, filepath.FromSlash(rel)))
	if err != nil {
		return "Cannot read game menu file"
	}
	defer f.Close()
	dec := xml.NewDecoder(&commentStripper{br: bufio.NewReader(f)})
	dec.Strict = false
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	type section struct {
		index  string
		groups [][]string
	}
	var sections []section
	current := -1
	zip := ""
	attr := func(el xml.StartElement, key string) string {
		for _, a := range el.Attr {
			if strings.EqualFold(a.Name.Local, key) {
				return a.Value
			}
		}
		return ""
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "Cannot read game ROM requirements"
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch strings.ToLower(el.Name.Local) {
			case "rom":
				sections = append(sections, section{index: attr(el, "index")})
				current = len(sections) - 1
				zip = attr(el, "zip")
			case "part":
				if current < 0 || attr(el, "name") == "" {
					continue
				}
				z := attr(el, "zip")
				if z == "" {
					z = zip
				}
				if z != "" {
					sections[current].groups = append(sections[current].groups, strings.Split(z, "|"))
				}
			}
		case xml.EndElement:
			if strings.EqualFold(el.Name.Local, "rom") {
				current = -1
			}
		}
	}
	root := arcadeROMRoot(card)
	present := map[string]bool{}
	missing := func(s section) string {
		for _, group := range s.groups {
			found := false
			for _, z := range group {
				z = strings.TrimSpace(z)
				if z == "" {
					continue
				}
				p := filepath.Join(root, "mame", filepath.FromSlash(z))
				if strings.HasPrefix(z, "/") {
					p = filepath.Join(root, filepath.FromSlash(strings.TrimLeft(z, "/")))
				}
				// Main also accepts an extracted archive directory.
				exists, checked := present[p]
				if !checked {
					_, err := os.Stat(p)
					exists = err == nil
					present[p] = exists
				}
				if exists {
					found = true
					break
				}
			}
			if !found {
				return "Missing game ROM: " + strings.Join(group, " or ")
			}
		}
		return ""
	}
	// Multiple index-0 sections are alternative ROM layouts in Main. One
	// satisfiable layout suffices; later sections are suppressed after success.
	zeroSeen, zeroOK, zeroIssue := false, false, ""
	for _, s := range sections {
		issue := missing(s)
		if s.index == "0" {
			zeroSeen = true
			if issue == "" {
				zeroOK = true
			} else if zeroIssue == "" {
				zeroIssue = issue
			}
		} else if issue != "" {
			return issue
		}
	}
	if zeroSeen && !zeroOK {
		return zeroIssue
	}
	return ""
}
