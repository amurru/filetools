package exclusions

import (
	"path/filepath"
	"strings"

	"amurru/filetools/internal/output"
)

// ExclusionMatcher defines the interface for matching exclusions
type ExclusionMatcher interface {
	Matches(path string, isDir bool) bool
	GetReason() string
}

// GlobMatcher matches paths using glob patterns
type GlobMatcher struct {
	pattern string
	reason  string
}

func (gm *GlobMatcher) Matches(path string, isDir bool) bool {
	matched, _ := filepath.Match(gm.pattern, filepath.Base(path))
	return matched
}

func (gm *GlobMatcher) GetReason() string {
	return gm.reason
}

// FileTypeMatcher matches files by extension
type FileTypeMatcher struct {
	extension string
}

func (ftm *FileTypeMatcher) Matches(path string, isDir bool) bool {
	if isDir {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == strings.ToLower(ftm.extension) || (ftm.extension == "*" && ext != "")
}

func (ftm *FileTypeMatcher) GetReason() string {
	return "file_type"
}

// ParseExclusions parses comma-separated patterns into ExclusionMatchers
func ParseExclusions(patterns string, isFile bool) []ExclusionMatcher {
	if patterns == "" {
		return nil
	}

	var matchers []ExclusionMatcher
	patternList := strings.Split(patterns, ",")

	for _, pattern := range patternList {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		if isFile {
			// Check if it's a file type pattern (starts with *)
			if strings.HasPrefix(pattern, "*.") {
				matchers = append(matchers, &FileTypeMatcher{extension: pattern[1:]})
			} else {
				matchers = append(matchers, &GlobMatcher{pattern: pattern, reason: "file_pattern"})
			}
		} else {
			matchers = append(matchers, &GlobMatcher{pattern: pattern, reason: "dir_pattern"})
		}
	}

	return matchers
}

// CheckExclusions checks if a path matches any exclusion and returns the exclusion if matched
func CheckExclusions(path string, isDir bool, fileMatchers, dirMatchers []ExclusionMatcher) *output.Exclusion {
	var matchers []ExclusionMatcher
	if isDir {
		matchers = dirMatchers
	} else {
		matchers = fileMatchers
	}

	for _, matcher := range matchers {
		if matcher.Matches(path, isDir) {
			return &output.Exclusion{
				Path:   path,
				Reason: matcher.GetReason(),
			}
		}
	}

	return nil
}
