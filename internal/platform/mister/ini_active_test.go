package mister

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAltcfgMarker(t *testing.T) {
	for _, tc := range []struct {
		b    []byte
		want int
	}{
		{[]byte{0x34, 0x99, 0xBA, 2}, 2},
		{[]byte{0x34, 0x99, 0xBA, 0}, 0},
		{[]byte{0, 0, 0, 2}, 0},
		{[]byte{0x34, 0x99}, 0},
		{nil, 0},
	} {
		if got := altcfgIndex(tc.b); got != tc.want {
			t.Fatalf("%v: got %d want %d", tc.b, got, tc.want)
		}
	}
}

func TestIniNameFollowsMainOrdering(t *testing.T) {
	card := t.TempDir()
	for _, n := range []string{"MiSTer.ini", "MiSTer_alt_1.ini", "mister_crt.INI", "MiSTer_Arcade.ini", "notes.ini", "MiSTer_.txt"} {
		if err := os.WriteFile(filepath.Join(card, n), []byte("osd_rotate=0\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Case-insensitive sort: alt_1, Arcade, crt.
	for alt, want := range map[int]string{0: "MiSTer.ini", 1: "MiSTer_alt_1.ini", 2: "MiSTer_Arcade.ini", 3: "mister_crt.INI", 4: "MiSTer.ini", -1: "MiSTer.ini"} {
		if got := IniName(card, alt); got != want {
			t.Fatalf("alt %d: got %q want %q", alt, got, want)
		}
	}
	// A selected slot with no file falls back to the primary INI.
	os.Remove(filepath.Join(card, "mister_crt.INI"))
	if got := IniName(card, 3); got != "MiSTer.ini" {
		t.Fatal(got)
	}
	if got := IniName(filepath.Join(card, "absent"), 1); got != "MiSTer.ini" {
		t.Fatal(got)
	}
}
