package access

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPermanentAccess(t *testing.T) {
	old := grants
	grants = nil
	t.Cleanup(func() { grants = old })
	add := func(m Month, code, legacy string) Grant {
		sum := sha256.Sum256([]byte(code))
		g := Grant{m, hex.EncodeToString(sum[:]), legacy}
		Register(g)
		return g
	}
	g := add(202610, "123456", "arcade-1")
	add(202612, "654321", "")
	dir := t.TempDir()
	if Load(dir) != 0 {
		t.Fatal("fresh install unlocked")
	}
	if _, err := Unlock(dir, "999999"); !errors.Is(err, ErrCode) {
		t.Fatal(err)
	}
	if m, err := Unlock(dir, "123456"); m != 202610 || err != nil {
		t.Fatal(m, err)
	}
	if Load(dir) != 202610 {
		t.Fatal("old code lost on reload")
	}
	if m, err := Unlock(dir, "654321"); m != 202612 || err != nil {
		t.Fatal(m, err)
	}
	if m, err := Unlock(dir, "123456"); m != 202612 || err != nil {
		t.Fatal("older code reduced access", m, err)
	}
	if Load(dir) != 202612 {
		t.Fatal("highest month not retained")
	}
	legacy := t.TempDir()
	path := filepath.Join(legacy, "beta-unlocks", g.LegacyBatch+"-"+g.SHA256+".receipt")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte("unlocked\n"), 0644)
	if Load(legacy) != 202610 {
		t.Fatal("original beta receipt not honoured")
	}
	os.WriteFile(path, []byte("broken"), 0644)
	if Load(legacy) != 0 {
		t.Fatal("damaged receipt granted access")
	}
	if m, err := Unlock("", "123456"); m != 202610 || err == nil {
		t.Fatal("valid unsaved code", m, err)
	}
}

func TestFeatureLifecycle(t *testing.T) {
	for _, fancy := range []bool{false, true} {
		f := Feature{Fancy: fancy, Beta: true, Since: 202610}
		if f.Visible(false) || f.Allowed(202612, false) || f.Allowed(202609, true) {
			t.Fatal("beta gate bypassed", f)
		}
		if !f.Visible(true) || !f.Allowed(202610, true) {
			t.Fatal("qualifying beta hidden", f)
		}
		f.Beta = false
		if !f.Visible(false) || !f.Allowed(202610, false) {
			t.Fatal("graduation lost access", f)
		}
		if f.Allowed(0, false) == fancy {
			t.Fatal("graduation changed fancy requirement", f)
		}
	}
	if (Feature{Fancy: true}).Allowed(202610, true) {
		t.Fatal("missing threshold fails open")
	}
}

func TestForgetAllReceiptsKeepsSettingsAndCode(t *testing.T) {
	old := grants
	grants = nil
	t.Cleanup(func() { grants = old })
	sum := sha256.Sum256([]byte("123456"))
	g := Grant{Month: 202610, SHA256: hex.EncodeToString(sum[:]), LegacyBatch: "old-batch"}
	Register(g)
	dir := t.TempDir()
	if _, err := Unlock(dir, "123456"); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(dir, "beta-unlocks")
	os.MkdirAll(legacy, 0755)
	os.WriteFile(filepath.Join(legacy, g.LegacyBatch+"-"+g.SHA256+".receipt"), []byte("unlocked\n"), 0644)
	os.WriteFile(filepath.Join(legacy, "unknown.receipt"), []byte("unlocked\n"), 0644)
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("keep settings"), 0644)
	if m, err := Forget(dir); err != nil || m != 0 || Load(dir) != 0 {
		t.Fatal(m, err)
	}
	for _, name := range []string{"unlocks", "beta-unlocks"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatal("receipt directory retained", err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	if string(b) != "keep settings" {
		t.Fatal("settings changed")
	}
	if m, err := Unlock(dir, "123456"); err != nil || m != 202610 {
		t.Fatal("code revoked", m, err)
	}
	if _, err := Forget(""); err == nil {
		t.Fatal("empty root accepted")
	}
	if _, err := Forget(filepath.Join(dir, "settings.json")); err == nil {
		t.Fatal("invalid storage reported success")
	}
}
