// Package version provides build metadata injected via ldflags.
package version

// Values set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)
