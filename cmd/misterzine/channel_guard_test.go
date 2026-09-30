//go:build linux

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
)

// guardCard is a card whose misterzine entry follows url ("" for no entry),
// with Downloader in Scripts; root is its settings folder.
func guardCard(t *testing.T, url string) (root, card string) {
	t.Helper()
	card = t.TempDir()
	root = filepath.Join(card, "misterzine")
	os.MkdirAll(filepath.Join(card, "Scripts"), 0755)
	os.MkdirAll(root, 0755)
	ini := "[mister]\nverbose = false\n"
	if url != "" {
		ini += "[MisterZine]\ndb_url = " + url + "\n"
	}
	if err := os.WriteFile(filepath.Join(card, "downloader.ini"), []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(card, "Scripts", "downloader.sh"), []byte("#!/bin/bash\n"), 0755)
	return root, card
}

// lockedBeta makes this a locked beta build for one test and returns the
// batch's code.
func lockedBeta(t *testing.T) string {
	t.Helper()
	old := [3]string{beta.Channel, beta.Batch, beta.CodeSHA256}
	t.Cleanup(func() { beta.Channel, beta.Batch, beta.CodeSHA256 = old[0], old[1], old[2] })
	sum := sha256.Sum256([]byte("123456"))
	beta.Channel, beta.Batch, beta.CodeSHA256 = "beta", "guard", hex.EncodeToString(sum[:])
	return "123456"
}

// The free build asks once, and only on a card that had the beta, follows
// the free releases now and has the member's installer.
func TestArcadeBackOffered(t *testing.T) {
	root, card := guardCard(t, buildinfo.FreeDBURL)
	if arcadeBackOffered(root, card, false) {
		t.Fatal("offered on a card that never had the beta")
	}
	code := lockedBeta(t)
	if err := beta.Unlock(root, code); err != nil {
		t.Fatal(err)
	}
	beta.Set(false)
	if arcadeBackOffered(root, card, false) {
		t.Fatal("offered without the member's installer")
	}
	os.WriteFile(filepath.Join(card, "Scripts", "MisterZine-Install-Beta.sh"), []byte("#!/bin/bash\n"), 0644)
	if !arcadeBackOffered(root, card, false) {
		t.Fatal("not offered on a card that had the beta and slid back to free")
	}
	if arcadeBackOffered(root, card, true) {
		t.Fatal("asked again after an answer")
	}
	// an entry that follows the beta (the swap has not reached Update All
	// yet) or anything else is not a slide back to free
	for _, url := range []string{buildinfo.BetaDBURL, "https://example.org/fork.json.zip", ""} {
		root2, card2 := guardCard(t, url)
		os.MkdirAll(filepath.Join(root2, "beta-unlocks"), 0755)
		os.WriteFile(filepath.Join(root2, "beta-unlocks", "guard-x.receipt"), []byte("unlocked\n"), 0644)
		os.WriteFile(filepath.Join(card2, "Scripts", "MisterZine-Install-Beta.sh"), []byte("#!/bin/bash\n"), 0644)
		if arcadeBackOffered(root2, card2, false) {
			t.Fatalf("offered with the entry at %q", url)
		}
	}
	// the beta itself never asks
	lockedBeta(t)
	if arcadeBackOffered(root, card, false) {
		t.Fatal("the beta offers the way back to the beta")
	}
}

// An unlocked beta points its entry back when it follows the free
// releases or is gone; a locked one, or one already on the beta or on
// someone's own database, is left alone.
func TestBetaEntryNeeded(t *testing.T) {
	code := lockedBeta(t)
	root, card := guardCard(t, buildinfo.FreeDBURL)
	if betaEntryNeeded(root, card) {
		t.Fatal("a locked beta rewrites the entry")
	}
	if err := beta.Unlock(root, code); err != nil {
		t.Fatal(err)
	}
	if !betaEntryNeeded(root, card) {
		t.Fatal("an unlocked beta following the free releases is left alone")
	}
	for url, want := range map[string]bool{"": true, buildinfo.BetaDBURL: false, "https://example.org/fork.json.zip": false} {
		root2, card2 := guardCard(t, url)
		if err := beta.Unlock(root2, code); err != nil {
			t.Fatal(err)
		}
		if got := betaEntryNeeded(root2, card2); got != want {
			t.Fatalf("entry %q: needed %v", url, got)
		}
	}
	// a card without downloader.ini is not a MiSTer the guard knows about
	os.Remove(filepath.Join(card, "downloader.ini"))
	if betaEntryNeeded(root, card) {
		t.Fatal("rewrites a card with no downloader.ini")
	}
	beta.Set(false)
	if _, card := guardCard(t, buildinfo.FreeDBURL); betaEntryNeeded(root, card) {
		t.Fatal("the free build rewrites the entry")
	}
}
