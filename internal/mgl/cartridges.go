package mgl

import (
	"archive/zip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// A row carries the card-relative target, including the path inside a collection ZIP.
func gamePath(target string) (folder, rel string, ok bool) {
	if len(target) > 900 || badPath(target) || strings.IndexFunc(target, unicode.IsControl) >= 0 {
		return "", "", false
	}
	for parent := range platform.GameDirParents {
		rest := target
		if parent != "" {
			var found bool
			if rest, found = strings.CutPrefix(target, parent+"/"); !found {
				continue
			}
		}
		name, rel, ok := strings.Cut(rest, "/")
		if ok && strings.EqualFold(name, "NeoGeo") && fs.ValidPath(rest) {
			return path.Join(parent, name), rel, true
		}
	}
	return "", "", false
}

// GameBackingPath is the file or directory the operating system can stat.
// MiSTer treats a path past the first ZIP as a member of that archive.
func GameBackingPath(target string) (string, bool) {
	folder, rel, ok := gamePath(target)
	if !ok {
		return "", false
	}
	backing, _ := splitArchive(rel)
	return path.Join(folder, backing), true
}

func splitArchive(rel string) (backing, member string) {
	parts := strings.Split(rel, "/")
	for i, part := range parts {
		if strings.EqualFold(path.Ext(part), ".zip") {
			return strings.Join(parts[:i+1], "/"), strings.Join(parts[i+1:], "/")
		}
	}
	return rel, ""
}

// Read only the central directory. Never unpack a collection or open its ROMs.
func readPack(root *os.Root, rel string, visit func(*zip.Reader) error) error {
	f, err := root.OpenFile(rel, packReadFlags, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("game pack is not a regular file")
	}
	z, err := zip.NewReader(f, info.Size())
	if err != nil {
		return err
	}
	if len(z.File) > MaxEntries {
		return errors.New("too many entries in game pack")
	}
	// archive/zip's FS cleans names such as ../mslug.neo. Only use names
	// that MiSTer could open exactly as seen in the original directory.
	for _, entry := range z.File {
		name := strings.TrimSuffix(entry.Name, "/")
		if !fs.ValidPath(name) || name == "." || badPath(name) {
			return errors.New("invalid path in game pack")
		}
	}
	return visit(z)
}

// Darksoft cartridges have their program and graphics ROMs directly in the
// set folder. Checking both keeps CD images, BIOS ZIPs and ordinary folders
// with a familiar game title out of the list.
func cartridgeGame(fsys fs.FS, p, name string, mode fs.FileMode) (Game, bool) {
	g, ok := NeoGeoCatalogue().matchTarget(name)
	if !ok {
		return Game{}, false
	}
	if !mode.IsDir() {
		return g, mode.IsRegular() && strings.EqualFold(path.Ext(name), ".neo")
	}
	entries, err := fs.ReadDir(fsys, p)
	if err != nil {
		return Game{}, false
	}
	var program, graphics bool
	for _, e := range entries {
		if e.Type().IsRegular() {
			switch e.Name() {
			case "prom":
				program = true
			case "crom0":
				graphics = true
			}
		}
	}
	return g, program && graphics
}

// ValidateCartridge re-checks the chosen game, including its member in the
// current ZIP, immediately before creating the launch shortcut.
func ValidateCartridge(card, target string) (string, error) {
	folder, rel, ok := gamePath(target)
	if !ok {
		return "", errors.New("invalid Neo Geo cartridge path")
	}
	if folder != neoGeoFolder(card) {
		return "", errors.New("Neo Geo storage changed; rescan the card")
	}
	root, err := os.OpenRoot(filepath.Join(card, filepath.FromSlash(folder)))
	if err != nil {
		return "", err
	}
	defer root.Close()
	backing, member := splitArchive(rel)
	check := func(fsys fs.FS, p string) error {
		info, err := fs.Stat(fsys, p)
		if err != nil {
			return err
		}
		if _, ok := cartridgeGame(fsys, p, path.Base(rel), info.Mode()); !ok {
			return errors.New("arcade cartridge missing or unrecognised")
		}
		return nil
	}
	if strings.EqualFold(path.Ext(backing), ".zip") {
		err = readPack(root, backing, func(z *zip.Reader) error {
			p := member
			if p == "" {
				p = "."
			}
			if _, nested := splitArchive(member); nested != "" || (member != "" && strings.EqualFold(path.Ext(member), ".zip")) {
				return errors.New("nested game ZIP")
			}
			return check(z, p)
		})
		if err != nil {
			return "", err
		}
	} else if err := check(root.FS(), rel); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(filepath.Join(card, filepath.FromSlash(target)))
	if err != nil {
		return "", err
	}
	if len(abs) > 900 {
		return "", errors.New("Neo Geo game path too long")
	}
	return filepath.ToSlash(abs), nil
}

// A shortcut may use the core's games folder, a custom setname, or the
// common ../../../../../media/fat/... convention. Keep paths on other
// volumes distinct even when they have the same game filename.
func shortcutGamePath(card, folder string, d Document) string {
	file := d.Files[0]
	target := file.Path
	home := datedCore.ReplaceAllString(strings.TrimSuffix(strings.ToLower(path.Base(d.RBF)), ".rbf"), "")
	if d.Setname.Name != "" && d.Setname.SameDir != "1" {
		home = d.Setname.Name
	}
	rootGames := strings.EqualFold(file.Root, "games")
	if rootGames {
		home, target, _ = strings.Cut(strings.TrimLeft(target, "/"), "/")
		target = strings.TrimLeft(target, "/")
	} else if file.Root != "" {
		return ""
	}
	if !path.IsAbs(target) {
		if home == "." || !fs.ValidPath(home) || strings.Contains(home, "/") {
			return ""
		}
		base := folder
		if rootGames || !strings.EqualFold(home, "NeoGeo") {
			var ok bool
			if base, ok = platform.FindGameDir(card, home); !ok {
				base = path.Join("games", home)
			}
		}
		base = path.Join(filepath.ToSlash(card), base)
		target = path.Join(base, target)
	}
	library := path.Join(filepath.ToSlash(card), folder)
	target = path.Clean(target)
	parent, ok := strings.CutPrefix(target, path.Dir(library)+"/")
	name, rel, nested := strings.Cut(parent, "/")
	if !ok || !nested || !strings.EqualFold(name, path.Base(library)) {
		return ""
	}
	if name == path.Base(library) {
		return rel
	}
	// Absolute MGL paths can spell the folder differently on FAT.
	original, err := os.Stat(filepath.FromSlash(library))
	other, otherErr := os.Stat(filepath.Join(filepath.Dir(library), name))
	if err == nil && otherErr == nil && os.SameFile(original, other) {
		return rel
	}
	return ""
}

// The stock core calls its folder NEOGEO. Keep the on-disk spelling in
// targets so upgrades retain existing SD favourites and version choices.
func neoGeoFolder(card string) string {
	selected, found := platform.FindGameDir(card, "NEOGEO")
	if !found {
		return ""
	}
	parent := path.Dir(selected)
	// ReadDir sorts NEOGEO first if the filesystem has multiple spellings.
	entries, _ := os.ReadDir(filepath.Join(card, filepath.FromSlash(parent)))
	for _, e := range entries {
		if strings.EqualFold(e.Name(), "NEOGEO") {
			return path.Join(parent, e.Name())
		}
	}
	return selected // a known name can be traversable even if its parent can't be listed
}

// ScanCartridges is called only for an opted-in card scan. The chosen library
// root may be a NAS mount or link. WalkDir does not follow links inside it.
func ScanCartridges(card string, result *Result) error {
	folder := neoGeoFolder(card)
	if folder == "" {
		return nil
	}
	root, err := os.OpenRoot(filepath.Join(card, filepath.FromSlash(folder)))
	if err != nil {
		return err
	}
	defer root.Close()
	shortcuts := map[string]bool{}
	for _, entry := range result.Entries {
		if entry.Decision == "included" && entry.Document != nil {
			shortcuts[shortcutGamePath(card, folder, *entry.Document)] = true
		}
	}
	return walkLibrary(root, folder, result, shortcuts)
}

func walkLibrary(root *os.Root, folder string, result *Result, shortcuts map[string]bool) error {
	add := func(rel string, game Game) {
		if shortcuts[rel] {
			return
		}
		target := path.Join(folder, rel)
		if _, _, ok := gamePath(target); !ok {
			return
		}
		key := fmt.Sprintf("local:neogeo:file:%x", sha256.Sum256([]byte(target)))
		result.Rows = append(result.Rows, game.row(key, "NeoGeo", target))
	}
	limit := errors.New("Neo Geo game folder scan limit reached")
	count := 0
	var walk func(fs.FS, string) error
	walk = func(fsys fs.FS, archive string) error {
		return fs.WalkDir(fsys, ".", func(p string, entry fs.DirEntry, err error) error {
			if err != nil {
				result.Problems = append(result.Problems, fmt.Sprintf("%s: %v", path.Join(folder, archive, p), err))
				return nil
			}
			count++
			if count > MaxEntries {
				return limit
			}
			if p == "." || entry.Type()&fs.ModeSymlink != 0 {
				return nil
			}
			rel := path.Join(archive, p)
			if entry.IsDir() && (strings.Count(rel, "/") >= MaxDepth || strings.EqualFold(path.Ext(p), ".zip")) {
				return fs.SkipDir
			}
			if !entry.IsDir() && strings.EqualFold(path.Ext(p), ".zip") {
				if archive != "" || !entry.Type().IsRegular() {
					return nil
				}
				err := readPack(root, p, func(z *zip.Reader) error {
					if g, ok := cartridgeGame(z, ".", path.Base(p), fs.ModeDir); ok {
						add(p, g)
						return nil
					}
					return walk(z, p)
				})
				if errors.Is(err, limit) {
					return err
				}
				if err != nil {
					result.Problems = append(result.Problems, fmt.Sprintf("%s: %v", path.Join(folder, p), err))
				}
				return nil
			}
			if g, ok := cartridgeGame(fsys, p, entry.Name(), entry.Type()); ok {
				add(rel, g)
				if entry.IsDir() {
					return fs.SkipDir
				}
			}
			return nil
		})
	}
	return walk(root.FS(), "")
}
