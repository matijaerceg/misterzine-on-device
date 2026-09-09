package debugsrv

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDebugGuardRequiresAuthenticatedNonBrowserRequests(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, host, peer, token, origin string
		want                                          int
	}{
		{"valid key", "POST", "/api/key/start", "192.168.1.94:8195", "192.168.1.2:12", "secret", "", 204},
		{"valid read", "GET", "/api/state", "192.168.1.94:8195", "192.168.1.2:12", "secret", "", 204},
		{"no auth", "POST", "/api/quit", "192.168.1.94:8195", "192.168.1.2:12", "", "", 401},
		{"wrong auth", "GET", "/api/log", "192.168.1.94:8195", "192.168.1.2:12", "wrong", "", 401},
		{"image attack", "GET", "/api/key/start", "192.168.1.94:8195", "192.168.1.2:12", "secret", "", 405},
		{"form attack", "POST", "/api/quit", "192.168.1.94:8195", "192.168.1.2:12", "secret", "https://example.com", 403},
		{"rebind", "GET", "/api/state", "attacker.example:8195", "192.168.1.2:12", "secret", "", 403},
		{"public peer", "GET", "/api/state", "192.168.1.94:8195", "8.8.8.8:12", "secret", "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }), log.New(io.Discard, "", 0), "secret")
			r := httptest.NewRequest(tc.method, "http://"+tc.host+tc.path, nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-MisterZine-Token", tc.token)
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.want || called != (tc.want == 204) {
				t.Fatalf("status %d, called=%v", w.Code, called)
			}
		})
	}
}

func TestDebugTokenPersistsAndRejectsCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug-token")
	first, err := loadToken(path)
	if err != nil || len(first) != 64 {
		t.Fatalf("token generation: %v", err)
	}
	again, err := loadToken(path)
	if err != nil || again != first {
		t.Fatal("token changed on reopen")
	}
	if err = os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = loadToken(path); err == nil {
		t.Fatal("corrupt token accepted")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "broken" {
		t.Fatal("corrupt credential replaced")
	}
}
