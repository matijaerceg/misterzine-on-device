package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalSnapURL(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Path)
		w.Write([]byte("png"))
	}))
	defer srv.Close()
	oldSite, oldSnap := Site, SnapService
	defer func() { Site, SnapService = oldSite, oldSnap }()
	Site, SnapService = srv.URL, ""
	c := NewClient("test")

	// Service unset: a local snap is missing without any request.
	if _, err := c.Image(context.Background(), SlotLocalSnap, "orphan"); !errors.Is(err, ErrNotFound) || len(got) != 0 {
		t.Fatalf("unset service: err=%v requests=%v", err, got)
	}
	// Catalogue slots still go to the site.
	if _, err := c.Image(context.Background(), "snap", "colony7"); err != nil || got[0] != "/images/snap/colony7.png" {
		t.Fatalf("site snap: %v %v", err, got)
	}
	if _, err := c.Image(context.Background(), "system", "C64"); err != nil || got[1] != "/images/systems/C64.png" {
		t.Fatalf("system photo: %v %v", err, got)
	}
	SnapService = srv.URL + "/svc"
	if _, err := c.Image(context.Background(), SlotLocalSnap, "orphan"); err != nil || got[2] != "/svc/snap/orphan.png" {
		t.Fatalf("service snap: %v %v", err, got)
	}
}
