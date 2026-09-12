package mister

import "time"

// awaitConsole polls the active console until it is want or timeout has
// passed. A VT_ACTIVATE is dropped by the kernel while the front console is
// in graphics mode with nobody owning VT switching (a frontend drawing on
// tty2), and VT_WAITACTIVE would then sleep for the rest of the boot; a
// bounded poll turns that into a reported failure.
func awaitConsole(want string, active func() string, pause func(time.Duration), timeout time.Duration) bool {
	const step = 50 * time.Millisecond
	for waited := time.Duration(0); ; waited += step {
		if active() == want {
			return true
		}
		if waited >= timeout {
			return false
		}
		pause(step)
	}
}
