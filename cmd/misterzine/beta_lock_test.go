//go:build linux

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/data"
	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// lockedBuild makes this a beta build locked behind 123456 for one test.
func lockedBuild(t *testing.T) {
	t.Helper()
	channel, batch, code := beta.Channel, beta.Batch, beta.CodeSHA256
	t.Cleanup(func() { beta.Channel, beta.Batch, beta.CodeSHA256 = channel, batch, code })
	sum := sha256.Sum256([]byte("123456"))
	beta.Channel, beta.Batch, beta.CodeSHA256 = "beta", "test", hex.EncodeToString(sum[:])
}

func TestFreeBuildHasNoLock(t *testing.T) {
	defer beta.Set(false)()
	if betaLock(t.TempDir(), log.New(io.Discard, "", 0)) != nil {
		t.Fatal("the free build locks")
	}
	beta.Set(true)
	if betaLock(t.TempDir(), log.New(io.Discard, "", 0)) != nil {
		t.Fatal("a beta build without a code locks")
	}
}

// A locked beta writes nothing but the unlock: quitting from the lock
// screen is not a visit, and the settings and favourites stay as they were.
// The code unlocks into the list, saves the receipt beside settings.json,
// and the next start opens without asking.
func TestLockedBetaSavesNothingUntilUnlocked(t *testing.T) {
	lockedBuild(t)
	root := t.TempDir()
	h := favoritesHost(root)
	unlock := betaLock(root, h.lg)
	if unlock == nil {
		t.Fatal("no lock without a receipt")
	}
	h.a = app.New(app.Config{PhysW: 320, PhysH: 240, BetaUnlock: unlock},
		data.Ingest([]data.Row{{K: "game", Base: "Arcade", Title: "Game"}}, "test", time.Now()), nil)
	h.dirty, h.setDirty, h.favDirty = true, true, true
	h.saveAll(true)
	h.autosave(time.Now().Add(time.Second), true)
	for _, name := range []string{"state.json", "settings.json", "favorites.json"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("the locked app wrote %s", name)
		}
	}
	at := time.Now()
	for _, ch := range "123456" {
		at = at.Add(50 * time.Millisecond)
		h.a.Handle(platform.Event{Key: platform.KeyOther, Text: ch, Pressed: true, At: at})
	}
	h.a.Handle(platform.Event{Key: platform.KeyEnter, Pressed: true, At: at.Add(50 * time.Millisecond)})
	if h.a.Locked() {
		t.Fatal("the right code did not unlock")
	}
	receipts, _ := filepath.Glob(filepath.Join(root, "beta-unlocks", "test-*.receipt"))
	if len(receipts) != 1 {
		t.Fatalf("receipts %q", receipts)
	}
	h.saveAll(true)
	for _, name := range []string{"state.json", "settings.json", "favorites.json"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("the unlocked app did not save %s: %v", name, err)
		}
	}
	if betaLock(root, h.lg) != nil {
		t.Fatal("the next start asks for the code again")
	}
}
