package debugsrv

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// An existing malformed key disables the server; never silently replace access.
func loadToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		token := strings.TrimSpace(string(b))
		decoded, e := hex.DecodeString(token)
		if e != nil || len(decoded) != 32 {
			return "", fmt.Errorf("invalid debug token file")
		}
		return token, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	var key [32]byte
	if _, err = rand.Read(key[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(key[:])
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err = f.WriteString(token + "\n"); err != nil {
		return "", err
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	return token, nil
}
