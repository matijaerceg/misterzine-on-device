package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/data"
)

func writeSettings(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRememberSortMigration(t *testing.T) {
	for _, tc := range []struct {
		body     string
		remember bool
		sort     data.SortMode
	}{
		{`{"rotation":"left"}`, true, data.SortUpdated},
		{`{"remember_sort":false,"last_sort":2}`, false, data.SortAlphabetical},
		{`{"remember_sort":true,"last_sort":1}`, true, data.SortDebut},
		{`{"last_sort":-1}`, true, data.SortUpdated},
		{`{"last_sort":999}`, true, data.SortUpdated},
		{`{"last_sort":3}`, true, data.SortFavorites},
	} {
		path := writeSettings(t, tc.body)
		s, err := LoadSettings(path)
		if err != nil || s.RememberSort != tc.remember || s.LastSort != tc.sort {
			t.Fatalf("load %s: %+v, %v", tc.body, s, err)
		}
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
		got, err := LoadSettings(path)
		if err != nil || got != s {
			t.Fatalf("restart lost preferences: %+v, %v", got, err)
		}
	}
	if !DefaultSettings().RememberSort {
		t.Fatal("new installations must remember sort by default")
	}
}

func TestFollowRotationMigration(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"rotation":"right"}`, true}, {`{"follow_ini_rotation":false,"rotation":"left"}`, false},
	} {
		p := writeSettings(t, tc.body)
		s, err := LoadSettings(p)
		if err != nil || s.FollowRotation != tc.want {
			t.Fatal(s, err)
		}
		if err := Save(p, s); err != nil {
			t.Fatal(err)
		}
		again, err := LoadSettings(p)
		if err != nil || again != s {
			t.Fatal("rotation setting did not persist", err)
		}
	}
	if !DefaultSettings().FollowRotation {
		t.Fatal("must default on")
	}
}

func TestFilterRotationDefaultAndSave(t *testing.T) {
	p := writeSettings(t, `{"rotation":"left"}`)
	s, err := LoadSettings(p)
	if err != nil || s.FilterRotation {
		t.Fatal("INI filter must default off")
	}
	s.FilterRotation = true
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	restored, err := LoadSettings(p)
	if err != nil || !restored.FilterRotation {
		t.Fatal("INI filter not saved")
	}
}

func TestHoldDelayMigrationAndSave(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"scroll":"60","rotation":"left"}`, 300},
		{`{"hold_delay_ms":0}`, 300},
		{`{"hold_delay_ms":999}`, 300},
		{`{"hold_delay_ms":200}`, 200},
		{`{"hold_delay_ms":500}`, 500},
	} {
		path := writeSettings(t, tc.body)
		s, err := LoadSettings(path)
		if err != nil || s.HoldDelay != tc.want {
			t.Fatalf("load %s: delay=%d err=%v", tc.body, s.HoldDelay, err)
		}
		s.HoldDelay = 500
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
		reloaded, err := LoadSettings(path)
		if err != nil || reloaded.HoldDelay != 500 || reloaded.Rotation != s.Rotation || reloaded.Scroll != s.Scroll {
			t.Fatalf("preferences did not survive restart: %+v err=%v", reloaded, err)
		}
	}
}

func TestScreensaverMigrationAndSave(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{`{"rotation":"left","inset":15}`, "1"},
		{`{"screensaver_minutes":"off"}`, "off"},
		{`{"screensaver_minutes":"5"}`, "5"},
		{`{"screensaver_minutes":"invalid"}`, "1"},
	} {
		path := writeSettings(t, tc.body)
		s, err := LoadSettings(path)
		if err != nil || s.Screensaver != tc.want {
			t.Fatalf("load %s: %+v, %v", tc.body, s, err)
		}
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
		got, err := LoadSettings(path)
		if err != nil || got != s {
			t.Fatalf("settings changed after save: %+v, %v", got, err)
		}
	}
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
