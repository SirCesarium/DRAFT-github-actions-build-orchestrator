BINARY_NAME=refinery
VERSION=$$(git describe --tags --always --dirty)
COMMIT=$$(git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

.PHONY: all build test clean help

all: test build

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/refinery

run:
	go run ./cmd/refinery

test:
	go test -v ./...

fmt:
	go fmt ./...
	goimports -w .

clean:
	rm -rf bin/
	go clean

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'