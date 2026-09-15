package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestROMRequirements(t *testing.T) {
	for _, tc := range []struct {
		name, mra string
		files     []string
		want      string
	}{
		{"jurassic", `<rom index="4" zip="jpark.zip"><part name="program"/></rom><rom index="9" zip="jpark.zip"><part name="sprites"/></rom><rom index="0"><part>00</part></rom>`, nil, "jpark.zip"},
		{"present", `<rom index="4" zip="jpark.zip"><part name="program"/></rom>`, []string{"jpark.zip"}, ""},
		{"parent fallback", `<rom index="0" zip="clone.zip|parent.zip"><part name="program"/></rom>`, []string{"parent.zip"}, ""},
		{"part override", `<rom index="0" zip="parent.zip"><part name="program" zip="extra.zip"/></rom>`, []string{"parent.zip"}, "extra.zip"},
		{"inline", `<rom index="0"><part>1234</part></rom>`, nil, ""},
		{"alternate layouts", `<rom index="0" zip="absent.zip"><part name="a"/></rom><rom index="0" zip="present.zip"><part name="b"/></rom>`, []string{"present.zip"}, ""},
		{"later requirement", `<rom index="0"><part>00</part></rom><rom index="5" zip="sound.zip"><part name="sound"/></rom>`, nil, "sound.zip"},
		{"comments", `<!-- invalid XML -- comment --><rom index="4" zip="jpark.zip"><part name="program"/></rom>`, nil, "jpark.zip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := filepath.Join(t.TempDir(), "fat")
			put := func(p, content string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			put(filepath.Join(card, "_Arcade/game.mra"), "<misterromdescription>"+tc.mra+"</misterromdescription>")
			for _, f := range tc.files {
				put(filepath.Join(card, "games/mame", f), "archive presence only")
			}
			got := checkROMs(card, "_Arcade/game.mra")
			if tc.want == "" && got != "" || tc.want != "" && !strings.Contains(got, tc.want) {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestROMStorageAndFreshCheck(t *testing.T) {
	media := t.TempDir()
	card := filepath.Join(media, "fat")
	for _, p := range []string{filepath.Join(card, "games/mame"), filepath.Join(media, "usb0/games/mame")} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(card, "game.mra"), []byte(`<misterromdescription><rom index="4" zip="jpark.zip"><part name="a"/></rom></misterromdescription>`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(card, "games/mame/jpark.zip"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	check := NewROMCheck(card)
	if got := check("game.mra", false); !strings.Contains(got, "jpark.zip") {
		t.Fatalf("must use USB directory, got %q", got)
	}
	if err := os.WriteFile(filepath.Join(media, "usb0/games/mame/jpark.zip"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got := check("game.mra", true); got != "" {
		t.Fatalf("fresh Start check must see new archive: %s", got)
	}
}
