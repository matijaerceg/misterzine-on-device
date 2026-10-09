package scan

import "sync"

// Gate holds a background walk while the screen is in motion and lets it go
// on when the motion ends, as ROMCheck.SetPaused does for the ROM sweep.
// The walks call Wait between files or rows; it costs a mutex when open.
type Gate struct {
	mu sync.Mutex
	ch chan struct{} // non-nil while paused; closed to release
}

// SetPaused opens or closes the gate.
func (g *Gate) SetPaused(p bool) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if p && g.ch == nil {
		g.ch = make(chan struct{})
	} else if !p && g.ch != nil {
		close(g.ch)
		g.ch = nil
	}
}

// Wait returns once the gate is open.
func (g *Gate) Wait() {
	if g == nil {
		return
	}
	for {
		g.mu.Lock()
		ch := g.ch
		g.mu.Unlock()
		if ch == nil {
			return
		}
		<-ch
	}
}

// Motion is the gate the card scan waits on: the host closes it while a
// held key scrolls the list or a transition runs, as the scan's four passes
// (core statuses, the alternatives walk, the family resolver, the local
// walk) each cost the boards frames when they run under the motion. Open
// unless the host says otherwise, so tools and tests never wait.
var Motion Gate
