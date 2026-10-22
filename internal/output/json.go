package output

import (
	"encoding/json"
	"github.com/amurru/filetools/internal/report"
	"io"
)

// JSONFormatter implements the OutputFormatter interface for JSON output
// JSONFormatter renders a report as JSON.
type JSONFormatter struct{}

// FormatDuplicates formats duplicate results as JSON
func (f *JSONFormatter) FormatDuplicates(result *report.DuplicateResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatDirStat formats directory statistics as JSON
func (f *JSONFormatter) FormatDirStat(result *report.DirStatResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatRename formats rename results as JSON
func (f *JSONFormatter) FormatRename(result *report.RenameResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatOrganize formats organize results as JSON
func (f *JSONFormatter) FormatOrganize(result *report.OrganizeResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// FormatSizeReport formats a size analysis as JSON
func (f *JSONFormatter) FormatSizeReport(result *report.SizeReportResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
