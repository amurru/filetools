package exclusions

import (
	"testing"
)

func TestParseExclusions_Empty(t *testing.T) {
	matchers := ParseExclusions("", true)
	if matchers != nil {
		t.Errorf("Expected nil matchers for empty pattern, got %d", len(matchers))
	}
}

func TestParseExclusions_OnlyCommas(t *testing.T) {
	matchers := ParseExclusions(", ,", true)
	if len(matchers) != 0 {
		t.Errorf("Expected no matchers for blank patterns, got %d", len(matchers))
	}
}

func TestParseExclusions_FileGlobs(t *testing.T) {
	// Patterns starting with "*." are treated as file type matchers, so glob
	// patterns here deliberately avoid that prefix.
	matchers := ParseExclusions("app_*.txt, data-*", true)
	if len(matchers) != 2 {
		t.Fatalf("Expected 2 matchers, got %d", len(matchers))
	}

	for _, m := range matchers {
		if m.GetReason() != "file_pattern" {
			t.Errorf("Expected reason 'file_pattern', got '%s'", m.GetReason())
		}
	}
}

func TestParseExclusions_FileTypes(t *testing.T) {
	matchers := ParseExclusions("*.log", true)
	if len(matchers) != 1 {
		t.Fatalf("Expected 1 matcher, got %d", len(matchers))
	}

	if matchers[0].GetReason() != "file_type" {
		t.Errorf("Expected reason 'file_type', got '%s'", matchers[0].GetReason())
	}
}

func TestParseExclusions_DirPatterns(t *testing.T) {
	matchers := ParseExclusions("node_modules,*.git", false)
	if len(matchers) != 2 {
		t.Fatalf("Expected 2 matchers, got %d", len(matchers))
	}

	for _, m := range matchers {
		if m.GetReason() != "dir_pattern" {
			t.Errorf("Expected reason 'dir_pattern', got '%s'", m.GetReason())
		}
	}
}

func TestGlobMatcher_Matches(t *testing.T) {
	m := &GlobMatcher{pattern: "*.log", reason: "file_pattern"}

	if !m.Matches("app.log", false) {
		t.Error("Expected 'app.log' to match '*.log'")
	}
	if m.Matches("app.txt", false) {
		t.Error("Expected 'app.txt' not to match '*.log'")
	}
}

func TestGlobMatcher_MatchesBaseNameOnly(t *testing.T) {
	m := &GlobMatcher{pattern: "*.log", reason: "file_pattern"}

	// The pattern is applied to the base name, so a directory component
	// should not influence the match.
	if !m.Matches("logs/app.log", false) {
		t.Error("Expected 'logs/app.log' to match '*.log' on base name")
	}
}

func TestFileTypeMatcher_Matches(t *testing.T) {
	m := &FileTypeMatcher{extension: ".log"}

	if !m.Matches("app.log", false) {
		t.Error("Expected 'app.log' to match '.log'")
	}
	if !m.Matches("app.LOG", false) {
		t.Error("Expected extension matching to be case-insensitive")
	}
	if m.Matches("app.txt", false) {
		t.Error("Expected 'app.txt' not to match '.log'")
	}
}

func TestFileTypeMatcher_DoesNotMatchDirectories(t *testing.T) {
	m := &FileTypeMatcher{extension: ".log"}

	if m.Matches("somedir.log", true) {
		t.Error("Expected directories not to match file type patterns")
	}
}

func TestFileTypeMatcher_WildcardExtension(t *testing.T) {
	m := &FileTypeMatcher{extension: "*"}

	if !m.Matches("app.log", false) {
		t.Error("Expected '*' to match any file with an extension")
	}
	if m.Matches("README", false) {
		t.Error("Expected '*' not to match a file without an extension")
	}
}

func TestCheckExclusions_NoMatch(t *testing.T) {
	fileMatchers := ParseExclusions("*.log", true)
	dirMatchers := ParseExclusions("node_modules", false)

	exclusion := CheckExclusions("app.txt", false, fileMatchers, dirMatchers)
	if exclusion != nil {
		t.Errorf("Expected no exclusion, got %+v", *exclusion)
	}
}

func TestCheckExclusions_FileMatch(t *testing.T) {
	fileMatchers := ParseExclusions("*.log", true)

	exclusion := CheckExclusions("app.log", false, fileMatchers, nil)
	if exclusion == nil {
		t.Fatal("Expected exclusion for matching file")
	}
	if exclusion.Path != "app.log" {
		t.Errorf("Expected path 'app.log', got '%s'", exclusion.Path)
	}
	if exclusion.Reason != "file_type" {
		t.Errorf("Expected reason 'file_type', got '%s'", exclusion.Reason)
	}
}

func TestCheckExclusions_DirMatch(t *testing.T) {
	dirMatchers := ParseExclusions("node_modules", false)

	exclusion := CheckExclusions("node_modules", true, nil, dirMatchers)
	if exclusion == nil {
		t.Fatal("Expected exclusion for matching directory")
	}
	if exclusion.Reason != "dir_pattern" {
		t.Errorf("Expected reason 'dir_pattern', got '%s'", exclusion.Reason)
	}
}

func TestCheckExclusions_DirMatchersNotAppliedToFiles(t *testing.T) {
	dirMatchers := ParseExclusions("*.log", false)

	exclusion := CheckExclusions("app.log", false, nil, dirMatchers)
	if exclusion != nil {
		t.Errorf("Expected directory patterns to be ignored for files, got %+v", *exclusion)
	}
}

func TestCheckExclusions_FileMatchersNotAppliedToDirs(t *testing.T) {
	fileMatchers := ParseExclusions("*.log", true)

	exclusion := CheckExclusions("app.log", true, fileMatchers, nil)
	if exclusion != nil {
		t.Errorf("Expected file patterns to be ignored for directories, got %+v", *exclusion)
	}
}
