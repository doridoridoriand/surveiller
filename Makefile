GO ?= go
BINARY ?= surveiller
BIN_DIR ?= bin
PKG ?= ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS ?= -s -w -X main.version=$(VERSION)
PACKAGING_SCRIPT_DIR ?= scripts/packaging
PACKAGING_MANIFEST_SCRIPT ?= $(PACKAGING_SCRIPT_DIR)/generate_release_manifest.sh
PACKAGING_LINUX_SCRIPT ?= $(PACKAGING_SCRIPT_DIR)/build_linux_packages.sh
PACKAGING_HOMEBREW_SCRIPT ?= $(PACKAGING_SCRIPT_DIR)/render_homebrew_formula.sh
PACKAGING_CHOCO_SCRIPT ?= $(PACKAGING_SCRIPT_DIR)/build_choco_package.ps1
PACKAGING_SMOKE_SCRIPT ?= $(PACKAGING_SCRIPT_DIR)/ci/smoke_us1.sh

.PHONY: all build test test-prop test-all lint clean clean-build fmt vet install \
	package-help package-manifest package-linux package-homebrew package-choco package-build package-smoke

all: build

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

test:
	$(GO) test -v $(PKG)

test-prop:
	$(GO) test -v -tags=property ./internal/ping

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
	go clean -cache -testcache -modcache

clean-build: clean build

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

# Package build helpers (Phase 1 scaffold)
package-help:
	@echo "Package helper targets:"
	@echo "  make package-manifest  # Generate release manifest (Phase 2)"
	@echo "  make package-linux     # Build DEB/RPM packages (US1/T012)"
	@echo "  make package-homebrew  # Render Homebrew formula (US1/T013)"
	@echo "  make package-choco     # Build Chocolatey package payload (US1/T014)"
	@echo "  make package-build     # Run all package helper steps"

package-manifest:
	@if [ -x "$(PACKAGING_MANIFEST_SCRIPT)" ]; then \
		"$(PACKAGING_MANIFEST_SCRIPT)"; \
	else \
		echo "Skipping: $(PACKAGING_MANIFEST_SCRIPT) is not available yet (Phase 2/T006)."; \
	fi

package-linux:
	@if [ -x "$(PACKAGING_LINUX_SCRIPT)" ]; then \
		"$(PACKAGING_LINUX_SCRIPT)"; \
	else \
		echo "Skipping: $(PACKAGING_LINUX_SCRIPT) is not available yet (US1/T012)."; \
	fi

package-homebrew:
	@if [ -x "$(PACKAGING_HOMEBREW_SCRIPT)" ]; then \
		"$(PACKAGING_HOMEBREW_SCRIPT)"; \
	else \
		echo "Skipping: $(PACKAGING_HOMEBREW_SCRIPT) is not available yet (US1/T013)."; \
	fi

package-choco:
	@if [ -f "$(PACKAGING_CHOCO_SCRIPT)" ]; then \
		if command -v pwsh >/dev/null 2>&1; then \
			pwsh -File "$(PACKAGING_CHOCO_SCRIPT)"; \
		else \
			echo "Skipping: pwsh is not installed; cannot run $(PACKAGING_CHOCO_SCRIPT)."; \
		fi \
	else \
		echo "Skipping: $(PACKAGING_CHOCO_SCRIPT) is not available yet (US1/T014)."; \
	fi

package-build: package-manifest package-linux package-homebrew package-choco

package-smoke:
	@if [ -x "$(PACKAGING_SMOKE_SCRIPT)" ]; then \
		"$(PACKAGING_SMOKE_SCRIPT)"; \
	else \
		echo "Skipping: $(PACKAGING_SMOKE_SCRIPT) is not available."; \
	fi

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
