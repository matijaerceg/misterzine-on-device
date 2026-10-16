// Package controls stores MisterZine-only controller preferences. MiSTer's
// global maps and in-game controls are never written here.
package controls

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
	"github.com/matijaerceg/misterzine-on-device/internal/store"
)

// Legacy styles are accepted only to preserve existing profiles during conversion.
var legacyStyles = []string{"default", "mister", "xbox", "playstation", "numbers", "neogeo", "custom"}
var actionKeys = map[string]platform.Key{"start": platform.KeyStart, "enter": platform.KeyEnter, "back": platform.KeyBack, "tab": platform.KeyTab, "space": platform.KeySpace, "pageup": platform.KeyPageUp, "pagedown": platform.KeyPageDown, "select": platform.KeySelect, "menu": platform.KeyMenu, "up": platform.KeyUp, "down": platform.KeyDown, "left": platform.KeyLeft, "right": platform.KeyRight}

func Key(id string) platform.Key { return actionKeys[id] }

func ValidCode(c uint16) bool { return c > 0 && c < 0x380 && c != 88 }

var identity = regexp.MustCompile(`^[0-9a-f]{4}_[0-9a-f]{4}(_[0-9a-f]{16})?$`)

func (f File) Validate() error {
	if f.Schema != 1 {
		return errors.New("unsupported controls file version")
	}
	if len(f.Profiles) > 64 {
		return errors.New("too many controller profiles")
	}
	for id, p := range f.Profiles {
		if !identity.MatchString(id) || id == "0000_0000" {
			return errors.New("invalid controller identity")
		}
		if len(p.Name) > 256 {
			return errors.New("controller name too long")
		}
		if p.Style != "" {
			found := false
			for _, s := range legacyStyles {
				found = found || p.Style == s
			}
			if !found {
				return errors.New("unknown label style")
			}
		}
		if p.NameBase != "" {
			valid := false
			for _, style := range legacyStyles {
				valid = valid || (style == p.NameBase && style != "custom")
			}
			if !valid {
				return errors.New("unknown base name preset")
			}
		}
		if len(p.Names) > 896 {
			return errors.New("too many button names")
		}
		for code, name := range p.Names {
			if !ValidCode(code) || len(name) < 1 || len(name) > 8 || strings.IndexFunc(name, func(r rune) bool { return r < 32 || r > 126 }) >= 0 {
				return errors.New("invalid button name")
			}
		}
		if p.MappingVersion != 0 && p.MappingVersion != 2 {
			return errors.New("unsupported input mapping version")
		}
		if p.MappingVersion == 0 && (len(p.Inputs) > 0 || p.BasicsComplete) {
			return errors.New("unversioned input mapping")
		}
		for code, id := range p.Inputs {
			if !ValidCode(code) || Key(id) == platform.KeyNone {
				return errors.New("invalid input assignment")
			}
		}
		if p.MappingVersion == 2 && p.BasicsComplete {
			if err := ValidateInputs(p.Inputs); err != nil {
				return err
			}
		}
		seen := map[uint16]bool{}
		for a, c := range p.Bindings {
			if Key(a) == platform.KeyNone || (c != 0 && !ValidCode(c)) {
				return errors.New("invalid button assignment")
			}
			if c != 0 && seen[c] {
				return errors.New("a button is assigned twice")
			}
			seen[c] = true
		}
		for a, l := range p.Labels {
			if Key(a) == platform.KeyNone || len(l) != 1 || !((l[0] >= 'A' && l[0] <= 'Z') || (l[0] >= '0' && l[0] <= '9')) {
				return errors.New("labels must be one letter or number")
			}
		}
	}
	return nil
}

func Load(path string) (File, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return New(), err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return New(), err
	}
	if len(b) > 65536 {
		return New(), errors.New("controls file is too large")
	}
	var v File
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&v); err != nil {
		return New(), err
	}
	if d.Decode(new(any)) != io.EOF {
		return New(), errors.New("extra data in controls file")
	}
	if err = v.Validate(); err != nil {
		return New(), err
	}
	if v.Profiles == nil {
		v.Profiles = map[string]Profile{}
	}
	return v, nil
}
func Save(path string, f File) error {
	if err := f.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		return err
	}
	if len(b) > 65536 {
		return errors.New("controls file is too large")
	}
	return store.Save(path, f)
}

// PanelBase follows the common MAME keyboard-encoder layout. It is only
// enabled by the player, never by a guessed USB ID.
func PanelBase() map[string]uint16 {
	return map[string]uint16{"start": 2, "enter": 29, "back": 56, "tab": 57, "space": 42, "pageup": 44, "pagedown": 45, "select": 6, "menu": 1, "up": 103, "down": 108, "left": 105, "right": 106}
}

func Effective(base map[string]uint16, p Profile) map[string]uint16 {
	m := map[string]uint16{}
	for k, v := range base {
		m[k] = v
	}
	// Overrides claim their codes before inherited actions, so an external
	// MiSTer redefine cannot give one physical button two actions.
	for k, v := range p.Bindings {
		for other, c := range m {
			if v != 0 && c == v && other != k {
				m[other] = 0
			}
		}
		m[k] = v
	}
	return m
}

// ValidateInputs allows multiple physical inputs per action, but requires navigation.
func ValidateInputs(inputs map[uint16]string) error {
	found := map[string]bool{}
	for code, id := range inputs {
		if !ValidCode(code) || Key(id) == platform.KeyNone {
			return errors.New("invalid input assignment")
		}
		found[id] = true
	}
	for _, id := range []string{"right", "left", "up", "down", "enter", "back"} {
		if !found[id] {
			return fmt.Errorf("Set %s first", id)
		}
	}
	return nil
}
