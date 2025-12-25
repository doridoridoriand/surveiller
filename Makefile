GO ?= go
BINARY ?= deadman-go
BIN_DIR ?= bin
PKG ?= ./...

.PHONY: all build test lint clean

all: build

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) .

test:
	$(GO) test $(PKG)

lint:
	@command -v golangci-lint >/dev/null 2>&1 && \
		golangci-lint run $(PKG) || \
		($(GO) vet $(PKG))

clean:
	@rm -rf $(BIN_DIR)
