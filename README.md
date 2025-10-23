# File Tools

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A collection of command-line tools for efficient file management and analysis, built with Go.

## Features

### Current Tools

- **dupfind**: Find duplicate files in a directory tree by comparing file hashes. Efficiently identifies identical files regardless of filename or location.

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
```

The binary will be created as `filetools`.

### Install with Go

```bash
go install github.com/amurru/filetools@latest
```

## Usage

### dupfind

Find duplicate files in a directory:

```bash
filetools dupfind /path/to/directory
```

If no directory is specified, it uses the current directory:

```bash
filetools dupfind
```

Example output:

```
Duplicate files found:
- file1.txt (size: 1024 bytes)
  - /path/to/dir1/file1.txt
  - /path/to/dir2/file1.txt
- file2.txt (size: 2048 bytes)
  - /path/to/dir3/file2.txt
  - /path/to/dir4/file2.txt
```

### Version

Check the version:

```bash
filetools version
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
