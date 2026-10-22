// Package scan contains the analysis engines of filetools.
//
// An engine takes an options struct, reports its progress through a
// progress.Indicator and returns a report structure. The engines know
// nothing about the command line interface or the graphical user interface,
// so both front ends share exactly the same behaviour.
//
// The engines are safe to call from a goroutine: they never mutate their
// options. They report the paths they cannot read through a warning on
// os.Stderr, so the command line interface shows them to the user.
package scan

import (
	"fmt"
	"os"

	"github.com/amurru/filetools/internal/exclusions"
)

// matchers holds the parsed exclusion patterns of a single run.
type matchers struct {
	files []exclusions.ExclusionMatcher
	dirs  []exclusions.ExclusionMatcher
}

// newMatchers parses the comma separated exclusion patterns of a run.
func newMatchers(excludeFile, excludeDir string) matchers {
	return matchers{
		files: exclusions.ParseExclusions(excludeFile, true),
		dirs:  exclusions.ParseExclusions(excludeDir, false),
	}
}

// validateDir checks that path exists and is a directory.
func validateDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}
