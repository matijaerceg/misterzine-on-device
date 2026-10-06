// Package access implements permanent, month-based MisterZine feature access.
// Months identify entitlements, never expiry dates; no clock or network is used.
package access

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Month int

func (m Month) Valid() bool { return m >= 202601 && m <= 999912 && m%100 >= 1 && m%100 <= 12 }
func (m Month) String() string {
	if !m.Valid() {
		return "None"
	}
	return fmt.Sprintf("%s %d", time.Month(m%100), m/100)
}

// Short names identify coverage, never an expiry date.
func (m Month) Short() string {
	if !m.Valid() {
		return "No code entered"
	}
	return fmt.Sprintf("%.3s %d", time.Month(m%100).String(), m/100)
}

// Feature separates readiness from access. Graduation changes Beta only.
type Feature struct {
	Fancy, Beta bool
	Since       Month
}

func (f Feature) Covered(month Month) bool {
	return (!f.Fancy && !f.Beta) || (f.Since.Valid() && month.Valid() && month >= f.Since)
}
func (f Feature) Visible(showBeta bool) bool { return !f.Beta || showBeta }
func (f Feature) Allowed(month Month, showBeta bool) bool {
	return f.Visible(showBeta) && f.Covered(month)
}

var (
	Controls  = Feature{Beta: true, Since: 202610}
	Themes    = Feature{Fancy: true, Beta: true, Since: 202610}
	Tallies   = Feature{Fancy: true, Beta: true, Since: 202610}
	ROMReport = Feature{Beta: true, Since: 202610}
	ErrCode   = errors.New("access: unrecognised MisterZine code")
)

// Grant is an issued code's public verifier, shared by all official apps.
// Never remove an issued grant. LegacyBatch permits the first beta's receipts.
type Grant struct {
	Month       Month
	SHA256      string
	LegacyBatch string
}

var grants []Grant

// Register is called by the private registry at startup. Invalid/ambiguous
// registries fail the build's startup instead of silently misgranting access.
func Register(g Grant) {
	b, err := hex.DecodeString(g.SHA256)
	if !g.Month.Valid() || err != nil || len(b) != sha256.Size || hex.EncodeToString(b) != g.SHA256 {
		panic("invalid access grant")
	}
	if g.LegacyBatch != "" && filepath.Base(g.LegacyBatch) != g.LegacyBatch {
		panic("invalid legacy batch")
	}
	for _, old := range grants {
		if old.SHA256 == g.SHA256 || old.Month == g.Month {
			panic("duplicate access grant")
		}
	}
	grants = append(grants, g)
}

func receipt(dir string, g Grant) string { return filepath.Join(dir, "unlocks", g.SHA256+".receipt") }
func has(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && string(b) == "unlocked\n"
}

// Load checks known receipts, including the original beta receipt, and returns
// the highest entitlement. Unknown or damaged files never grant access.
func Load(dir string) Month {
	if dir == "" {
		return 0
	}
	var best Month
	for _, g := range grants {
		ok := has(receipt(dir, g))
		if g.LegacyBatch != "" {
			ok = ok || has(filepath.Join(dir, "beta-unlocks", g.LegacyBatch+"-"+g.SHA256+".receipt"))
		}
		if ok && g.Month > best {
			best = g.Month
		}
	}
	return best
}

// Unlock returns the highest valid entitlement even if saving fails. An
// unrecognised code returns zero and ErrCode. Older receipts are never removed.
func Unlock(dir, code string) (Month, error) {
	if len(code) != 6 {
		return 0, ErrCode
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return 0, ErrCode
		}
	}
	sum := sha256.Sum256([]byte(code))
	for _, g := range grants {
		if hex.EncodeToString(sum[:]) != g.SHA256 {
			continue
		}
		month := max(g.Month, Load(dir))
		if dir == "" {
			return month, errors.New("access: no settings directory")
		}
		path := receipt(dir, g)
		if has(path) {
			return month, nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return month, err
		}
		f, err := os.CreateTemp(filepath.Dir(path), ".unlock-*")
		if err != nil {
			return month, err
		}
		defer os.Remove(f.Name())
		if _, err = f.WriteString("unlocked\n"); err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(f.Name(), path)
		}
		return month, err
	}
	return 0, ErrCode
}
