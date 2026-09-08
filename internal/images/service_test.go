package images

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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
		busy := s.inflight[k]
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
