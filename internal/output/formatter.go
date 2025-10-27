package output

import (
	"io"
)

// Flag represents a command-line flag with name and value
type Flag struct {
	Name  string `json:"name" xml:"name"`
	Value string `json:"value" xml:",chardata"`
}

// Metadata contains information about the tool execution
type Metadata struct {
	ToolName    string `json:"tool_name" xml:"toolName"`
	SubCommand  string `json:"sub_command" xml:"subCommand"`
	Flags       []Flag `json:"flags" xml:"flags>flag"`
	Version     string `json:"version" xml:"version"`
	GeneratedAt string `json:"generated_at" xml:"generatedAt"`
}

// DuplicateGroup represents a group of duplicate files with the same hash
type DuplicateGroup struct {
	Hash     string   `json:"hash" xml:"hash"`
	HashType string   `json:"hash_type" xml:"hashType"`
	Size     int64    `json:"size" xml:"size"`
	Files    []string `json:"files" xml:"files"`
}

// DuplicateResult represents the complete result of a duplicate file search
type DuplicateResult struct {
	Metadata *Metadata        `json:"metadata" xml:"metadata"`
	Groups   []DuplicateGroup `json:"groups" xml:"groups"`
	Found    bool             `json:"found" xml:"found"`
}

// OutputFormatter defines the interface for different output formats
type OutputFormatter interface {
	FormatDuplicates(result *DuplicateResult, writer io.Writer) error
}

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatXML  OutputFormat = "xml"
	FormatHTML OutputFormat = "html"
)

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format OutputFormat) OutputFormatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatXML:
		return &XMLFormatter{}
	case FormatHTML:
		return &HTMLFormatter{}
	case FormatText:
		fallthrough
	default:
		return &TextFormatter{}
	}
}
