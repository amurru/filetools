package output

import (
	"encoding/xml"
	"io"
)

// XMLFormatter implements the OutputFormatter interface for XML output
type XMLFormatter struct{}

// FormatDuplicates formats duplicate results as XML
func (f *XMLFormatter) FormatDuplicates(result *DuplicateResult, writer io.Writer) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}

	// Encode the result
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Write a newline at the end
	_, err := writer.Write([]byte("\n"))
	return err
}
