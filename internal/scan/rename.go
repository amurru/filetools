package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/progress"
	"amurru/filetools/internal/report"
)

// RenameOptions configures a rename run.
type RenameOptions struct {
	// Dir is the directory tree to scan.
	Dir string
	// Match is the glob pattern a file name must match.
	Match string
	// Sed is the sed style replacement expression, for example s/old/new/g.
	Sed string
	// DryRun reports the planned operations without touching the filesystem.
	DryRun bool
	// Overwrite allows replacing a file that already exists at the target.
	Overwrite bool
	// ExcludeFile and ExcludeDir are comma separated exclusion patterns.
	ExcludeFile string
	ExcludeDir  string
}

// Rename renames the files matching the configured pattern.
//
// The run is a dry run unless DryRun is false, which mirrors the safety
// default of the command line interface.
func Rename(opts RenameOptions, p progress.Indicator) (*report.RenameResult, error) {
	if err := validateDir(opts.Dir); err != nil {
		return nil, err
	}

	sedRegex, replacement, global, err := ParseSedExpression(opts.Sed)
	if err != nil {
		return nil, err
	}

	m := newMatchers(opts.ExcludeFile, opts.ExcludeDir)

	operations, exclusionsList, err := performRenames(opts, sedRegex, replacement, global, m, p)
	if err != nil {
		return nil, err
	}

	return &report.RenameResult{
		Operations: operations,
		DryRun:     opts.DryRun,
		Exclusions: exclusionsList,
	}, nil
}

// ParseSedExpression parses a sed-style replacement expression.
// Format: s/pattern/replacement/flags
// Supported flags: g (global)
func ParseSedExpression(expr string) (*regexp.Regexp, string, bool, error) {
	if !strings.HasPrefix(expr, "s/") {
		return nil, "", false, fmt.Errorf("invalid sed expression: must start with 's/'")
	}

	parts := strings.Split(expr[2:], "/")
	// "s/old/" has no replacement section, but "s/old//" replaces the
	// pattern with an empty string, which removes the pattern.
	if len(parts) < 2 || (len(parts) == 2 && parts[1] == "") {
		return nil, "", false, fmt.Errorf("invalid sed expression: missing replacement")
	}

	pattern := parts[0]
	replacement := parts[1]
	global := false

	if len(parts) > 2 {
		flags := parts[2]
		if strings.Contains(flags, "g") {
			global = true
		}
		// Other flags could be added here
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, "", false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re, replacement, global, nil
}

// performRenames traverses the directory and performs rename operations
func performRenames(opts RenameOptions, sedRegex *regexp.Regexp, replacement string, global bool, m matchers, p progress.Indicator) ([]report.RenameOperation, []report.Exclusion, error) {
	var operations []report.RenameOperation
	var exclusionsList []report.Exclusion

	// Single-pass walk: process files dynamically
	p.Start(-1, "Processing files for rename")

	err := filepath.WalkDir(opts.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == opts.Dir {
			return nil
		}

		isDir := d.IsDir()

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(opts.Dir, path)
		if err != nil {
			return err
		}

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, isDir, m.files, m.dirs); exclusion != nil {
			exclusionsList = append(exclusionsList, *exclusion)
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}

		if isDir {
			return nil // Skip directories for renaming
		}

		// Check if file matches the pattern
		oldName := d.Name()
		matched, err := filepath.Match(opts.Match, oldName)
		if err != nil {
			return fmt.Errorf("invalid match pattern: %w", err)
		}
		if !matched {
			return nil
		}

		// Apply sed replacement to the filename
		var newName string
		if global {
			newName = sedRegex.ReplaceAllString(oldName, replacement)
		} else {
			// Replace only the first occurrence
			loc := sedRegex.FindStringIndex(oldName)
			if loc != nil {
				newName = oldName[:loc[0]] + replacement + oldName[loc[1]:]
			} else {
				newName = oldName
			}
		}

		if oldName == newName {
			// No change needed
			return nil
		}

		newPath := filepath.Join(filepath.Dir(path), newName)

		op := report.RenameOperation{
			OldPath: relPath,
			NewPath: filepath.Join(filepath.Dir(relPath), newName),
		}

		// Check if target already exists
		if _, err := os.Stat(newPath); err == nil {
			if !opts.Overwrite {
				op.Error = "target file already exists"
			}
		} else if !os.IsNotExist(err) {
			op.Error = fmt.Sprintf("cannot check target: %v", err)
		}

		// Perform rename if not dry run and no error
		if !opts.DryRun && op.Error == "" {
			if err := os.Rename(path, newPath); err != nil {
				op.Error = fmt.Sprintf("rename failed: %v", err)
			}
		}

		operations = append(operations, op)
		p.Increment()
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return operations, exclusionsList, nil
}
