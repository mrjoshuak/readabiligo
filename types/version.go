// Package types provides the core data structures for the ReadabiliGo library.
package types

// Version information for the ReadabiliGo library.
const (
	Version = "0.2.0"
	Name    = "ReadabiliGo"
)

// BuildInfo contains version and build information for the ReadabiliGo library.
// It includes the version number, name, and Go version used to build the library.
type BuildInfo struct {
	Version   string
	Name      string
	GoVersion string
}

// GetBuildInfo returns the current version information for the ReadabiliGo library.
// This is useful for displaying version information in logs or help output.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Name:      Name,
		GoVersion: "go1.22", // TODO: Make this dynamic
	}
}
