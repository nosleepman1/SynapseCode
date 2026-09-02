# Changelog

All notable changes to SynapseCode will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-09-02

### Added
* **Multi-Language AST Expansion**:
  * **Java**: Complete syntax support including Spring Boot annotations (`@RestController`, `@Service`, `@Autowired`, `@GetMapping`), Java records, enums, interfaces, and method signatures.
  * **PHP**: Full modern PHP & Laravel support including Eloquent models, controllers, PHP 8+ attributes (`#[Route]`, `#[HttpGet]`), traits, and enums.
  * **TypeScript/JavaScript**: Enhanced arrow functions (`const x = () => ...`), class methods, constructors, and enums.
  * **Python**: Decorators (`@dataclass`, `@property`, `@classmethod`), async methods, and class inheritance.
* **Graph Dependency Engine**:
  * Multi-pass dependency resolver constructing `EdgeCalls`, `EdgeImports`, and `EdgeImplements` relationships across files.
  * Graph pruning methods (`RemoveNode`, `RemoveFileNodes`) for live node and edge retraction.
* **Expanded MCP Tool Suite**:
  * `get_symbol_definition`: Complete signature, docstring, exact location, body snippet, and dependencies.
  * `get_impact_analysis` (Blast Radius): Reverse BFS dependency traversal computing direct callers, transitive dependents, affected files, impacted test suites, and risk level.
  * `get_file_outline`: Hierarchical outline of symbols and exports in a specific file.
  * Enriched `get_repo_map` (language & symbol distributions) and `get_symbol_callers` (hierarchical caller locations).
* **Performance & Persistence**:
  * Zero-dependency on-disk cache (`.synapse/cache.json`) with atomic file writes and SHA-256 / modtime verification.
  * Incremental indexing engine diffing `Cached`, `Added`, `Modified`, and `Deleted` files (sub-30ms execution).
  * Real-time event-driven file watcher (`fsnotify`) with 300ms debounce pipeline.
  * Adaptive Knapsack context budgeting with 1-hop graph neighbor skeleton enrichment.
* **CLI Additions**:
  * `synapse index` command with `--force` flag and execution metrics.
  * `--watch` flag for `synapse mcp` live mode.

## [1.0.0] - 2026-08-27

### Added
* Multi-language AST parsing support for Go, TypeScript, JavaScript, Python, and Rust.
* High-performance repository scanner with `.gitignore` and binary file filtering.
* In-memory directed multi-edge code graph (`CALLS`, `IMPORTS`, `DEFINES`, `IMPLEMENTS`, `EXTENDS`).
* Personalized PageRank ranking engine with deterministic score convergence.
* Knapsack-style token budgeting selector with Markdown context formatter.
* Complete Model Context Protocol (MCP) server implementation over JSON-RPC 2.0 stdio.
* Command-line interface with `synapse map`, `synapse context`, and `synapse mcp` commands.
* On-disk index cache persistence under `.synapse/index.json`.
* Multi-OS continuous integration and GoReleaser automated binary packaging.
