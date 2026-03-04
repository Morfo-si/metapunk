BINARY   := metapunk
MODULE   := github.com/Morfo-si/metapunk
CMD      := .
GO       := go
GOFLAGS  :=
VERSION  := $(shell cat VERSION)

.PHONY: all build run test test-verbose test-cover test-cover-html fmt vet lint tidy install hooks tag snapshot clean help

## all: build the binary (default target)
all: build

## build: compile the binary (injects VERSION into the binary)
build:
	$(GO) build $(GOFLAGS) -ldflags "-X main.version=$(VERSION)" -o $(BINARY) $(CMD)

## run: build and run the binary
run: build
	./$(BINARY)

## test: run all tests
test:
	$(GO) test $(GOFLAGS) ./...

## test-verbose: run all tests with verbose output
test-verbose:
	$(GO) test $(GOFLAGS) -v ./...

## test-cover: run tests and show a coverage report
test-cover:
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

## test-cover-html: open an HTML coverage report in the browser
test-cover-html: test-cover
	$(GO) tool cover -html=coverage.out

## fmt: format all Go source files
fmt:
	$(GO) fmt ./...

## vet: run go vet on all packages
vet:
	$(GO) vet ./...

## lint: fmt + vet (requires no extra tools)
lint: fmt vet

## tidy: tidy and verify the module dependencies
tidy:
	$(GO) mod tidy
	$(GO) mod verify

## install: install the binary to GOPATH/bin
install:
	$(GO) install $(GOFLAGS) $(MODULE)

## hooks: install git hooks from .githooks/ into .git/hooks/
hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks installed (core.hooksPath = .githooks)"

## tag: create and push a signed git tag for the current VERSION (triggers the release workflow)
##      Edit the VERSION file first, commit it, then run: make tag
tag:
	@echo "Tagging v$(VERSION)…"
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Tag v$(VERSION) pushed — the release workflow will now build and publish the binaries."

## snapshot: build a local multi-platform snapshot with GoReleaser (no tag or publish)
##           Requires goreleaser to be installed: https://goreleaser.com/install/
snapshot:
	goreleaser release --snapshot --clean

## clean: remove build artefacts
clean:
	$(GO) clean
	rm -f $(BINARY) coverage.out
	rm -rf dist/

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
