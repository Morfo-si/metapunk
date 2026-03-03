BINARY   := metapunk
MODULE   := github.com/morfo-si/metapunk
CMD      := .
GO       := go
GOFLAGS  :=

.PHONY: all build run test test-verbose test-cover lint fmt vet tidy clean install hooks

## all: build the binary (default target)
all: build

## build: compile the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY) $(CMD)

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

## clean: remove build artefacts
clean:
	$(GO) clean
	rm -f $(BINARY) coverage.out

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
