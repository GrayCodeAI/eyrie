NAME := eyrie
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: all build test lint fmt vet security bench clean help

all: lint test build

build: ## Build binary
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(NAME) ./cmd/$(NAME)

test: ## Run tests with race detector
	go test ./... -race -count=1 -timeout=120s

test-coverage: ## Run tests with coverage
	go test ./... -race -coverprofile=coverage.out -covermode=atomic -timeout=120s
	go tool cover -func=coverage.out | grep "^total:"

lint: ## Run linter
	golangci-lint run ./... --timeout=5m

fmt: ## Format code
	gofumpt -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

security: ## Run security scanner
	govulncheck ./...

bench: ## Run benchmarks
	go test ./... -bench=. -benchmem -count=3 -timeout=300s

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
