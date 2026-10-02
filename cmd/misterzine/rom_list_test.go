//go:build linux

package main

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/access"
)

func TestROMListRequiresAccessAndBetaOptIn(t *testing.T) {
	for _, month := range []access.Month{0, 202609, 202610, 202612} {
		for _, show := range []bool{false, true} {
			got := romListPath("/media/fat/misterzine", month, show)
			if (got != "") != (show && month >= 202610) {
				t.Fatalf("month=%d show=%v path=%q", month, show, got)
			}
		}
	}
	path := romListPath("/media/fat/misterzine", 202610, true)
	if path != "/media/fat/misterzine/rom_problems.txt" {
		t.Fatal(path)
	}
	for _, tc := range []struct{ path, want string }{
		{path, "misterzine/rom_problems.txt"},
		{"", ""},
		{"/tmp/elsewhere.txt", "/tmp/elsewhere.txt"},
	} {
		if got := cardRelative("/media/fat", tc.path); got != tc.want {
			t.Errorf("%q on the card reads %q, want %q", tc.path, got, tc.want)
		}
	}
}
