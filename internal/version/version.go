// Package version exposes the build metadata of the filetools binaries.
//
// The values are injected at link time by the Makefile and by the release
// pipeline. They are exported so that the command line interface and the
// graphical user interface report the same build.
package version

var (
	// Version is the release version of the build.
	Version string
	// Commit is the git commit the build was made from.
	Commit string
	// CommitDate is the commit date of the build.
	CommitDate string
	// TreeState reports whether the working tree was clean at build time.
	TreeState string
)

// String returns the version, falling back to "dev" for unreleased builds.
func String() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// CommitString returns the commit, falling back to "none" for builds made
// outside of a git checkout.
func CommitString() string {
	if Commit == "" {
		return "none"
	}
	return Commit
}

// CommitDateString returns the commit date, falling back to "unknown".
func CommitDateString() string {
	if CommitDate == "" {
		return "unknown"
	}
	return CommitDate
}

// TreeStateString returns the tree state, falling back to "unknown".
func TreeStateString() string {
	if TreeState == "" {
		return "unknown"
	}
	return TreeState
}
