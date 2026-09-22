package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendReport(t *testing.T) {
	var got []byte
	var gotType, gotUA string
	status, answer := http.StatusCreated, `{"code":"K7Q2"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/reports" {
			http.NotFound(w, r)
			return
		}
		got, _ = io.ReadAll(r.Body)
		gotType, gotUA = r.Header.Get("Content-Type"), r.Header.Get("User-Agent")
		w.WriteHeader(status)
		io.WriteString(w, answer)
	}))
	defer srv.Close()
	old := ReportService
	ReportService = srv.URL
	defer func() { ReportService = old }()
	c := NewClient("v1.1.2-test")

	code, err := c.SendReport(context.Background(), []byte("MisterZine report v1\nApp: test\n"))
	if err != nil || code != "K7Q2" {
		t.Fatalf("code %q, err %v", code, err)
	}
	if string(got) != "MisterZine report v1\nApp: test\n" || !strings.HasPrefix(gotType, "text/plain") || gotUA != "misterzine-on-device/v1.1.2-test" {
		t.Fatalf("sent %q as %q by %q", got, gotType, gotUA)
	}
	// the eight-character codes of the first service still read
	answer = `{"code":"7K2Q9XMB"}`
	if code, err := c.SendReport(context.Background(), []byte("x")); err != nil || code != "7K2Q9XMB" {
		t.Fatalf("an eight-character code: %q, %v", code, err)
	}
	for _, bad := range []string{`{"code":"K7Q"}`, `{"code":"K7Q2K7Q2K"}`, `{"code":"k7q2"}`} {
		answer = bad
		if _, err := c.SendReport(context.Background(), []byte("x")); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	answer = `{"code":"K7Q2"}`

	for _, c2 := range []struct {
		status int
		answer string
		want   error
	}{
		{http.StatusRequestEntityTooLarge, `{"error":"too_large"}`, ErrReportTooLarge},
		{http.StatusTooManyRequests, `{"error":"rate_limited"}`, ErrReportBusy},
		{http.StatusServiceUnavailable, `{"error":"disabled"}`, ErrReportOff},
	} {
		status, answer = c2.status, c2.answer
		if _, err := c.SendReport(context.Background(), []byte("x")); !errors.Is(err, c2.want) {
			t.Errorf("HTTP %d: %v, want %v", c2.status, err, c2.want)
		}
	}
	status, answer = http.StatusInternalServerError, `{"error":"internal"}`
	if _, err := c.SendReport(context.Background(), []byte("x")); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("HTTP 500: %v", err)
	}
	status, answer = http.StatusCreated, `{"code":"../etc"}`
	if _, err := c.SendReport(context.Background(), []byte("x")); err == nil {
		t.Error("a malformed code was accepted")
	}

	srv.Close()
	if _, err := c.SendReport(context.Background(), []byte("x")); !errors.Is(err, ErrOffline) {
		t.Errorf("service down: %v, want offline", err)
	}
	ReportService = ""
	if _, err := c.SendReport(context.Background(), []byte("x")); !errors.Is(err, ErrReportOff) {
		t.Errorf("no service: %v", err)
	}
}
