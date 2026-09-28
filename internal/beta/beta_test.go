package beta

import "testing"

func TestFreeByDefault(t *testing.T) {
	if On() {
		t.Fatal("a build without the ldflag is the beta")
	}
	restore := Set(true)
	if !On() {
		t.Fatal("Set(true) left the beta off")
	}
	restore()
	if On() {
		t.Fatal("restore left the beta on")
	}
}
