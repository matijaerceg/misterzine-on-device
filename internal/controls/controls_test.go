package controls

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestControlsRoundTripAndRejectUnsafeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "controls.json")
	f, err := Load(path)
	if err != nil || len(f.Profiles) != 0 {
		t.Fatal(f, err)
	}
	f.Profiles["d209_0301"] = Profile{Panel: true, Style: "custom", Labels: map[string]string{"start": "Y"}, Bindings: map[string]uint16{"start": 44, "pageup": 2}}
	if err = Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatal(got, err)
	}
	clone := f.Clone()
	p := clone.Profiles["d209_0301"]
	p.Bindings["start"] = 46
	if f.Profiles["d209_0301"].Bindings["start"] != 44 {
		t.Fatal("clone aliases draft")
	}
	for _, body := range []string{
		`{"schema":2}`, `{"schema":1,"future":true}`, `{"schema":1} {}`,
		`{"schema":1,"profiles":{"bad":{}}}`,
		`{"schema":1,"profiles":{"0000_0000":{}}}`,
		`{"schema":1,"profiles":{"d209_0301":{"bindings":{"enter":88}}}}`,
		`{"schema":1,"profiles":{"d209_0301":{"bindings":{"enter":29,"back":29}}}}`,
		`{"schema":1,"profiles":{"d209_0301":{"bindings":{"quit":29}}}}`,
		`{"schema":1,"profiles":{"d209_0301":{"labels":{"start":"Yellow"}}}}`,
		`{"schema":1,"profiles":{"d209_0301":{"label_style":"future"}}}`,
	} {
		if err = os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = Load(path); err == nil {
			t.Fatalf("accepted %s", body)
		}
		after, _ := os.ReadFile(path)
		if string(after) != body {
			t.Fatal("failed read changed file")
		}
	}
}

func TestEffectiveMappingHonorsOverridesAfterMiSTerChanges(t *testing.T) {
	p := Profile{Bindings: map[string]uint16{"start": 29, "enter": 2}}
	base := PanelBase()
	base["tab"] = 29
	m := Effective(base, p)
	if m["tab"] != 0 || m["start"] != 29 || m["enter"] != 2 {
		t.Fatal(m)
	}
	for _, code := range []uint16{0, 88, 896, 65535} {
		if ValidCode(code) {
			t.Fatalf("unsafe capture %d", code)
		}
	}
	for _, code := range []uint16{29, 304, 773, 803} {
		if !ValidCode(code) {
			t.Fatalf("valid capture rejected %d", code)
		}
	}
}

func TestFullMappingRequiresAllMinimumInputs(t *testing.T) {
	for _, required := range []string{"up", "down", "left", "right", "enter", "back"} {
		m := PanelBase()
		m[required] = 0
		if ValidateInputs(testInputs(m)) == nil {
			t.Fatalf("missing %s accepted", required)
		}
	}
	m := PanelBase()
	for _, optional := range []string{"start", "tab", "space", "pageup", "pagedown", "select", "menu"} {
		m[optional] = 0
	}
	if err := ValidateInputs(testInputs(m)); err != nil {
		t.Fatal(err)
	}
}

func TestInstanceProfileRoundTrip(t *testing.T) {
	f := New()
	f.Profiles["045e_028e_0123456789abcdef"] = Profile{Reviewed: true, Bindings: map[string]uint16{"enter": 304, "back": 305}}
	path := filepath.Join(t.TempDir(), "controls.json")
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatal("instance profile not preserved", err)
	}
}

func TestPhysicalButtonNamesRoundTrip(t *testing.T) {
	f := New()
	f.Profiles["0001_0002"] = Profile{Style: "custom", NameBase: "neogeo", Names: map[uint16]string{304: "LP", 330: "Red"}}
	path := filepath.Join(t.TempDir(), "controls.json")
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatal(got, err)
	}
	clone := got.Clone()
	p := clone.Profiles["0001_0002"]
	p.Names[304] = "HP"
	if got.Profiles["0001_0002"].Names[304] != "LP" {
		t.Fatal("draft aliases saved names")
	}
	p.Names[304] = "a name far too long"
	if clone.Validate() == nil {
		t.Fatal("unbounded legend accepted")
	}
}

func TestVersionedInputsRoundTripAndRequiredValidation(t *testing.T) {
	f := New()
	p := Profile{MappingVersion: 2, BasicsComplete: true, Inputs: map[uint16]string{1: "right", 2: "left", 3: "up", 4: "down", 5: "enter", 6: "back", 7: "enter"}, Names: map[uint16]string{7: "Red"}}
	f.Profiles["0001_0002"] = p
	path := filepath.Join(t.TempDir(), "controls.json")
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(f, got) {
		t.Fatal(got, err)
	}
	clone := f.Clone()
	q := clone.Profiles["0001_0002"]
	delete(q.Inputs, 5)
	if err := clone.Validate(); err != nil {
		t.Fatal("alias should satisfy OK", err)
	}
	delete(q.Inputs, 7)
	if clone.Validate() == nil {
		t.Fatal("missing OK accepted")
	}
	if f.Profiles["0001_0002"].Inputs[5] != "enter" {
		t.Fatal("clone changed saved mapping")
	}
	q.MappingVersion = 3
	clone.Profiles["0001_0002"] = q
	if clone.Validate() == nil {
		t.Fatal("future version accepted")
	}
}

func testInputs(m map[string]uint16) map[uint16]string {
	out := map[uint16]string{}
	for id, c := range m {
		if c != 0 {
			out[c] = id
		}
	}
	return out
}
