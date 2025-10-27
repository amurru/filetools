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

// FileInfo represents information about a single file
type FileInfo struct {
	Name string `json:"name" xml:"name"`
	Size int64  `json:"size" xml:"size"`
	Path string `json:"path" xml:"path"`
}

// FileType represents statistics for files of a specific type/extension
type FileType struct {
	Extension  string  `json:"extension" xml:"extension"`
	Count      int     `json:"count" xml:"count"`
	TotalSize  int64   `json:"total_size" xml:"totalSize"`
	Percentage float64 `json:"percentage" xml:"percentage"`
}

// DirectoryInfo represents statistics for a subdirectory
type DirectoryInfo struct {
	Path       string  `json:"path" xml:"path"`
	FileCount  int     `json:"file_count" xml:"fileCount"`
	TotalSize  int64   `json:"total_size" xml:"totalSize"`
	Percentage float64 `json:"percentage" xml:"percentage"`
}

// DirStatResult represents the complete result of a directory statistics analysis
type DirStatResult struct {
	Metadata    *Metadata       `json:"metadata" xml:"metadata"`
	TotalFiles  int             `json:"total_files" xml:"totalFiles"`
	TotalSize   int64           `json:"total_size" xml:"totalSize"`
	LargestFile *FileInfo       `json:"largest_file" xml:"largestFile"`
	FileTypes   []FileType      `json:"file_types" xml:"fileTypes"`
	Directories []DirectoryInfo `json:"directories" xml:"directories"`
}

// OutputFormatter defines the interface for different output formats
type OutputFormatter interface {
	FormatDuplicates(result *DuplicateResult, writer io.Writer) error
	FormatDirStat(result *DirStatResult, writer io.Writer) error
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
