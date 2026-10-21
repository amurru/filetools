.PHONY: build build-gui test test-race vet fmt clean run

MODULE := amurru/filetools
BINARY := filetools
GUI_BINARY := filetools-gui
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "none")
COMMIT_DATE := $(shell git log --date=iso8601-strict -1 --pretty=%ct 2>/dev/null || echo "unknown")
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | cut -c2- || echo "dev-$(COMMIT)")
TREE_STATE := $(shell if git diff --quiet 2>/dev/null; then echo "clean"; else echo "dirty"; fi)

LDFLAGS := -X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.CommitDate=$(COMMIT_DATE)' \
	-X '$(MODULE)/internal/version.TreeState=$(TREE_STATE)'

# build produces the pure Go command line tool. It has no cgo dependency and
# cross compiles to any target.
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

# build-gui produces the same tool with the graphical interface enabled. On
# Linux it needs cgo and the X11, Wayland and EGL development packages.
build-gui:
	go build -tags gui -ldflags "$(LDFLAGS)" -o bin/$(GUI_BINARY) .

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

clean:
	rm -f bin/$(BINARY) bin/$(GUI_BINARY)

run: build
	./bin/$(BINARY)
