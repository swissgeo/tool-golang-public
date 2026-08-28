package version //nolint:revive,nolintlint

import (
	"runtime/debug"
)

func GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown" // Return fallback if build info is not available
	}

	// If built with `go install module@version`
	if info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	// Fallback to VCS info
	var r = "unknown"
	var m string
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			r = s.Value
		}
		if s.Key == "vcs.modified" && s.Value == "true" {
			m = "-dirty"
		}
	}
	return r + m
}
