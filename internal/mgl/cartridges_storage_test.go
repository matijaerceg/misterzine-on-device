package mgl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNeoGeoDirectoryPriority(t *testing.T) {
	card := filepath.Join(t.TempDir(), "fat")
	// Add each higher-priority directory without removing the previous ones.
	for _, want := range []string{
		"games/NEOGEO", "NEOGEO", "cifs/games/NEOGEO", "cifs/NEOGEO",
		"../network/games/NEOGEO", "../network/NEOGEO",
		"../usb5/games/NEOGEO", "../usb5/NEOGEO", "../usb4/games/NEOGEO",
		"../usb3/games/NEOGEO", "../usb2/games/NEOGEO", "../usb1/games/NEOGEO",
		"../usb0/games/NEOGEO", "../usb0/NEOGEO",
	} {
		if err := os.MkdirAll(filepath.Join(card, want), 0755); err != nil {
			t.Fatal(err)
		}
		if got := neoGeoFolder(card); got != want {
			t.Fatalf("selected %q, want %q", got, want)
		}
		if got, ok := GameBackingPath(want + "/set.zip/sub/mslug.neo"); !ok || got != want+"/set.zip" {
			t.Fatalf("backing path %q for library %q", got, want)
		}
	}
	for _, target := range []string{
		"../usb6/NEOGEO/a.neo", "../usb0/other/a.neo", "../usb0/games/NEOGEO/../a.neo",
		"../usb0/../usb1/NEOGEO/a.neo", "../../usb0/NEOGEO/a.neo", "/media/usb0/NEOGEO/a.neo",
	} {
		if _, ok := GameBackingPath(target); ok {
			t.Errorf("accepted invalid library target %q", target)
		}
	}
}

func TestNeoGeoExternalFolderSpelling(t *testing.T) {
	card := filepath.Join(t.TempDir(), "fat")
	const sd = "games/NEOGEO"
	const usb = "../usb0/games/NeoGeo"
	for _, dir := range []string{sd, usb} {
		if err := os.MkdirAll(filepath.Join(card, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// FAT resolves the stock core's NEOGEO name to the mixed-case USB
	// directory. On a case-sensitive disk Main skips it and uses the SD
	// directory. Keep the actual spelling for saved game keys.
	want := sd
	if _, err := os.Stat(filepath.Join(card, "../usb0/games/NEOGEO")); err == nil {
		want = usb
	}
	if got := neoGeoFolder(card); got != want {
		t.Fatalf("selected %q, want %q", got, want)
	}
}
