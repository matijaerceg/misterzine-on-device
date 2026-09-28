package buildinfo

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
)

func TestStringNamesTheBeta(t *testing.T) {
	oldV, oldC, oldD, oldB := Version, Commit, Date, beta.Batch
	t.Cleanup(func() { Version, Commit, Date, beta.Batch = oldV, oldC, oldD, oldB })
	Version, Commit, Date, beta.Batch = "v1.2.0", "abc1234", "2026-09-28", ""
	if got := String(); got != "v1.2.0 (abc1234, 2026-09-28)" {
		t.Fatalf("free build: %q", got)
	}
	defer beta.Set(true)()
	if got := String(); got != "v1.2.0 (abc1234, 2026-09-28), beta" {
		t.Fatalf("beta without a batch: %q", got)
	}
	beta.Batch = "arcade-1"
	if got := String(); got != "v1.2.0 (abc1234, 2026-09-28), beta batch arcade-1" {
		t.Fatalf("beta: %q", got)
	}
	Commit, Date = "", ""
	if got := String(); got != "v1.2.0, beta batch arcade-1" {
		t.Fatalf("beta without build stamps: %q", got)
	}
}
