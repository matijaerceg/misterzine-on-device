//go:build linux

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/buildinfo"
)

func (h *host) requestAppCheck() {
	if h.appCheckRunning || time.Now().Before(h.nextAppCheck) {
		return
	}
	h.appCheckRunning = true
	h.nextAppCheck = time.Now().Add(30 * time.Minute)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		version, err := buildinfo.CheckUpdate(ctx, http.DefaultClient, buildinfo.LatestReleaseURL, buildinfo.Version)
		h.runOnUI(func() {
			h.appCheckRunning = false
			if err != nil {
				h.lg.Printf("app update check: %v", err)
				return
			}
			h.a.SetAppUpdate(version)
		})
	}()
}
