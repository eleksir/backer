#!/usr/bin/env gmake -f

.PHONY: all build clean test upgrade help

BINARY=backer
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"
BUILDOPTS=-a -gcflags=all=-l -trimpath -buildvcs=false

# Binary name with .exe for Windows
ifdef GOOS
  ifeq ($(GOOS),windows)
    BINARY=backer.exe
  endif
else ifeq ($(OS),Windows_NT)
  BINARY=backer.exe
endif

all: clean build

build:
	CGO_ENABLED=0 go build $(BUILDOPTS) $(LDFLAGS) -o $(BINARY) ./cmd/backer

clean:
	rm -f $(BINARY)

upgrade:
	go get -u ./...
	go mod tidy
	go mod vendor

test:
	go test ./...

help:
	@echo "Backer build system"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build     Build the binary"
	@echo "  clean     Remove built binary"
	@echo "  test      Run all tests"
	@echo "  upgrade   Update dependencies"
	@echo "  help      Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION   Set version (default: git describe)"
	@echo ""
	@echo "Example:"
	@echo "  make build"

# vim: set ft=make noet ai ts=4 sw=4 sts=4:
