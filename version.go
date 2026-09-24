package mappe

import "github.com/maloquacious/semver"

var (
	version = semver.Version{
		Major: 1,
		Minor: 0,
		Patch: 0,

		// Automatically populate build metadata with commit info
		Build: semver.Commit(), // Uses Git commit hash from build info
	}
)

func Version() semver.Version {
	return version
}
