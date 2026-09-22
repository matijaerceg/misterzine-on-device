package scan

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Layout limits keep the diagnostic report readable on a card of any size.
const (
	layoutDepth   = 3    // _Arcade folders listed this deep
	layoutFolders = 250  // folder lines at most
	layoutNames   = 1000 // arcade core names at most
	layoutTop     = 200  // top-level entries at most
)

// CardLayout describes the card's folders for the diagnostic report, where a
// misplaced folder often explains a missing game on its own:
//   - the card's top level, with the entry count of each folder
//   - the _Arcade tree three folders deep, each folder's MRA count, and the
//     folders the local walk leaves out marked with its reason
//   - every file in _Arcade/cores, and the core counts of the other core folders
//   - the ROM folder MiSTer's arcade loader picks, with its zip counts
//   - the app's own folder (appDir), with file sizes
//
// It reads names only, never a file's contents.
func CardLayout(card, appDir string) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }

	top, err := os.ReadDir(card)
	if err != nil {
		return []string{"Card: " + err.Error()}
	}
	var parts []string
	for i, e := range top {
		if i == layoutTop {
			parts = append(parts, fmt.Sprintf("... %d more", len(top)-layoutTop))
			break
		}
		if isDir(card, e) {
			parts = append(parts, fmt.Sprintf("%s/ (%d)", e.Name(), countEntries(filepath.Join(card, e.Name()))))
		} else {
			parts = append(parts, e.Name())
		}
	}
	add("Card top level: %s", strings.Join(parts, ", "))

	// Every folder line counts against layoutFolders; past it one line says
	// so and the rest of the tree is left out.
	folders := 0
	folder := func(depth int, format string, args ...any) bool {
		folders++
		if folders > layoutFolders {
			if folders == layoutFolders+1 {
				add("%s... more folders not listed", indent(depth))
			}
			return false
		}
		add(indent(depth)+format, args...)
		return true
	}
	var tree func(rel string, depth int)
	tree = func(rel string, depth int) {
		name := rel
		if depth > 0 {
			name = path.Base(rel)
		}
		entries, err := os.ReadDir(filepath.Join(card, filepath.FromSlash(rel)))
		if err != nil {
			folder(depth, "%s/: %v", name, err)
			return
		}
		mras, dirs := 0, 0
		for _, e := range entries {
			switch {
			case isDir(card, e, rel):
				dirs++
			case strings.HasSuffix(strings.ToLower(e.Name()), ".mra"):
				mras++
			}
		}
		if !folder(depth, "%s/: %d MRAs, %d folders", name, mras, dirs) {
			return
		}
		for _, e := range entries {
			if !isDir(card, e, rel) {
				continue
			}
			sub := path.Join(rel, e.Name())
			l := strings.ToLower(e.Name())
			ok := true
			switch {
			case e.Type()&os.ModeSymlink != 0:
				ok = folder(depth+1, "%s/: skipped, a symbolic link", e.Name())
			case skipDirReason(e.Name()) != "":
				ok = folder(depth+1, "%s/: skipped, %s", e.Name(), skipDirReason(e.Name()))
			case l == "cores" && depth == 0:
				ok = folder(depth+1, "%s/: %d files (listed below)", e.Name(), countEntries(filepath.Join(card, filepath.FromSlash(sub))))
			case l == "_alternatives":
				ok = folder(depth+1, "%s/: %d folders (read as versions of their games)", e.Name(), countEntries(filepath.Join(card, filepath.FromSlash(sub))))
			case depth+1 >= layoutDepth:
				ok = folder(depth+1, "%s/: %d MRAs, not listed deeper", e.Name(), countSuffix(filepath.Join(card, filepath.FromSlash(sub)), ".mra"))
			default:
				tree(sub, depth+1)
			}
			if !ok || folders > layoutFolders {
				return
			}
		}
	}
	tree("_Arcade", 0)

	cores, err := os.ReadDir(filepath.Join(card, "_Arcade", "cores"))
	if err != nil {
		add("Arcade cores: %v", err)
	} else {
		var names []string
		for _, e := range cores {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		n := len(names)
		if n > layoutNames {
			names = append(names[:layoutNames], fmt.Sprintf("... %d more", n-layoutNames))
		}
		add("Arcade cores (%d files in _Arcade/cores):", n)
		for len(names) > 0 {
			k := min(8, len(names))
			add("  %s", strings.Join(names[:k], ", "))
			names = names[k:]
		}
	}
	var others []string
	for _, dir := range coreDirs[1:] {
		others = append(others, fmt.Sprintf("%s %d", dir, countSuffix(filepath.Join(card, dir), ".rbf")))
	}
	add("Other core folders (.rbf files): %s", strings.Join(others, ", "))

	root, rootMame := arcadeROMRoot(card)
	add("Arcade ROMs: MiSTer reads %s (mame: %d zips, hbmame: %d zips)", filepath.ToSlash(root),
		countSuffix(filepath.Join(root, "mame"), ".zip"), countSuffix(filepath.Join(root, "hbmame"), ".zip"))
	if rootMame {
		add("  a mame folder at the top of the SD card: MiSTer picks it and then cannot read from it")
	}

	if appDir != "" {
		entries, err := os.ReadDir(appDir)
		if err != nil {
			add("App folder: %v", err)
		} else {
			var files []string
			for _, e := range entries {
				if e.IsDir() {
					files = append(files, e.Name()+"/")
					continue
				}
				size := int64(0)
				if info, err := e.Info(); err == nil {
					size = info.Size()
				}
				files = append(files, fmt.Sprintf("%s (%s)", e.Name(), byteSize(size)))
			}
			add("App folder %s: %s", filepath.ToSlash(appDir), strings.Join(files, ", "))
		}
	}
	return out
}

// isDir reports a folder, following a symlink to see whether it is one.
func isDir(card string, e os.DirEntry, rel ...string) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	p := filepath.Join(card, e.Name())
	if len(rel) > 0 {
		p = filepath.Join(card, filepath.FromSlash(rel[0]), e.Name())
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func countEntries(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	return len(entries)
}

func countSuffix(dir, suffix string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), suffix) {
			n++
		}
	}
	return n
}

func indent(depth int) string { return strings.Repeat("  ", depth) }

func byteSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
