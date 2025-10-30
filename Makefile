.PHONY: build test clean run

MODULE := amurru/filetools
BINARY := filetools
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "none")
COMMIT_DATE := $(shell git log --date=iso8601-strict -1 --pretty=%ct 2>/dev/null || echo "unknown")
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | cut -c2- || echo "dev-$(COMMIT)")
TREE_STATE := $(shell if git diff --quiet 2>/dev/null; then echo "clean"; else echo "dirty"; fi)

build:
	go build -ldflags "-X '$(MODULE)/cmd.version=$(VERSION)' -X '$(MODULE)/cmd.commit=$(COMMIT)' -X '$(MODULE)/cmd.commitDate=$(COMMIT_DATE)' -X '$(MODULE)/cmd.treeState=$(TREE_STATE)'" -o bin/$(BINARY) .

test:
	go test ./...

clean:
	rm -f bin/$(BINARY)

run: build
	./bin/$(BINARY)
