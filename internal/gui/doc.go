// Package gui implements the graphical interface of filetools.
//
// The interface is a second front end over the analysis engines in
// internal/scan. It shows the same tools as the command line interface:
// duplicate search, directory statistics, size analysis, renaming and
// organizing.
//
// The package is guarded by the "gui" build tag because Gio needs cgo and
// the platform windowing libraries on Linux. A build without the tag never
// compiles this package and never pulls in Gio, so the command line tool
// stays pure Go.
//
// Build with the tag:
//
//	go build -tags gui ./...
//
// On Linux this additionally requires the X11, Wayland and EGL development
// packages. See the installation instructions of Gio for the exact list.
package gui
