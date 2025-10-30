package output

import (
	"encoding/json"
	"io"
)

// JSONFormatter implements the OutputFormatter interface for JSON output
type JSONFormatter struct{}

// FormatDuplicates formats duplicate results as JSON
func (f *JSONFormatter) FormatDuplicates(result *DuplicateResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatDirStat formats directory statistics as JSON
func (f *JSONFormatter) FormatDirStat(result *DirStatResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatRename formats rename results as JSON
func (f *JSONFormatter) FormatRename(result *RenameResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
