.PHONY: build test clean run

MODULE := amurru/filetools
BINARY := filetools
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
VERSION := $(shell git describe --exact-match --tags 2>/dev/null || echo "dev-$(COMMIT)")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	go build -ldflags "-X '$(MODULE)/cmd.version=$(VERSION)' -X '$(MODULE)/cmd.date=$(DATE)'" -o bin/$(BINARY) .

test:
	go test ./...

clean:
	rm -f bin/$(BINARY)

run: build
	./bin/$(BINARY)
