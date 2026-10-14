package images

import (
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
)

func waitProgress(t *testing.T, s *Service, have, total int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		h, n := s.Progress()
		if h == have && n == total {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("progress %d/%d, want %d/%d", h, n, have, total)
		}
		time.Sleep(time.Millisecond)
	}
}

// Two rows sharing one screenshot used to put it in the list twice: the
// total counted both, the download only one, so the tally never filled up
// until a restart recounted the disk.
func TestPrefetchTallyCountsSharedPictureOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	}))
	defer server.Close()
	old := fetch.Site
	fetch.Site = server.URL
	defer func() { fetch.Site = old }()
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 0)
	defer s.Close()
	a, b := Pic{"kikcubic", "snap"}, Pic{"other", "snap"}
	s.SetPrefetch([]Pic{a, b, a}, true)
	waitProgress(t, s, 2, 2)
}

// A new list of the same length (a catalogue refresh swapping one game for
// another) must be recounted, not keep the old list's count.
func TestPrefetchTallyRecountsSameLengthList(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, nil, log.New(io.Discard, "", 0), 0)
	defer s.Close()
	a, b := Pic{"one", "snap"}, Pic{"two", "snap"}
	s.SetPrefetch([]Pic{a}, false)
	waitProgress(t, s, 0, 1)
	if err := os.WriteFile(filepath.Join(dir, "snap", "two.png"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	s.SetPrefetch([]Pic{b}, false)
	waitProgress(t, s, 1, 1)
}
