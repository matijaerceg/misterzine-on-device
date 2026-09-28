// Package beta is the switch between the free MisterZine and the Patreon
// beta, MisterZine Arcade. Both are built from the same source: the release
// build for members adds
//
//	-ldflags "-X github.com/matijaerceg/misterzine-on-device/internal/beta.Channel=beta"
//
// and every other build is the free version. Code that belongs only to the
// beta asks On() before it shows itself, so a feature goes public by deleting
// its check.
package beta

// Channel is "beta" in a beta build and empty in the free one.
var Channel = ""

// On reports whether this is the beta build.
func On() bool { return Channel == "beta" }

// Set switches the beta on or off for the harness or a test and returns the
// function that puts the previous value back. Tests that call it must not
// run in parallel.
func Set(on bool) (restore func()) {
	old := Channel
	Channel = ""
	if on {
		Channel = "beta"
	}
	return func() { Channel = old }
}
