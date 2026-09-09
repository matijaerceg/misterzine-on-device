//go:build linux

package main

import (
	"time"

	"github.com/matijaerceg/misterzine-on-device/internal/updater"
)

type updateResult struct {
	state    updater.State
	start    bool
	cancelID string
	err      error
}

func (h *host) initUpdates() {
	h.updates = make(chan updateResult, 4)
	if s, err := updater.Read(h.root); err == nil {
		h.a.SetUpdate(s, updater.ShouldOpen(h.root, s))
		h.updateRunning = s.Active()
		h.img.SetPaused(h.updateRunning)
	}
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-h.quit:
				return
			case <-t.C:
				if s, err := updater.Read(h.root); err == nil {
					select {
					case h.updates <- updateResult{state: s}:
					default:
					}
				}
			}
		}
	}()
}

func (h *host) startUpdate() {
	if h.updatePending {
		return
	}
	h.updatePending = true
	go func() {
		s, err := updater.Start(h.root, h.card)
		select {
		case h.updates <- updateResult{state: s, start: true, err: err}:
		case <-h.quit:
		}
	}()
}

func (h *host) cancelUpdate(id string) {
	if err := updater.Cancel(h.root, id); err != nil {
		h.lg.Printf("update cancel: %v", err)
		select {
		case h.updates <- updateResult{cancelID: id, err: err}:
		case <-h.quit:
		}
	}
}

func (h *host) dismissUpdate(id string) {
	if err := updater.Acknowledge(h.root, id); err != nil {
		h.lg.Printf("update dismissal: %v", err)
		h.a.Notice("Dismissal not saved; shown next launch", 8*time.Second)
	}
}

func (h *host) receiveUpdate(u updateResult) {
	if u.cancelID != "" {
		if h.a.UpdateState().ID == u.cancelID {
			h.a.UpdateCancelError(u.err.Error())
		}
		return
	}
	if h.updatePending && !u.start {
		return
	}
	if u.start {
		h.updatePending = false
	}
	if u.err != nil {
		u.state = updater.State{Status: "failed", Label: "Update All did not start", Message: u.err.Error()}
	}
	// Ignore snapshots of a prior run after a new start failed.
	if !u.start && h.a.UpdateState().ID == "" && !u.state.Active() {
		return
	}
	if current := h.a.UpdateState(); !u.start && current.Active() && current.ID != "" && u.state.ID != current.ID {
		return // a queued snapshot from before this run started
	}
	h.a.SetUpdate(u.state, u.start || u.state.Active() && !h.updateRunning)
	active := u.state.Active()
	finished := h.updateRunning && !active
	if active != h.updateRunning {
		h.img.SetPaused(active)
	}
	h.updateRunning = active
	if finished || (!active && h.scanPending) {
		h.requestScan()
	}
	if !active && h.checkPending {
		h.requestCheck()
	}
}
