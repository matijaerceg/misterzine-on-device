// Package buildinfo carries the version stamped in at build time.
package buildinfo

// Set with -ldflags "-X .../buildinfo.Version=v0.1.0 -X .../buildinfo.Commit=abc -X .../buildinfo.Date=2026-09-08".
var (
	Version = "v1.0.25"
	Commit  = ""
	Date    = ""
)

// String is "v0.1.0 (abc1234, 2026-09-08)".
func String() string {
	s := Version
	if Commit != "" || Date != "" {
		s += " (" + Commit
		if Commit != "" && Date != "" {
			s += ", "
		}
		s += Date + ")"
	}
	return s
}
