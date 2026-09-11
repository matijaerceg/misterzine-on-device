package data

import "testing"

func TestReleaseYear(t *testing.T) {
	for raw, want := range map[string]string{"1980": "1980", " 1999 ": "1999", "": "", "198?": "", "1980-1982": "", "1980?": "", "0": "", "0000": ""} {
		if got := ReleaseYear(raw); got != want {
			t.Errorf("ReleaseYear(%q)=%q want %q", raw, got, want)
		}
	}
	for y, want := range map[string]string{"1979": "1970s", "1980": "1980s", "1989": "1980s", "1990": "1990s", "": ""} {
		if got := Decade(y); got != want {
			t.Errorf("Decade(%q)=%q want %q", y, got, want)
		}
	}
}
