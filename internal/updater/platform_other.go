//go:build !linux

package updater

func workerAlive(s State) bool { return false }

func restarted(s State) bool { return false }
