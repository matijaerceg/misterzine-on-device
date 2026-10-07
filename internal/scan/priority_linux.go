//go:build linux

package scan

import "syscall"

// LowerThreadPriority gives the calling thread a nice value of 10, so the
// kernel runs it only when the threads drawing the screen (which start at
// -10) leave a core free. Call it from a goroutine that has locked itself
// to its thread with runtime.LockOSThread and never unlocks: the thread is
// then destroyed when the goroutine returns, and its nice value never
// reaches another goroutine. On Linux PRIO_PROCESS with a thread id names
// that one thread.
func LowerThreadPriority() {
	syscall.Setpriority(syscall.PRIO_PROCESS, syscall.Gettid(), 10)
}
