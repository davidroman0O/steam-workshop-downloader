# Workshop - Steam Workshop Downloader
# Makefile for easy building and development

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Build flags
LDFLAGS := -ldflags="-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.buildTime=$(BUILD_TIME)' -s -w"

# Default target
.PHONY: build
build:
	go build $(LDFLAGS) -o workshop .

# Build for all platforms (like GitHub Actions)
.PHONY: build-all
build-all:
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o workshop-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o workshop-linux-arm64 .
	GOOS=linux GOARCH=386 go build $(LDFLAGS) -o workshop-linux-386 .
	GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o workshop-linux-armv7 .
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o workshop-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o workshop-darwin-arm64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o workshop-windows-amd64.exe .
	GOOS=windows GOARCH=386 go build $(LDFLAGS) -o workshop-windows-386.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o workshop-windows-arm64.exe .

# Clean build artifacts
.PHONY: clean
clean:
	rm -f workshop workshop-*

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	go test -v ./...

# Development build (faster, no optimizations)
.PHONY: dev
dev:
	go build -o workshop .

# Show help
.PHONY: help
help:
	@echo "Workshop - Steam Workshop Downloader"
	@echo ""
	@echo "Available targets:"
	@echo "  build      - Build for current platform with version info"
	@echo "  build-all  - Build for all platforms"
	@echo "  dev        - Fast development build"
	@echo "  clean      - Remove build artifacts"
	@echo "  deps       - Install/update dependencies"
	@echo "  test       - Run tests"
	@echo "  help       - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION    - Override version (default: git describe)"

# Default target when just running 'make'
.DEFAULT_GOAL := build 