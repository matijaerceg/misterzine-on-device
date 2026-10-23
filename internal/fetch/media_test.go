package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matijaerceg/misterzine-on-device/internal/media"
)

func TestMediaURLsAndVideoCap(t *testing.T) {
	oldMedia := MediaService
	MediaService = ""
	defer func() { MediaService = oldMedia }()
	var path string
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		path = r.URL.Path
		if strings.Contains(path, "large-video") {
			w.Write(make([]byte, media.MaxBytes+1))
			return
		}
		if strings.Contains(path, "large-flyer") {
			w.Write(make([]byte, media.MaxFlyerBytes+1))
			return
		}
		w.Write([]byte("media"))
	}))
	defer srv.Close()
	old := Site
	Site = srv.URL
	defer func() { Site = old }()
	c := NewClient("test")

	if _, err := c.Image(context.Background(), SlotVideo, "demo"); err != nil || path != "/images/video/demo.mzv" {
		t.Fatalf("video: path=%q err=%v", path, err)
	}
	if _, err := c.Image(context.Background(), SlotFlyer, "demo"); err != nil || path != "/images/flyer/demo.png" {
		t.Fatalf("flyer: path=%q err=%v", path, err)
	}
	if _, err := c.Image(context.Background(), SlotVideo, "large-video"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize video err=%v", err)
	}
	if _, err := c.Image(context.Background(), SlotFlyer, "large-flyer"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize flyer err=%v", err)
	}
	before := requests
	for _, slot := range []string{SlotVideo, SlotFlyer} {
		if _, err := c.Image(context.Background(), slot, "../escape"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("invalid %s key err=%v", slot, err)
		}
	}
	if requests != before {
		t.Fatal("invalid media key reached the network")
	}
	MediaService = srv.URL + "/objects/"
	for _, tc := range []struct{ slot, path string }{
		{SlotVideo, "/objects/video/demo.mzv"},
		{SlotFlyer, "/objects/flyer/demo.png"},
		{"snap", "/images/snap/demo.png"},
	} {
		if _, err := c.Image(context.Background(), tc.slot, "demo"); err != nil || path != tc.path {
			t.Fatalf("media host %s: path=%q err=%v", tc.slot, path, err)
		}
	}
}
