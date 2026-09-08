// Package debugsrv is the LAN debug channel: it injects keys into the app's
// event stream, dumps the canvas as PNG and reports state, so the whole dev
// loop runs from a workstation with nobody at the device. Off unless the
// wrapper finds a debug flag file; private-network peers only; every call
// is logged.
package debugsrv

import (
	"encoding/json"
	"image"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/platform"
)

// Hooks are what the server needs from the host. Run executes a function
// on the UI goroutine and waits for it; Inject feeds an event as if from
// the pad; Quit asks the app to exit.
type Hooks struct {
	Run    func(f func())
	Inject func(ev platform.Event)
	Shot   func() *image.RGBA // called via Run
	State  func() any         // called via Run
	Quit   func()
	Goto   func(k string) // called via Run: put the cursor on a row key
	Log    string         // log file path
}

// Serve starts the server; it returns immediately.
func Serve(addr string, h Hooks, lg *log.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/shot.png", func(w http.ResponseWriter, r *http.Request) {
		var img *image.RGBA
		h.Run(func() { img = h.Shot() })
		if img == nil {
			http.Error(w, "no frame", 500)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		var st any
		h.Run(func() { st = h.State() })
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(st)
	})
	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST a JSON array of {key,hold_ms,gap_ms,count}", 405)
			return
		}
		var reqs []struct {
			Key   string `json:"key"`
			Hold  int    `json:"hold_ms"`
			Gap   int    `json:"gap_ms"`
			Count int    `json:"count"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		for _, q := range reqs {
			k := platform.ParseKey(q.Key)
			if k == platform.KeyNone {
				http.Error(w, "unknown key "+q.Key, 400)
				return
			}
			if q.Count < 1 {
				q.Count = 1
			}
			if q.Hold < 1 {
				q.Hold = 40
			}
			if q.Gap < 1 {
				q.Gap = 60
			}
			for i := 0; i < q.Count; i++ {
				h.Inject(platform.Event{Key: k, Pressed: true, At: time.Now(), Source: "debug"})
				time.Sleep(time.Duration(q.Hold) * time.Millisecond)
				h.Inject(platform.Event{Key: k, Pressed: false, At: time.Now(), Source: "debug"})
				time.Sleep(time.Duration(q.Gap) * time.Millisecond)
			}
		}
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/key/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/key/")
		k := platform.ParseKey(name)
		if k == platform.KeyNone {
			http.Error(w, "unknown key", 400)
			return
		}
		hold, _ := strconv.Atoi(r.URL.Query().Get("hold"))
		if hold < 1 {
			hold = 40
		}
		count, _ := strconv.Atoi(r.URL.Query().Get("count"))
		if count < 1 {
			count = 1
		}
		for i := 0; i < count; i++ {
			h.Inject(platform.Event{Key: k, Pressed: true, At: time.Now(), Source: "debug"})
			time.Sleep(time.Duration(hold) * time.Millisecond)
			h.Inject(platform.Event{Key: k, Pressed: false, At: time.Now(), Source: "debug"})
			time.Sleep(60 * time.Millisecond)
		}
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/log", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("tail"))
		if n < 1 {
			n = 100
		}
		b, err := os.ReadFile(h.Log)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if len(lines) > n {
			lines = lines[len(lines)-n:]
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Join(lines, "\n") + "\n"))
	})
	mux.HandleFunc("/api/goto", func(w http.ResponseWriter, r *http.Request) {
		k := r.URL.Query().Get("k")
		if k == "" || h.Goto == nil {
			http.Error(w, "need ?k=<row key>", 400)
			return
		}
		h.Run(func() { h.Goto(k) })
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/stack", func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		w.Header().Set("Content-Type", "text/plain")
		w.Write(buf[:n])
	})
	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bye\n"))
		go h.Quit()
	})
	srv := &http.Server{Addr: addr, Handler: guard(mux, lg), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		lg.Printf("debug: listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lg.Printf("debug: %v", err)
		}
	}()
}

// guard allows private-network peers only and logs every call.
func guard(next http.Handler, lg *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if ip == nil || !(ip.IsLoopback() || ip.IsPrivate()) {
			http.Error(w, "forbidden", 403)
			return
		}
		lg.Printf("debug: %s %s from %s", r.Method, r.URL.Path, host)
		next.ServeHTTP(w, r)
	})
}
