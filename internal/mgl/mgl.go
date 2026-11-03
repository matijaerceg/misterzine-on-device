// Package mgl is the isolated Neo Geo shortcut import prototype. It never
// rewrites launchers, opens their ROM targets, or treats an MGL as an MRA.
package mgl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

const MaxBytes = 64 << 10

type File struct {
	Path  string `xml:"path,attr" json:"path"`
	Type  string `xml:"type,attr" json:"type"`
	Index string `xml:"index,attr" json:"index"`
	Delay string `xml:"delay,attr" json:"delay"`
}

type Setname struct {
	Name    string `xml:",chardata" json:"name"`
	SameDir string `xml:"same_dir,attr" json:"same_dir,omitempty"`
}

type Document struct {
	RBF             string  `json:"rbf"`
	Setname         Setname `json:"setname"`
	Files           []File  `json:"files"`
	LegacyAmpersand bool    `json:"legacy_ampersand,omitempty"`
	Fingerprint     string  `json:"fingerprint"`
}

// Legacy generators interpolated filenames into attributes without escaping
// bare ampersands. Repair only those ampersands; leave all other XML strict.
// The original file is always the launch target, never this parsing copy.
var entity = regexp.MustCompile(`^&(?:amp|lt|gt|quot|apos|#[0-9]+|#x[0-9a-fA-F]+);`)

func Parse(r io.Reader) (Document, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Document{}, err
	}
	if len(b) > MaxBytes {
		return Document{}, errors.New("MGL exceeds 64 KiB")
	}
	var repaired bytes.Buffer
	legacy := false
	for i := 0; i < len(b); i++ {
		if b[i] == '&' && !entity.Match(b[i:]) {
			repaired.WriteString("&amp;")
			legacy = true
		} else {
			repaired.WriteByte(b[i])
		}
	}
	d := xml.NewDecoder(bytes.NewReader(repaired.Bytes()))
	sum := sha256.Sum256(b)
	out := Document{LegacyAmpersand: legacy, Fingerprint: hex.EncodeToString(sum[:])}
	depth, roots, rbfs, sets := 0, 0, 0, 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Document{}, fmt.Errorf("invalid MGL XML: %w", err)
		}
		switch v := t.(type) {
		case xml.Directive:
			return Document{}, errors.New("XML directives are not supported")
		case xml.StartElement:
			if v.Name.Space != "" {
				return Document{}, errors.New("XML namespaces are not supported")
			}
			attrs := map[xml.Name]bool{}
			for _, attr := range v.Attr {
				if attrs[attr.Name] {
					return Document{}, errors.New("duplicate XML attribute")
				}
				attrs[attr.Name] = true
			}
			if depth == 0 {
				roots++
				if roots != 1 || v.Name.Local != "mistergamedescription" {
					return Document{}, errors.New("expected one mistergamedescription root")
				}
				depth++
				continue
			}
			// Consume each complete direct child so nested file/rbf tags in
			// an unrelated extension cannot masquerade as launch instructions.
			switch v.Name.Local {
			case "rbf":
				rbfs++
				err = d.DecodeElement(&out.RBF, &v)
			case "setname":
				sets++
				err = d.DecodeElement(&out.Setname, &v)
			case "file":
				var f File
				err = d.DecodeElement(&f, &v)
				out.Files = append(out.Files, f)
			default:
				// reset/options belong to MiSTer. Keep the original MGL so
				// importing a shortcut cannot silently lose these settings.
				err = d.Skip()
			}
			if err != nil {
				return Document{}, fmt.Errorf("invalid MGL element: %w", err)
			}
		case xml.EndElement:
			depth--
		case xml.CharData:
			if strings.TrimSpace(string(v)) != "" {
				return Document{}, errors.New("unexpected text outside MGL elements")
			}
		}
	}
	out.RBF = strings.TrimSpace(out.RBF)
	out.Setname.Name = strings.TrimSpace(out.Setname.Name)
	if roots != 1 || depth != 0 || rbfs != 1 || out.RBF == "" || sets > 1 {
		return Document{}, errors.New("MGL needs one core and at most one setname")
	}
	if badPath(out.RBF) || badPath(out.Setname.Name) {
		return Document{}, errors.New("invalid core or setname")
	}
	for _, f := range out.Files {
		if f.Path == "" || badPath(f.Path) {
			return Document{}, errors.New("invalid game path")
		}
	}
	return out, nil
}

func badPath(s string) bool { return strings.ContainsAny(s, "\\\x00\r\n") }

var datedCore = regexp.MustCompile(`_[0-9]{8}[a-z]?$`)

// Cartridge identifies the loading convention, not the game's historical
// arcade provenance. MVS and AES use the same cartridge slot.
func (d Document) Cartridge() (bool, string) {
	core := strings.ToLower(path.Base(d.RBF))
	core = strings.TrimSuffix(core, ".rbf")
	core = datedCore.ReplaceAllString(core, "")
	switch core {
	case "neogeo", "neogeo_llapi", "neogeoyc":
	default:
		return false, "core is outside the supported Neo Geo variants"
	}
	if len(d.Files) != 1 {
		return false, "requires exactly one game target (core-only and multi-file shortcuts excluded)"
	}
	f := d.Files[0]
	if f.Type != "f" || f.Index != "1" {
		return false, "not the Neo Geo cartridge slot (CD/mounted media excluded)"
	}
	ext := strings.ToLower(path.Ext(strings.TrimRight(f.Path, "/")))
	if ext != "" && ext != ".neo" && ext != ".zip" && ext != ".rom" {
		// A reviewed extracted-set directory can itself contain a period.
		// This does not admit arbitrary extensions or CD image filenames.
		base := strings.ToLower(path.Base(strings.TrimRight(f.Path, "/")))
		if _, knownDirectory := NeoGeoCatalogue().names[base]; !knownDirectory {
			return false, "unsupported cartridge target extension"
		}
	}
	return true, "Neo Geo cartridge shortcut"
}
