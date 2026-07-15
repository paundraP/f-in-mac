BINARY := pfparse
GO := go
LDFLAGS := -s -w
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: all build test fuzz cross clean vet release install

all: vet test build

build:
	$(GO) build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/pfparse/

test:
	$(GO) test ./... -count=1

vet:
	$(GO) vet ./...

fuzz:
	$(GO) test -fuzz=. -fuzztime=10s ./pkg/prefetch/

cross: cross-darwin-amd64 cross-darwin-arm64 cross-linux-amd64

cross-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-darwin-amd64 ./cmd/pfparse/

cross-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-darwin-arm64 ./cmd/pfparse/

cross-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64 ./cmd/pfparse/

release:
	@goreleaser release --clean

release-snapshot:
	@goreleaser release --clean --snapshot

install:
	$(GO) install -ldflags="$(LDFLAGS)" ./cmd/pfparse/

clean:
	rm -rf bin/ dist/
