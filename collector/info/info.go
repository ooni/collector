package info

import "fmt"

var (
	// Version is the full version number or dev
	Version = "dev"
	// CommitHash is the hash of the git commit
	CommitHash = "none"
	// BuildDate of when the build was made
	BuildDate = "unknown"
)

// FullVersionString returns a formatted version string
func FullVersionString() string {
	return fmt.Sprintf("%s (%s) %s", Version, CommitHash, BuildDate)
}
