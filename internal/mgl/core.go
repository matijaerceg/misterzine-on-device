package mgl

import (
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
)

// CorePresent checks only the directory explicitly named by the shortcut.
// It cannot establish ROM, BIOS, or configuration compatibility.
func CorePresent(card string, d Document) bool {
	rbf := strings.TrimSuffix(d.RBF, path.Ext(d.RBF))
	if path.IsAbs(rbf) || strings.Contains(rbf, "\\") {
		return false
	}
	root, err := os.OpenRoot(card)
	if err != nil {
		return false
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), path.Dir(rbf))
	if err != nil {
		return false
	}
	base := strings.ToLower(path.Base(rbf))
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if e.Type().IsRegular() && (name == base+".rbf" || regexp.MustCompile(`^`+regexp.QuoteMeta(base)+`_[0-9]{8}[a-z]?(?:_[^/]*)?\.rbf$`).MatchString(name)) {
			return true
		}
	}
	return false
}
