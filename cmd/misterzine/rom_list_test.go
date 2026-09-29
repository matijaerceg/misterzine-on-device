//go:build linux

package main

import (
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/beta"
)

// Only the Patreon beta writes the list of ROM problems, into MisterZine's
// folder, and Filters names it from the card's root.
func TestROMListOnlyInTheBeta(t *testing.T) {
	defer beta.Set(false)()
	if got := romListPath("/media/fat/misterzine"); got != "" {
		t.Fatalf("the free build writes a list to %q", got)
	}
	beta.Set(true)
	path := romListPath("/media/fat/misterzine")
	if path != "/media/fat/misterzine/rom_problems.txt" {
		t.Fatalf("the beta writes its list to %q", path)
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
