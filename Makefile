GOLANGCI_LINT_VERSION=1.56.2

GO_BUILD_VERSION_LDFLAGS=-X main.Version=$(shell git rev-parse --short HEAD)

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="$(GO_BUILD_VERSION_LDFLAGS)" -o dist/aws-checker .

.PHONY: lint
lint:
	docker run --rm -v $(shell pwd):/app -v ~/.cache/golangci-lint/v$(GOLANGCI_LINT_VERSION):/root/.cache -w /app golangci/golangci-lint:v$(GOLANGCI_LINT_VERSION) golangci-lint run -v --timeout=5m

.PHONY: test
test:
	go test -timeout 6m -v ./...

.PHONY: goreleaser-snapshot
goreleaser-snapshot:
	curl -sfL https://goreleaser.com/static/run | REGISTRY=examplecom bash -s -- --clean --snapshot
