//go:build linux

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
)

func (h *host) requestAppCheck() {
	if neoGeoPilot || h.appCheckRunning || time.Now().Before(h.nextAppCheck) {
		return
	}
	h.appCheckRunning = true
	// the beta follows its own track: the newest beta, never a free release
	check, url := buildinfo.CheckUpdate, buildinfo.LatestReleaseURL
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		version, err := check(ctx, http.DefaultClient, url, buildinfo.Version)
		h.runOnUI(func() { h.finishAppCheck(version, err) })
	}()
}

func (h *host) finishAppCheck(version string, err error) {
	h.appCheckRunning = false
	if err != nil {
		// Wi-Fi may still be connecting at boot. Retry soon, then
		// back off if the board stays offline.
		h.appCheckRetry = min(30*time.Minute, max(15*time.Second, 2*h.appCheckRetry))
		h.nextAppCheck = time.Now().Add(h.appCheckRetry)
		h.lg.Printf("app update check: %v", err)
		return
	}
	h.appCheckRetry = 0
	h.nextAppCheck = time.Now().Add(30 * time.Minute)
	h.a.SetAppUpdate(version)
}
