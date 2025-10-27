# File Tools

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A collection of command-line tools for efficient file management and analysis, built with Go.

## Features

### Current Tools

- **dupfind**: Find duplicate files in a directory tree by comparing file hashes. Efficiently identifies identical files regardless of filename or location.

### Key Features

- **Multiple Output Formats**: Support for text, JSON, XML, and HTML output formats
- **File Output**: Redirect output to files instead of stdout
- **Flexible Hashing**: Choose from MD5, SHA1, or SHA256 hash algorithms
- **Structured Data**: JSON/XML output provides machine-readable duplicate file information
- **Rich HTML Reports**: Generate professional HTML reports with styling and statistics

### Planned Tools

- File organizer
- Batch renamer
- File size analyzer

## Installation

### Prerequisites

- Go 1.24.5 or later

### Build from Source

```bash
git clone https://github.com/amurru/filetools.git
cd filetools
make build
make test  # Run tests
```

The binary will be created as `bin/filetools`.

### Install with Go

```bash
go install github.com/amurru/filetools@latest
```

### Development

```bash
make test      # Run all tests
make clean     # Clean build artifacts
make run       # Build and run the application
```

## Usage

### dupfind

Find duplicate files in a directory tree with flexible output options.

#### Basic Usage

Find duplicate files in a directory:

```bash
filetools dupfind /path/to/directory
```

If no directory is specified, it uses the current directory:

```bash
filetools dupfind
```

#### Output Formats

Choose from multiple output formats:

```bash
# Text output (default)
filetools dupfind /path/to/directory

# JSON output
filetools dupfind -o json /path/to/directory
filetools dupfind -j /path/to/directory

# XML output
filetools dupfind -o xml /path/to/directory
filetools dupfind -x /path/to/directory

# HTML output (generates a styled web page)
filetools dupfind -o html /path/to/directory
filetools dupfind -w /path/to/directory
```

#### File Output

Redirect output to a file instead of stdout:

```bash
# Save results to a file
filetools dupfind -f results.txt /path/to/directory
filetools dupfind -o json -f duplicates.json /path/to/directory
filetools dupfind -w -f report.html /path/to/directory
```

#### Hash Algorithms

Choose the hash algorithm for file comparison:

```bash
# Use different hash algorithms (default: md5)
filetools dupfind -H sha256 /path/to/directory
filetools dupfind -H sha1 /path/to/directory
filetools dupfind -H md5 /path/to/directory
```

#### Combined Usage

Combine multiple options:

```bash
# Generate JSON report with SHA256 hashes, save to file
filetools dupfind -H sha256 -o json -f report.json /path/to/directory

# Create HTML report with MD5 hashes
filetools dupfind -H md5 -w -f analysis.html /path/to/directory
```

#### Example Outputs

**Text Output (default):**
```
Duplicate files found:
- file1.txt (size: 1024 bytes, hash: a1b2c3d4...)
  - /path/to/dir1/file1.txt
  - /path/to/dir2/file1.txt
- file2.txt (size: 2048 bytes, hash: e5f6g7h8...)
  - /path/to/dir3/file2.txt
  - /path/to/dir4/file2.txt
```

**JSON Output:**
```json
{
  "groups": [
    {
      "hash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
      "size": 1024,
      "files": [
        "/path/to/dir1/file1.txt",
        "/path/to/dir2/file1.txt"
      ]
    }
  ],
  "found": true
}
```

**XML Output:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<DuplicateResult>
  <groups>
    <hash>a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6</hash>
    <size>1024</size>
    <files>/path/to/dir1/file1.txt</files>
    <files>/path/to/dir2/file1.txt</files>
  </groups>
  <found>true</found>
</DuplicateResult>
```

**HTML Output:**
Generates a complete HTML page with:
- Professional styling and layout
- Summary statistics
- Color-coded file badges (original/duplicate)
- Responsive design

### Version

Check the version and build information:

```bash
filetools version
```

Output:
```
version: dev-abc1234
date: 2025-10-27T08:55:40Z
```

## Project Structure

```
filetools/
├── cmd/                    # CLI commands
│   ├── dupfind.go         # Duplicate file finder command
│   ├── dupfind_test.go    # Tests for dupfind
│   ├── root.go            # Root command and global flags
│   └── version.go         # Version command
├── internal/
│   └── output/            # Output formatting module
│       ├── formatter.go   # Core interfaces and data structures
│       ├── json.go        # JSON formatter
│       ├── xml.go         # XML formatter
│       ├── html.go        # HTML formatter
│       ├── text.go        # Text formatter
│       └── formatter_test.go # Output tests
├── main.go                # Application entry point
├── go.mod                 # Go module definition
├── Makefile               # Build automation
└── README.md              # This file
```

## Architecture

The tool is built with a modular architecture:

- **CLI Layer**: Uses Cobra for command-line interface with persistent flags
- **Core Logic**: File hashing and duplicate detection algorithms
- **Output Layer**: Pluggable formatters for different output types
- **Data Flow**: Structured data flows from detection → formatting → output (stdout/file)

## Command Reference

### Global Flags

These flags work with all commands:

- `-o, --output string`: Output format (text, json, xml, html) (default "text")
- `-f, --file string`: Output file (default: stdout)
- `-j, --json`: Shortcut for `-o json`
- `-x, --xml`: Shortcut for `-o xml`
- `-w, --html`: Shortcut for `-o html`

### dupfind Flags

- `-H, --hash string`: Hash algorithm (md5, sha1, sha256) (default "md5")

### Examples

```bash
# View help
filetools --help
filetools dupfind --help

# Different output combinations
filetools dupfind -j -f results.json /path
filetools dupfind -o xml -f report.xml /path
filetools dupfind -w /path > report.html
```

## Development

### Testing

Run the test suite:

```bash
make test
```

Run specific tests:

```bash
go test -run TestCalculateHash ./cmd/
go test ./internal/output/
```

### Code Quality

The project follows Go best practices:

- Uses `gofmt` for consistent formatting
- Includes comprehensive unit tests
- Follows standard Go naming conventions
- Uses Cobra for CLI framework
- Modular architecture for maintainability

### Adding New Output Formats

To add a new output format:

1. Create a new formatter in `internal/output/`
2. Implement the `OutputFormatter` interface
3. Add the format to `NewFormatter()` function
4. Add corresponding flag if needed
5. Update tests and documentation

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Workflow

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Write tests for your changes
4. Ensure all tests pass (`make test`)
5. Update documentation if needed
6. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
7. Push to the branch (`git push origin feature/AmazingFeature`)
8. Open a Pull Request

### Guidelines

- Follow the existing code style and architecture
- Add tests for new functionality
- Update README.md for new features
- Ensure backward compatibility
- Use meaningful commit messages

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.