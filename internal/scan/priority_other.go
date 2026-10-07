//go:build !linux

package scan

// LowerThreadPriority does nothing off Linux (see priority_linux.go).
func LowerThreadPriority() {}
