package app

import "testing"

// A vertical local game (Andromeda, an HBMame set) used to carry a bare 3:4
// size; the twice-the-picture cap then made its Details box 6 pixels wide
// and clipped the "no shot" placeholder. The scan now gives local games a
// 240x320 or 320x240 stand-in size.
func TestLocalVerticalShotBoxIsReadable(t *testing.T) {
	if w, h := fitBox(240, 320, 96, 72); w != 54 || h != 72 {
		t.Fatalf("vertical local box %dx%d, want 54x72", w, h)
	}
	if w, _ := fitBox(3, 4, 96, 72); w != 6 {
		t.Fatalf("the old 3:4 size gave %d px; this test documents the bug", w)
	}
}
