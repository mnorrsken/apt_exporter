BINARY     := apt_exporter
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO         := go
GOLANGCI   := golangci-lint

.PHONY: build test test-integration run clean docker-build deb deb-amd64 deb-arm64 lint fmt vet

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
	rm -rf bin/ dist/

docker-build:
	docker build -t apt-exporter:$(VERSION) .

deb: deb-amd64 deb-arm64

deb-amd64:
	docker buildx build -f Dockerfile.deb \
		--platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		--target export \
		--output type=local,dest=./dist \
		.

deb-arm64:
	docker buildx build -f Dockerfile.deb \
		--platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--target export \
		--output type=local,dest=./dist \
		.

lint:
	$(GOLANGCI) run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...
