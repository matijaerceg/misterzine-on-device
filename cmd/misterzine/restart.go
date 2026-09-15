//go:build linux

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

// Compare bytes, not release labels or timestamps: a same-version reinstall
// must not ask for a restart unless the installed program actually changed.
func differentProgram(running, installed string) (bool, error) {
	hash := func(path string) ([32]byte, error) {
		var sum [32]byte
		f, err := os.Open(path)
		if err != nil {
			return sum, err
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			return sum, err
		}
		if !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
			return sum, fmt.Errorf("program is not executable: %s", path)
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return sum, err
		}
		copy(sum[:], h.Sum(nil))
		return sum, nil
	}
	a, err := hash(running)
	if err != nil {
		return false, err
	}
	b, err := hash(installed)
	return err == nil && a != b, err
}

func (h *host) requestUpdateRestart(id string) {
	if h.updatePending || h.updateRunning || h.a.UpdateState().ID != id || !h.a.UpdateRestartAvailable() {
		return
	}
	state, err := updater.Read(h.root)
	if err != nil || state.Active() || updater.OtherScript() {
		h.a.Notice("Cannot restart while updater status is uncertain", 8*time.Second)
		return
	}
	h.restartRequested = true
}

func (h *host) restartApp() int {
	path := filepath.Join(h.root, "misterzine")
	// Keep the wrapper and launcher waiting on this process while replacing
	// its image. Restore device resources and preserve the original flags.
	h.cleanup()
	if err := syscall.Exec(path, append([]string{path}, os.Args[1:]...), os.Environ()); err != nil {
		h.lg.Printf("restart failed: %v", err)
		return 1
	}
	return 0
}
