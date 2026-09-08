package store

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSettings(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A file from before the two-axis inset carries only "inset": both axes
// inherit it, even though the defaults (15/15) were loaded first.
func TestLoadSettingsLegacyInset(t *testing.T) {
	for _, in := range []int{0, 4, 22} {
		s, err := LoadSettings(writeSettings(t, `{"schema":1,"rotation":"auto","inset":`+itoa(in)+`,"scroll":"fast"}`))
		if err != nil {
			t.Fatal(err)
		}
		if s.InsetX != in || s.InsetY != in {
			t.Errorf("inset %d: got %d/%d", in, s.InsetX, s.InsetY)
		}
		if s.Scroll != "30" {
			t.Errorf("scroll: got %q", s.Scroll)
		}
	}
}

// A current file keeps its own axes, zero included.
func TestLoadSettingsTwoAxes(t *testing.T) {
	s, err := LoadSettings(writeSettings(t, `{"schema":1,"inset":9,"inset_x":0,"inset_y":3,"scroll":"60"}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.InsetX != 0 || s.InsetY != 3 || s.Scroll != "60" {
		t.Errorf("got %+v", s)
	}
}

func TestLoadSettingsMissingIsDefaults(t *testing.T) {
	s, err := LoadSettings(filepath.Join(t.TempDir(), "none.json"))
	if !os.IsNotExist(err) {
		t.Fatalf("err = %v", err)
	}
	if s != DefaultSettings() {
		t.Errorf("got %+v", s)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
