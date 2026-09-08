//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "misterzine runs on the MiSTer (Linux/ARM); use cmd/mzharness on this machine")
	os.Exit(2)
}
