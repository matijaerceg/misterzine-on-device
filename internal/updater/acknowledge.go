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

func (s State) ResultNotice() bool {
	switch s.Status {
	case "interrupted", "restarted", "completed", "failed", "errors", "cancelled":
		return true
	}
	return false
}

// ShouldOpen always reconnects an active run. A terminal result reopens
// until the user explicitly dismisses that run; unreadable acknowledgements
// cannot silently hide a warning.
func ShouldOpen(root string, s State) bool {
	if s.Active() {
		return true
	}
	if !s.ResultNotice() {
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
	if id == "" || s.ID != id || !s.ResultNotice() {
		return fmt.Errorf("that result is no longer current")
	}
	return saveCard(filepath.Join(stateDir(root), "acknowledged.json"), acknowledgement{ID: id})
}
