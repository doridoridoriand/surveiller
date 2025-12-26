GO ?= go
BINARY ?= deadman-go
BIN_DIR ?= bin
PKG ?= ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS ?= -s -w -X main.version=$(VERSION)

.PHONY: all build test test-prop test-all lint clean fmt vet install

all: build

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

test:
	$(GO) test -v $(PKG)

test-prop:
	$(GO) test -v -tags=property $(PKG)

test-all: test test-prop

lint:
	@command -v golangci-lint >/dev/null 2>&1 && \
		golangci-lint run $(PKG) || \
		($(GO) vet $(PKG))

fmt:
	$(GO) fmt $(PKG)

vet:
	$(GO) vet $(PKG)

install: build
	@cp $(BIN_DIR)/$(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || \
		cp $(BIN_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	@rm -rf $(BIN_DIR)

# Development helpers
dev-deps:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Release build for multiple platforms
release:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe .
	@cd dist && sha256sum * > checksums.txt

# Release management
release-check:
	@echo "Current version: $(VERSION)"
	@echo "Git status:"
	@git status --short
	@echo "Recent commits:"
	@git log --oneline -5

release-tag:
	@if [ -z "$(TAG)" ]; then \
		echo "Usage: make release-tag TAG=v0.0.1"; \
		exit 1; \
	fi
	@./scripts/release.sh $(TAG)

# Quick release (for development)
release-dev:
	@$(MAKE) release-tag TAG=v0.0.1
