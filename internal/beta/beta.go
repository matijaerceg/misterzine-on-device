// Package beta is the switch between the free MisterZine and the Patreon
// beta, MisterZine Arcade. Both are built from the same source: the release
// build for members adds
//
//	-ldflags "-X github.com/matijaerceg/misterzine-on-device/internal/beta.Channel=beta"
//
// and every other build is the free version. Code that belongs only to the
// beta asks On() before it shows itself, so a feature goes public by deleting
// its check.
//
// A members' release also sets Batch and CodeSHA256 (code.go), which lock
// the whole app behind the six-digit code from the Patreon post.
package beta

// Channel is "beta" in a beta build and empty in the free one.
var Channel = ""

// Batch names the group of releases one code opens, such as "arcade-1":
// lowercase letters, digits and dashes. Empty in the free build.
var Batch = ""

// CodeSHA256 is the SHA-256 of the batch's six-digit code as 64 lowercase
// hex digits, never the code itself. Empty leaves a beta build unlocked:
// development builds and the harness's -beta.
var CodeSHA256 = ""

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
