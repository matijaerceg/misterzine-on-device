package beta

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func hashOf(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// codeBuild makes this a locked beta build for one test: batch "fixture",
// code 012345.
func codeBuild(t *testing.T) string {
	t.Helper()
	oldChannel, oldBatch, oldCode := Channel, Batch, CodeSHA256
	t.Cleanup(func() { Channel, Batch, CodeSHA256 = oldChannel, oldBatch, oldCode })
	Channel, Batch, CodeSHA256 = "beta", "fixture", hashOf("012345")
	return t.TempDir()
}

func TestCodeLifecycle(t *testing.T) {
	dir := codeBuild(t)
	if !Locks() {
		t.Fatal("a beta build with a code does not lock")
	}
	if Check(dir) != ErrLocked {
		t.Fatal("no receipt, yet unlocked")
	}
	for _, code := range []string{"", "12345", "0123450", "abcdef", "012346", " 012345", "012345\n", "01234５"} {
		if Unlock(dir, code) != ErrLocked {
			t.Fatalf("accepted %q", code)
		}
	}
	if Check(dir) != ErrLocked {
		t.Fatal("a wrong code unlocked")
	}
	if err := Unlock(dir, "012345"); err != nil {
		t.Fatal(err)
	}
	if err := Check(dir); err != nil {
		t.Fatal("the receipt did not persist:", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "beta-unlocks", "fixture-"+hashOf("012345")+".receipt"))
	if err != nil || string(raw) != "unlocked\n" {
		t.Fatalf("receipt %q, %v", raw, err)
	}
	if err := Unlock(dir, "012345"); err != nil {
		t.Fatal("unlocking again:", err)
	}
	if err := Unlock(dir, "999999"); err != ErrLocked {
		t.Fatal("a wrong code on an unlocked card was accepted:", err)
	}

	// a new batch needs its new code; its receipt leaves the old one alone
	Batch, CodeSHA256 = "next", hashOf("654321")
	if Check(dir) != ErrLocked {
		t.Fatal("the old batch's receipt opened the new batch")
	}
	if Unlock(dir, "012345") != ErrLocked {
		t.Fatal("the old code opened the new batch")
	}
	if err := Unlock(dir, "654321"); err != nil {
		t.Fatal(err)
	}
	// going back to the older batch keeps working
	Batch, CodeSHA256 = "fixture", hashOf("012345")
	if err := Check(dir); err != nil {
		t.Fatal("the rollback lost its unlock:", err)
	}
	// the same batch rebuilt with another code asks again
	CodeSHA256 = hashOf("111111")
	if Check(dir) != ErrLocked {
		t.Fatal("a receipt for another code opened this one")
	}
	CodeSHA256 = hashOf("012345")

	// a damaged receipt locks, and the code repairs it
	if err := os.WriteFile(receiptPath(dir), []byte("damaged"), 0644); err != nil {
		t.Fatal(err)
	}
	if Check(dir) != ErrLocked {
		t.Fatal("a damaged receipt unlocked")
	}
	if err := Unlock(dir, "012345"); err != nil {
		t.Fatal("could not repair the receipt:", err)
	}
	if err := Check(dir); err != nil {
		t.Fatal(err)
	}
}

func TestCodeCannotSave(t *testing.T) {
	dir := codeBuild(t)
	// a file where the receipts folder should be
	if err := os.WriteFile(filepath.Join(dir, "beta-unlocks"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := Unlock(dir, "012345"); err == nil || err == ErrLocked || err == ErrBuild {
		t.Fatal("a failed save was not reported as one:", err)
	}
	if Check(dir) != ErrLocked {
		t.Fatal("a failed save unlocked")
	}
	if err := Unlock("", "012345"); err == nil || err == ErrLocked {
		t.Fatal("no folder, yet saved:", err)
	}
	if Check("") != ErrLocked {
		t.Fatal("no folder, yet unlocked")
	}
}

func TestUnlockedWithoutCode(t *testing.T) {
	dir := codeBuild(t)
	for _, tc := range []struct{ channel, batch string }{{"", ""}, {"beta", ""}, {"beta", "fixture"}} {
		Channel, Batch, CodeSHA256 = tc.channel, tc.batch, ""
		if Locks() || Check(dir) != nil || Unlock(dir, "123456") != nil {
			t.Fatalf("%+v locks", tc)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "beta-unlocks")); !os.IsNotExist(err) {
		t.Fatal("an unlocked build wrote a receipt")
	}
}

func TestCodeBuildMetadata(t *testing.T) {
	dir := codeBuild(t)
	hash := CodeSHA256
	for _, tc := range []struct{ channel, batch, code string }{
		{"beta", "", hash}, {"beta", "../fixture", hash}, {"beta", "Fixture", hash}, {"beta", "-fixture", hash},
		{"beta", "fixture", "invalid"}, {"beta", "fixture", hash[:62]}, {"beta", "fixture", hash + "00"},
		{"beta", "fixture", "ABCDEF" + hash[6:]}, {"beta", "bad batch", ""},
		{"", "fixture", ""}, {"", "", hash}, {"", "fixture", hash}, {"public", "", ""}, {"Beta", "fixture", hash},
	} {
		Channel, Batch, CodeSHA256 = tc.channel, tc.batch, tc.code
		if Validate() != ErrBuild {
			t.Fatalf("invalid values accepted: %+v", tc)
		}
		if Unlock(dir, "012345") != ErrBuild {
			t.Fatalf("invalid values unlock: %+v", tc)
		}
		if On() && Check(dir) != ErrBuild {
			t.Fatalf("a beta build with invalid values opened: %+v", tc)
		}
		if !On() && Check(dir) != nil {
			t.Fatalf("the free build locked: %+v", tc)
		}
	}
	for _, tc := range []struct{ channel, batch, code string }{
		{"", "", ""}, {"beta", "", ""}, {"beta", "arcade-1", ""}, {"beta", "arcade-1", hash}, {"beta", "0", hash},
	} {
		Channel, Batch, CodeSHA256 = tc.channel, tc.batch, tc.code
		if err := Validate(); err != nil {
			t.Fatalf("valid values refused: %+v", tc)
		}
	}
}

func TestDescribe(t *testing.T) {
	codeBuild(t)
	for _, tc := range []struct{ channel, batch, want string }{
		{"", "", ""}, {"beta", "", "beta"}, {"beta", "arcade-1", "beta batch arcade-1"},
	} {
		Channel, Batch = tc.channel, tc.batch
		if got := Describe(); got != tc.want {
			t.Fatalf("%+v: %q", tc, got)
		}
	}
}
