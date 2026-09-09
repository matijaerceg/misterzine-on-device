package images

import (
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/app"
	"github.com/matijaerceg/misterzine-on-device/internal/fetch"
)

// Keep failure cleanup bounded too: removing the wanted work lets the old,
// broken workers reach their idle stop check after a regression is reported.
func closeCheck(t *testing.T, s *Service) func() {
	t.Helper()
	var once sync.Once
	done := make(chan struct{})
	start := func() { once.Do(func() { go func() { s.Close(); close(done) }() }) }
	t.Cleanup(func() {
		s.Want(nil)
		start()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("image workers did not stop during test cleanup")
		}
	})
	return func() {
		t.Helper()
		start()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Close did not stop busy image workers within 2 seconds")
		}
	}
}

func TestCloseWithPendingDownload(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	oldSite := fetch.Site
	fetch.Site = server.URL
	t.Cleanup(func() { fetch.Site = oldSite })
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 0)
	closeAndCheck := closeCheck(t, s)
	s.Want([]app.ImageReq{{Key: "pending", Slot: "snap", W: 32, H: 32}})
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("image download never started")
	}
	closeAndCheck()
	p := Pic{Key: "pending", Slot: "snap"}
	if s.missing[p] || s.failed[p] || !s.retryAt[p].IsZero() {
		t.Fatal("cancelling shutdown must not blacklist or back off a picture")
	}
}

func TestCloseWithUnreadableCachedImage(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 0)
	closeAndCheck := closeCheck(t, s)
	req := app.ImageReq{Key: "unreadable", Slot: "snap", W: 32, H: 32}
	k := scaledKey{Pic{req.Key, req.Slot}, req.W, req.H, false, false}
	// A directory deterministically models stat succeeding but ReadFile
	// failing, on both Windows and Linux, without requiring a damaged card.
	if err := os.Mkdir(s.path(k.Pic), 0755); err != nil {
		t.Fatal(err)
	}
	s.Want([]app.ImageReq{req})
	deadline := time.Now().Add(2 * time.Second)
	for {
		s.mu.Lock()
		busy := s.inflight[k] || s.failed[k.Pic]
		s.mu.Unlock()
		if busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("decoder never started reading the cached image")
		}
		time.Sleep(time.Millisecond)
	}
	closeAndCheck()
}

func TestUnreadableImageIsSkippedWithoutDeletingIt(t *testing.T) {
	s := New(t.TempDir(), nil, log.New(io.Discard, "", 0), 0)
	defer s.Close()
	req := app.ImageReq{Key: "bad", Slot: "snap", W: 32, H: 32}
	p := Pic{req.Key, req.Slot}
	if err := os.Mkdir(s.path(p), 0755); err != nil {
		t.Fatal(err)
	}
	s.Want([]app.ImageReq{req})
	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("read error not reported")
	}
	if _, state := s.Get(req); state != app.ImageMissing {
		t.Fatalf("state=%v", state)
	}
	if _, ok := s.nextDecode(); ok {
		t.Fatal("unreadable image selected repeatedly")
	}
	if _, err := os.Stat(s.path(p)); err != nil {
		t.Fatal("unreadable original removed")
	}
}

func TestPrefetchRevisitsDelayedDownloads(t *testing.T) {
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 0)
	s.Close() // inspect scheduling without racing live worker selection
	first, second := Pic{"first", "snap"}, Pic{"second", "snap"}
	s.prefetch = []Pic{first, second}
	s.prefetchOn = true
	s.retryAt[first] = time.Now().Add(time.Minute)
	got, ok := s.nextDownload()
	if !ok || got != second {
		t.Fatal("delayed image blocked later prefetch")
	}
	delete(s.netBusy, second)
	s.missing[second] = true
	if _, ok := s.nextDownload(); ok {
		t.Fatal("retried before backoff expired")
	}
	s.retryAt[first] = time.Now().Add(-time.Second)
	if got, ok := s.nextDownload(); !ok || got != first {
		t.Fatal("delayed image lost after cursor passed it")
	}
	delete(s.netBusy, first)
	s.failed[first] = true
	if _, ok := s.nextDownload(); ok {
		t.Fatal("decoder failure should not be re-downloaded repeatedly")
	}
	s.SetOffline(true)
	s.SetOffline(false)
	if s.prefetchPos != 0 {
		t.Fatal("reconnect did not revisit interrupted prefetch")
	}
}

func TestPrefetchDownloadsAfterTemporaryServerFailure(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "try later", 500)
			return
		}
		png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	}))
	defer server.Close()
	old := fetch.Site
	fetch.Site = server.URL
	defer func() { fetch.Site = old }()
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 0)
	defer s.Close()
	p := Pic{"retry", "snap"}
	s.SetPrefetch([]Pic{p}, true)
	deadline := time.Now().Add(3 * time.Second)
	for {
		s.mu.Lock()
		waiting := !s.retryAt[p].IsZero() && !s.netBusy[p]
		if waiting {
			s.retryAt[p] = time.Now().Add(-time.Second)
		}
		s.mu.Unlock()
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("failure did not schedule backoff")
		}
		time.Sleep(time.Millisecond)
	}
	s.poke(s.kick)
	for {
		if have, total := s.Progress(); have == 1 && total == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("prefetch did not resume without being toggled")
		}
		time.Sleep(time.Millisecond)
	}
	if requests.Load() != 2 {
		t.Fatalf("request count=%d", requests.Load())
	}
	f, err := os.Open(s.path(p))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadProgressDoesNotSignalDrawableImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { png.Encode(w, image.NewRGBA(image.Rect(0, 0, 4, 4))) }))
	defer server.Close()
	oldSite := fetch.Site
	fetch.Site = server.URL
	defer func() { fetch.Site = oldSite }()
	s := New(t.TempDir(), fetch.NewClient("test"), log.New(io.Discard, "", 0), 0)
	defer s.Close()
	// No wanted image: completing a background download only changes Options.
	s.download(Pic{Key: "background", Slot: "snap"})
	select {
	case <-s.Ready():
		t.Fatal("background download requested full repaint")
	default:
	}
	select {
	case <-s.ProgressReady():
	default:
		t.Fatal("Options progress was not signalled")
	}
	// The same cached image becomes drawable only after it is requested/decoded.
	s.Want([]app.ImageReq{{Key: "background", Slot: "snap", W: 4, H: 4}})
	select {
	case <-s.Ready():
	case <-time.After(2 * time.Second):
		t.Fatal("decoded picture did not request repaint")
	}
}
