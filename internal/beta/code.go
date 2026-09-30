package beta

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

// The beta's lock. A members' release is built with
//
//	-X .../internal/beta.Batch=arcade-1 -X .../internal/beta.CodeSHA256=<64 hex digits>
//
// and opens on the lock screen until the batch's code is entered. Unlocking
// saves a receipt for the batch beside settings.json
// (beta-unlocks/<batch>-<sha>.receipt), so each batch asks once: an older
// batch's receipt keeps that batch opening after a rollback, and a new
// batch needs its new code. This is a convenience gate for members, not
// protection: a six-digit code can be found from its hash, and a modified
// build skips the check.

var batchName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)

var (
	// ErrLocked is a beta build still waiting for its code, or a wrong code.
	ErrLocked = errors.New("beta: the code from the Patreon post is needed")
	// ErrBuild is a build whose beta values do not belong together: it
	// cannot be unlocked and has to be rebuilt or reinstalled.
	ErrBuild = errors.New("beta: this build's beta settings are invalid")
)

const receiptBody = "unlocked\n"

// Validate reports ErrBuild when the values set at build time do not
// belong together: a channel other than "" or "beta", a batch or code on
// the free build, a code without a valid batch, or a code that is not
// 64 lowercase hex digits. The release workflow runs it through
// misterzine -version.
func Validate() error {
	if Channel != "" && Channel != "beta" {
		return ErrBuild
	}
	if Channel != "beta" && (Batch != "" || CodeSHA256 != "") {
		return ErrBuild
	}
	if Batch != "" && !batchName.MatchString(Batch) {
		return ErrBuild
	}
	if CodeSHA256 == "" {
		return nil
	}
	raw, err := hex.DecodeString(CodeSHA256)
	if Batch == "" || err != nil || len(raw) != sha256.Size || hex.EncodeToString(raw) != CodeSHA256 {
		return ErrBuild
	}
	return nil
}

// Locks reports whether this build asks for a code at all: a beta build
// with a code. The free build and a beta build without one never lock.
func Locks() bool { return On() && CodeSHA256 != "" }

// Check reports whether the app may open, reading the saved receipt again
// each time: nil for the free build, a beta build without a code, and a
// batch unlocked on this card; ErrLocked while the batch's code has not
// been entered; ErrBuild for a beta build with invalid values. dir is the
// folder settings.json lives in.
func Check(dir string) error {
	if !On() {
		return nil
	}
	if err := Validate(); err != nil {
		return err
	}
	if CodeSHA256 == "" {
		return nil
	}
	if dir == "" {
		return ErrLocked
	}
	b, err := os.ReadFile(receiptPath(dir))
	if err != nil || string(b) != receiptBody {
		return ErrLocked
	}
	return nil
}

// ValidCode reports whether a code has the shape of one: exactly six
// ASCII digits.
func ValidCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}

// Unlock checks a code against the build's and saves the batch's receipt
// in dir before it returns. ErrLocked is a wrong code, ErrBuild a build
// that cannot be unlocked; any other error means the code was right but
// the receipt could not be saved.
func Unlock(dir, code string) error {
	if err := Validate(); err != nil {
		return err
	}
	if !Locks() {
		return nil // nothing to unlock
	}
	if !ValidCode(code) {
		return ErrLocked
	}
	sum := sha256.Sum256([]byte(code))
	if hex.EncodeToString(sum[:]) != CodeSHA256 {
		return ErrLocked
	}
	if dir == "" {
		return errors.New("beta: no settings folder to save the unlock in")
	}
	if Check(dir) == nil {
		return nil
	}
	path := receiptPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".unlock-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(receiptBody); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// receiptPath is the batch's receipt. The code's hash is part of the name,
// so a batch rebuilt with another code asks again.
func receiptPath(dir string) string {
	return filepath.Join(dir, "beta-unlocks", Batch+"-"+CodeSHA256+".receipt")
}

// Unlocked reports whether the card in dir has ever been unlocked: any
// batch's receipt in beta-unlocks, this build's or another's. The free
// build asks it too, to know a card that had MisterZine Arcade.
func Unlocked(dir string) bool {
	return len(receipts(dir)) > 0
}

// EarlierUnlock reports a receipt for a batch other than this build's: a
// card that was unlocked before and now waits for a newer batch's code,
// which is where a lapsed member stands.
func EarlierUnlock(dir string) bool {
	own := filepath.Base(receiptPath(dir))
	for _, name := range receipts(dir) {
		if name != own {
			return true
		}
	}
	return false
}

// receipts lists the receipt files in dir's beta-unlocks, by name.
func receipts(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(dir, "beta-unlocks"))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() && filepath.Ext(e.Name()) == ".receipt" {
			names = append(names, e.Name())
		}
	}
	return names
}

// Describe names the build for version lines: "" for the free build,
// "beta" or "beta batch arcade-1" for the beta.
func Describe() string {
	if !On() {
		return ""
	}
	if Batch == "" {
		return "beta"
	}
	return "beta batch " + Batch
}
