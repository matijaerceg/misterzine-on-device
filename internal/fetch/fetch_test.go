package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidHash(t *testing.T) {
	good := strings.Repeat("0123456789abcdef", 4)
	for _, h := range []string{"", "abc", strings.ToUpper(good), good + "0", strings.Repeat("g", 64)} {
		if validHash(h) {
			t.Errorf("%q accepted", h)
		}
	}
	if !validHash(good) {
		t.Error("a real hash rejected")
	}
}

// A malformed meta hash is an error, not a panic in a background goroutine.
func TestCheckShortHashIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "meta.json") {
			w.Write([]byte(`{"hash":"abc","rows":0,"updated":"2026-09-08T00:00Z"}`))
			return
		}
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	old := Site
	Site = srv.URL
	defer func() { Site = old }()
	c := NewClient("test")
	if _, err := c.Check(context.Background(), "other"); err == nil || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("err = %v", err)
	}
}
