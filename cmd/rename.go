package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"amurru/filetools/internal/exclusions"
	"amurru/filetools/internal/output"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:   "rename [directory]",
	Short: "Rename files in a directory using pattern matching and sed-like replacements",
	Long: `Rename files in a directory using pattern matching and sed-like replacements.

This command will traverse the specified directory (or current directory if none provided)
and rename files that match the given pattern using a sed-style replacement expression.

Note: Glob patterns must be quoted to prevent shell expansion.

Examples:
    # Add prefix to all JPG files
    filetools rename --match "*.jpg" --sed "s/^/vacation_/" /photos

    # Remove suffix from files
    filetools rename --match "*_old.jpg" --sed "s/_old//" /photos

    # General replacement
    filetools rename --match "*.txt" --sed "s/draft/final/g" /docs

Dry-run mode is enabled by default for safety. Use --force to perform actual renames.
`,
	Run: runRename,
}

var (
	matchPattern   string
	sedExpression  string
	forceOverwrite bool
)

func init() {
	rootCmd.AddCommand(renameCmd)

	// Command-specific flags
	renameCmd.Flags().StringVar(&matchPattern, "match", "", "File pattern to match (glob, required)")
	renameCmd.Flags().StringVar(&sedExpression, "sed", "", "Sed-style replacement expression (e.g., s/old/new/g, required)")
	renameCmd.Flags().BoolVar(&forceOverwrite, "force", false, "Perform actual renames (disables dry-run)")
	renameCmd.MarkFlagRequired("match")
	renameCmd.MarkFlagRequired("sed")
}

// runRename executes the rename command
func runRename(cmd *cobra.Command, args []string) {
	rootDir := "."
	if len(args) > 0 {
		rootDir = args[0]
	}

	// Determine if dry run
	isDryRun := !forceOverwrite

	// Validate directory exists
	if info, err := os.Stat(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	} else if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", rootDir)
		os.Exit(1)
	}

	// Parse sed expression
	sedRegex, replacement, global, err := parseSedExpression(sedExpression)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing sed expression: %v\n", err)
		os.Exit(1)
	}

	// Parse exclusion patterns
	fileMatchers := exclusions.ParseExclusions(excludeFilePatterns, true)
	dirMatchers := exclusions.ParseExclusions(excludeDirPatterns, false)

	// Perform rename operations
	operations, exclusionsList, err := performRenames(rootDir, matchPattern, sedRegex, replacement, global, isDryRun, forceOverwrite, fileMatchers, dirMatchers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error performing renames: %v\n", err)
		os.Exit(1)
	}

	// Get output writer (file or stdout)
	writer, cleanup, err := getOutputWriter(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Get output format and create formatter
	format := getOutputFormat(cmd)
	formatter := output.NewFormatter(format)

	// Create metadata
	flags := []output.Flag{
		{Name: "match", Value: matchPattern},
		{Name: "sed", Value: sedExpression},
		{Name: "dry-run", Value: fmt.Sprintf("%t", isDryRun)},
	}

	// Add output flag
	flags = append(flags, output.Flag{Name: "output", Value: string(format)})

	// Add file flag if specified
	if outputFile != "" {
		flags = append(flags, output.Flag{Name: "file", Value: outputFile})
	}

	// Add exclusion flags if specified
	if excludeFilePatterns != "" {
		flags = append(flags, output.Flag{Name: "exclude-file", Value: excludeFilePatterns})
	}
	if excludeDirPatterns != "" {
		flags = append(flags, output.Flag{Name: "exclude-dir", Value: excludeDirPatterns})
	}

	metadata := &output.Metadata{
		ToolName:    "filetools",
		SubCommand:  "rename",
		Flags:       flags,
		Version:     version,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	result := &output.RenameResult{
		Operations: operations,
		DryRun:     isDryRun,
		Exclusions: exclusionsList,
		Metadata:   metadata,
	}

	// Output the results
	if err := formatter.FormatRename(result, writer); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}
}

// parseSedExpression parses a sed-style replacement expression
// Format: s/pattern/replacement/flags
// Supported flags: g (global)
func parseSedExpression(expr string) (*regexp.Regexp, string, bool, error) {
	if !strings.HasPrefix(expr, "s/") {
		return nil, "", false, fmt.Errorf("invalid sed expression: must start with 's/'")
	}

	parts := strings.Split(expr[2:], "/")
	if len(parts) < 2 || parts[1] == "" {
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
func performRenames(rootDir, matchPattern string, sedRegex *regexp.Regexp, replacement string, global bool, isDryRun, forceOverwrite bool, fileMatchers, dirMatchers []exclusions.ExclusionMatcher) ([]output.RenameOperation, []output.Exclusion, error) {
	var operations []output.RenameOperation
	var exclusionsList []output.Exclusion

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not access %s: %v\n", path, err)
			return nil
		}

		// Skip the root directory itself
		if path == rootDir {
			return nil
		}

		// Get relative path for exclusion checking
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Check for exclusions
		if exclusion := exclusions.CheckExclusions(relPath, info.IsDir(), fileMatchers, dirMatchers); exclusion != nil {
			exclusionsList = append(exclusionsList, *exclusion)
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil // Skip directories for renaming
		}

		// Check if file matches the pattern
		matched, err := filepath.Match(matchPattern, filepath.Base(path))
		if err != nil {
			return fmt.Errorf("invalid match pattern: %w", err)
		}
		if !matched {
			return nil
		}

		// Apply sed replacement to the filename
		oldName := filepath.Base(path)
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

		op := output.RenameOperation{
			OldPath: relPath,
			NewPath: filepath.Join(filepath.Dir(relPath), newName),
		}

		// Check if target already exists
		if _, err := os.Stat(newPath); err == nil {
			if !forceOverwrite {
				op.Error = "target file already exists"
			}
		} else if !os.IsNotExist(err) {
			op.Error = fmt.Sprintf("cannot check target: %v", err)
		}

		// Perform rename if not dry run and no error
		if !isDryRun && op.Error == "" {
			if err := os.Rename(path, newPath); err != nil {
				op.Error = fmt.Sprintf("rename failed: %v", err)
			}
		}

		operations = append(operations, op)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return operations, exclusionsList, nil
}
