package platform

import (
	"os"
	"path"
	"path/filepath"
)

// GameDirParents yields card-relative paths in MiSTer Main's findGamesDir order.
// Each device's legacy top-level layout precedes its games folder.
func GameDirParents(yield func(string) bool) {
	for _, parent := range [...]string{
		"../usb0", "../usb0/games", "../usb1", "../usb1/games",
		"../usb2", "../usb2/games", "../usb3", "../usb3/games",
		"../usb4", "../usb4/games", "../usb5", "../usb5/games",
		"../network", "../network/games", "cifs", "cifs/games", "", "games",
	} {
		if !yield(parent) {
			return
		}
	}
}

// FindGameDir returns the first existing directory, even when it is empty.
func FindGameDir(card, name string) (string, bool) {
	for parent := range GameDirParents {
		dir := path.Join(parent, name)
		if st, err := os.Stat(filepath.Join(card, filepath.FromSlash(dir))); err == nil && st.IsDir() {
			return dir, true
		}
	}
	return "", false
}
