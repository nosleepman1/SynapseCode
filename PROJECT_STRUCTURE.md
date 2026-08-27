# Project Structure & Repository Layout: SynapseCode

## 1. Directory Tree

```
synapse-code/
├── .github/
│   └── workflows/
│       ├── ci.yml                     # Multi-OS test & build matrix
│       ├── release.yml                # Automated release with GoReleaser
│       └── security.yml               # Vulnerability scanning with govulncheck
├── cmd/
│   └── synapse/
│       └── main.go                    # CLI & MCP server entrypoint
├── pkg/
│   └── model/                         # Public domain types (Symbol, Location, Graph, Context)
│       ├── symbol.go
│       ├── location.go
│       ├── graph.go
│       └── context.go
├── internal/
│   ├── ast/                           # AST extraction & registry
│   │   ├── parser.go                  # LanguageParser interface
│   │   ├── language.go                # Language detection
│   │   ├── registry.go                # Thread-safe parser registry
│   │   ├── golang/                    # Go parser (go/parser)
│   │   ├── typescript/                # TypeScript / JavaScript parser
│   │   ├── python/                    # Python parser
│   │   └── rust/                      # Rust parser
│   ├── discovery/                     # Repository file scanner & ignore filter
│   │   ├── scanner.go
│   │   └── ignore.go
│   ├── graph/                         # In-memory code graph
│   │   ├── graph.go                   # Bidirectional adjacency lists
│   │   ├── pagerank.go                # Personalized PageRank implementation
│   │   └── traversal.go               # k-hop graph exploration
│   ├── search/                        # Lexical query engine
│   │   ├── tokenizer.go               # Identifier tokenizer
│   │   └── scorer.go                  # BM25-style scorer
│   ├── context/                       # Token budgeting and selection
│   │   ├── tokenizer.go               # BPE token estimator
│   │   ├── formatter.go               # LLM Markdown formatter
│   │   └── builder.go                 # Knapsack context selector
│   ├── storage/                       # On-disk cache & index persistence
│   │   ├── store.go
│   │   └── metadata.go
│   ├── indexer/                       # Worker pool concurrent pipeline
│   │   └── indexer.go
│   ├── mcp/                           # Model Context Protocol server
│   │   └── server.go
│   ├── cli/                           # Command Line Interface
│   │   └── root.go
│   ├── config/                        # Configuration loader
│   │   └── config.go
│   └── observability/                 # Structured logging
│       └── logging.go
├── docs/                              # Technical documentation
├── assets/                            # Architecture diagrams & media
├── .gitignore
├── .golangci.yml                      # Linter configuration
├── .goreleaser.yaml                   # Build & packaging configuration
├── Makefile
├── go.mod
├── go.sum
├── README.md
├── ARCHITECTURE.md
├── PROJECT_SPEC.md
├── PROMPTS.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
└── LICENSE
```

---

## 2. Package Responsibilities

* **`pkg/model`**: Defines pure domain models without external dependencies.
* **`internal/ast`**: Encapsulates language parsers behind an extensible interface.
* **`internal/discovery`**: Traverses files while enforcing ignore rules (`.gitignore`, binary files).
* **`internal/graph`**: Manages the graph topology and centrality scoring.
* **`internal/context`**: Assembles the final context pack within token budget constraints.
* **`internal/mcp`**: Implements JSON-RPC 2.0 communication over standard I/O.
* **`internal/storage`**: Persists and restores indexed graph states from `.synapse/index.json`.
