package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTablesURL(t *testing.T) {
	var path, query string
	missing := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.Query().Get("t")
		if missing {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"version":1}`))
	}))
	defer srv.Close()
	old := Site
	defer func() { Site = old }()
	Site = srv.URL
	c := NewClient("test")
	b, err := c.Tables(context.Background())
	if err != nil || string(b) != `{"version":1}` || path != "/releases/app.json" || query == "" {
		t.Fatalf("got %q, %v from %s?t=%s", b, err, path, query)
	}
	// a site without the file yet: the caller keeps its tables
	missing = true
	if _, err := c.Tables(context.Background()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing file: %v", err)
	}
}
