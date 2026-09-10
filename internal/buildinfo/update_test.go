package buildinfo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewerStable(t *testing.T) {
	for _, tc := range []struct {
		next, current string
		want          bool
	}{
		{"v1.0.6", "v1.0.5", true}, {"v1.0.10", "v1.0.9", true},
		{"v1.1.0", "v1.0.99", true}, {"v1.0.5", "v1.0.5", false},
		{"v1.0.4", "v1.0.5", false}, {"v1.0.6-beta", "v1.0.5", false},
		{"v1.0.5", "v1.0.6-dev", false}, {"nonsense", "v1.0.5", false},
	} {
		if got := NewerStable(tc.next, tc.current); got != tc.want {
			t.Errorf("%+v: %v", tc, got)
		}
	}
}

func TestCheckUpdateResponses(t *testing.T) {
	for _, tc := range []struct {
		body   string
		code   int
		want   string
		failed bool
	}{
		{`{"tag_name":"v1.0.6"}`, 200, "v1.0.6", false},
		{`{"tag_name":"v1.0.6","prerelease":true}`, 200, "", false},
		{`{"tag_name":"v1.0.6","draft":true}`, 200, "", false},
		{`{"tag_name":"v1.0.4"}`, 200, "", false},
		{"bad json", 200, "", true}, {"rate limited", 403, "", true},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Error(r.Method)
			}
			w.WriteHeader(tc.code)
			w.Write([]byte(tc.body))
		}))
		got, err := CheckUpdate(context.Background(), srv.Client(), srv.URL, "v1.0.5")
		srv.Close()
		if got != tc.want || (err != nil) != tc.failed {
			t.Fatalf("%+v: %q %v", tc, got, err)
		}
	}
}
