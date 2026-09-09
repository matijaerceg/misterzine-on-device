//go:build linux

package main

import (
	"os"
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
	} else {
		h.updateReadFailure(err)
	}
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-h.quit:
				return
			case <-t.C:
				s, err := updater.Read(h.root)
				select {
				case h.updates <- updateResult{state: s, err: err}:
				default:
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
	if !u.start && u.err != nil {
		h.updateReadFailure(u.err)
		return
	}
	h.updateReadError = ""
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
	h.applyUpdate(u.state, u.start || u.state.Active() && !h.updateRunning)
}

func (h *host) updateReadFailure(err error) {
	if os.IsNotExist(err) && !h.updateRunning {
		return
	}
	if message := err.Error(); message != h.updateReadError {
		h.updateReadError = message
		h.lg.Printf("update status: %v", err)
		h.a.Notice("Update status unreadable; see log", 8*time.Second)
	}
	if h.updateRunning {
		// Keep the last status while its supervisor lives. If it has exited, do not
		// strand the user in a modal waiting for a file that may never be written.
		s := updater.ReconcileWorker(h.a.UpdateState())
		h.applyUpdate(s, false)
	}
}

func (h *host) applyUpdate(s updater.State, open bool) {
	h.a.SetUpdate(s, open)
	active := s.Active()
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
