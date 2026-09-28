// Package buildinfo carries the version stamped in at build time.
package buildinfo

import "github.com/matijaerceg/misterzine-on-device/internal/beta"

// Set with -ldflags "-X .../buildinfo.Version=v0.1.0 -X .../buildinfo.Commit=abc -X .../buildinfo.Date=2026-09-08".
var (
	Version = "v1.2.0"
	Commit  = ""
	Date    = ""
)

// String is "v0.1.0 (abc1234, 2026-09-08)", and in the Patreon beta
// "v0.1.0 (abc1234, 2026-09-08), beta batch arcade-1": every place that
// prints the version (the log, -version, Options, reports, the debug
// API) names the channel and batch with it.
func String() string {
	s := Version
	if Commit != "" || Date != "" {
		s += " (" + Commit
		if Commit != "" && Date != "" {
			s += ", "
		}
		s += Date + ")"
	}
	if b := beta.Describe(); b != "" {
		s += ", " + b
	}
	return s
}
