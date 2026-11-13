# Contributor Guide

Thank you for contributing to **SynapseCode**. This document outlines the engineering guidelines, testing protocols, and pull request standards for maintainers and contributors.

---

## 1. Development Prerequisites

* **Go**: Version 1.22 or higher
* **Git**
* **golangci-lint**: Version 1.58 or higher
* **Make**

To initialize the development environment:
```bash
git clone https://github.com/nosleepman1/SynapseCode.git
cd SynapseCode
go mod download
```

---

## 2. Common Build & Test Targets

The `Makefile` defines standard engineering commands:

```bash
# Compile binary locally
make build

# Run unit tests with race detection
make test

# Run benchmarks
make bench

# Run linter checks
make lint
```

---

## 3. Adding Support for a New Language

SynapseCode is designed to be easily extensible. To add a new programming language (e.g., C# or Java):

1. Create a new package under `internal/ast/<language>/`.
2. Implement the `ast.Parser` interface:
   ```go
   type Parser interface {
       Language() model.Language
       Extensions() []string
       Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error)
   }
   ```
3. Register your parser instance in `internal/mcp/server.go` and `internal/cli/root.go`:
   ```go
   reg.Register(yourlanguage.NewParser())
   ```
4. Add comprehensive unit tests in `internal/ast/<language>/parser_test.go`.

---

## 4. Engineering Standards

* **Error Wrapping**: Wrap contextual errors using `%w` (`fmt.Errorf("failed to parse file: %w", err)`).
* **Concurrency Safety**: Synchronize shared state using `sync.RWMutex` or channel-based communication.
* **Deterministic Behavior**: Ensure stable sorting algorithms are used on graph node lists.
* **Memory Optimization**: Avoid unnecessary heap allocations in hot loop indexers.

---

## 5. Pull Request Checklist

Before opening a pull request, ensure:
- [ ] `go test -race ./...` passes without errors.
- [ ] `golangci-lint run ./...` reports zero warnings.
- [ ] New functionality is covered by unit tests.
- [ ] Documentation is updated accordingly.
