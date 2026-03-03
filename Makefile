BINARY     := apt_exporter
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO         := go
GOLANGCI   := golangci-lint

.PHONY: build test test-integration run clean docker-build lint fmt vet

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/apt_exporter

test:
	$(GO) test -v -race -count=1 ./internal/...

test-integration: build-static
	$(GO) test -v -tags=integration -count=1 -timeout=300s ./test/...

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -s -w" -o bin/$(BINARY) ./cmd/apt_exporter

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin/

docker-build:
	docker build -t apt-exporter:$(VERSION) .

lint:
	$(GOLANGCI) run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...
