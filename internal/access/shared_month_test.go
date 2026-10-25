package access

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// Two codes may cover one month: the arcade code and the Plex/Jellyfin code
// are interchangeable, and each unlocks that month on its own. One code
// registered twice is still refused.
func TestCodesSharingAMonth(t *testing.T) {
	old := grants
	grants = nil
	t.Cleanup(func() { grants = old })
	add := func(m Month, code, legacy string) {
		sum := sha256.Sum256([]byte(code))
		Register(Grant{m, hex.EncodeToString(sum[:]), legacy})
	}
	add(202610, "111111", "arcade-1")
	add(202610, "222222", "")
	for _, code := range []string{"111111", "222222"} {
		dir := t.TempDir()
		if m, err := Unlock(dir, code); err != nil || m != 202610 {
			t.Fatal(code, m, err)
		}
		if Load(dir) != 202610 {
			t.Fatal(code, "receipt not read back")
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate code accepted")
		}
	}()
	add(202611, "111111", "")
}
