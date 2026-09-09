package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecoveryAcknowledgementGuards(t *testing.T) {
	root := t.TempDir()
	s := State{ID: "run", Status: "interrupted"}
	if err := saveCard(StatePath(root), s); err != nil {
		t.Fatal(err)
	}
	if err := Acknowledge(root, "different-run"); err == nil || !ShouldOpen(root, s) {
		t.Fatal("a stale dismissal hid an unseen run")
	}
	if err := Acknowledge(root, ""); err == nil {
		t.Fatal("an unidentified run was acknowledged")
	}
	if err := Acknowledge(root, s.ID); err != nil || ShouldOpen(root, s) {
		t.Fatalf("dismissal did not persist: %v", err)
	}
	for _, status := range []string{"starting", "running", "cancelling"} {
		if !ShouldOpen(root, State{ID: s.ID, Status: status}) {
			t.Fatalf("acknowledgement suppressed active state %s", status)
		}
	}
	ackPath := filepath.Join(stateDir(root), "acknowledged.json")
	if err := os.WriteFile(ackPath, []byte("broken JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if !ShouldOpen(root, s) {
		t.Fatal("an unreadable acknowledgement hid a warning")
	}
}

func TestFailedRecoveryAcknowledgementRemainsPending(t *testing.T) {
	root := t.TempDir()
	s := State{ID: "run", Status: "restarted", SawSuccess: true}
	if err := saveCard(StatePath(root), s); err != nil {
		t.Fatal(err)
	}
	// Reliable write failure even for root on the devices.
	if err := os.Mkdir(filepath.Join(stateDir(root), "acknowledged.json.tmp"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Acknowledge(root, s.ID); err == nil {
		t.Fatal("failed acknowledgement was reported as saved")
	}
	if !ShouldOpen(root, s) {
		t.Fatal("a failed save suppressed the next recovery warning")
	}
}

func TestTerminalResultsReopenOnce(t *testing.T) {
	for _, status := range []string{"completed", "errors", "failed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			root := t.TempDir()
			s := State{ID: "result", Status: status}
			if err := saveCard(StatePath(root), s); err != nil {
				t.Fatal(err)
			}
			if !ShouldOpen(root, s) {
				t.Fatal("unseen result hidden")
			}
			if err := Acknowledge(root, s.ID); err != nil {
				t.Fatal(err)
			}
			if ShouldOpen(root, s) {
				t.Fatal("dismissed result reopened")
			}
		})
	}
}
