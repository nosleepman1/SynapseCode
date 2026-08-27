.PHONY: all build test bench lint clean run-mcp

BINARY_NAME=synapse
BUILD_DIR=bin

all: lint test build

build:
	@echo "==> Building $(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/synapse

test:
	@echo "==> Running unit tests with race detector..."
	@go test -v -race ./...

bench:
	@echo "==> Running benchmarks..."
	@go test -bench=. -benchmem ./...

lint:
	@echo "==> Running golangci-lint..."
	@golangci-lint run ./...

clean:
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

run-mcp: build
	@echo "==> Running Synapse MCP server on $(PATH)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) mcp --path $(PATH)
