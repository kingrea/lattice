package tui

import "strings"

// Version is the LATTICE build version.
//
// Override at build time with:
// go build -ldflags "-X lattice/internal/tui.Version=vX.Y.Z"
var Version = "dev"

func titleWithVersion() string {
	version := strings.TrimSpace(Version)
	if version == "" {
		version = "dev"
	}

	return "LATTICE " + version
}
