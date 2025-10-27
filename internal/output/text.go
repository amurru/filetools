package output

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
)

// TextFormatter implements the OutputFormatter interface for plain text output
type TextFormatter struct{}

// FormatDuplicates formats duplicate results as plain text (preserves current behavior)
func (f *TextFormatter) FormatDuplicates(result *DuplicateResult, writer io.Writer) error {
	if !result.Found {
		fmt.Fprintln(writer, "No duplicate files found.")
		return nil
	}

	fmt.Fprintln(writer, "Duplicate files found:")
	for _, group := range result.Groups {
		// Sort files alphabetically
		files := make([]string, len(group.Files))
		copy(files, group.Files)
		sort.Strings(files)

		sizeStr := "unknown size"
		if group.Size >= 0 {
			sizeStr = fmt.Sprintf("%d bytes", group.Size)
		}

		// Display the first file as the "original"
		hashDisplay := group.Hash
		if len(hashDisplay) > 8 {
			hashDisplay = hashDisplay[:8] + "..."
		}

		fmt.Fprintf(writer, "- %s (size: %s, hash: %s)\n", filepath.Base(files[0]), sizeStr, hashDisplay)
		for _, file := range files {
			fmt.Fprintf(writer, "  - %s\n", file)
		}
		fmt.Fprintln(writer)
	}

	return nil
}
