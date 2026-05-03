BINARY     := classifiler
GO         := go
GOFLAGS    :=
LINT       := golangci-lint

.PHONY: all build test lint clean

all: build

## build: compile the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY) .

## test: run all unit tests
test:
	$(GO) test ./...

## lint: run golangci-lint (install from https://golangci-lint.run)
lint:
	$(LINT) run ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
