package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type acknowledgement struct {
	ID string `json:"id"`
}

func (s State) RecoveryNotice() bool {
	return s.Status == "interrupted" || s.Status == "restarted"
}

// ShouldOpen always reconnects an active run. A recovery warning reopens
// until the user explicitly dismisses that run; unreadable acknowledgements
// cannot silently hide a warning.
func ShouldOpen(root string, s State) bool {
	if s.Active() {
		return true
	}
	if !s.RecoveryNotice() {
		return false
	}
	var ack acknowledgement
	b, err := os.ReadFile(filepath.Join(stateDir(root), "acknowledged.json"))
	return err != nil || json.Unmarshal(b, &ack) != nil || s.ID == "" || ack.ID != s.ID
}

// Acknowledge records only the run ID, leaving the worker's evidence intact.
// A new worker may start during this save: its different ID is unaffected.
func Acknowledge(root, id string) error {
	s, err := Read(root)
	if err != nil {
		return err
	}
	if id == "" || s.ID != id || !s.RecoveryNotice() {
		return fmt.Errorf("that recovery warning is no longer current")
	}
	return saveCard(filepath.Join(stateDir(root), "acknowledged.json"), acknowledgement{ID: id})
}
