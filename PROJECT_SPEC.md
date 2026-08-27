# Engineering Specification: SynapseCode

## 1. Executive Summary

SynapseCode is an open-source, high-performance code intelligence and context optimization engine written in Go. Its objective is to eliminate prompt context bloat and excessive API token consumption in AI coding workflows.

Rather than passing full files or unpruned directory structures to Large Language Models (LLMs), SynapseCode parses the codebase into an in-memory dependency graph, ranks candidate nodes using task-personalized PageRank, and selects the most relevant implementations along with direct interface skeletons under a strict token budget.

---

## 2. Core Value Propositions

| Dimension | Raw File Injection | SynapseCode Context Optimization |
| :--- | :--- | :--- |
| **Token Consumption** | 50,000 to 200,000 tokens per prompt | 2,000 to 8,000 tokens per prompt |
| **Financial Cost** | High linear cost per query | 75% to 90% reduction in API bills |
| **Latency (TTFT)** | 8 to 15 seconds | 1 to 2 seconds |
| **Attention Focus** | Model diluted by noise | Focused strictly on relevant targets |

---

## 3. Quantitative Targets

* **Token Reduction**: Greater than or equal to 75% per coding session.
* **Cold Indexing Latency**: Less than 1.5 seconds for 10,000 source files.
* **Incremental Cache Load**: Less than 50 milliseconds from serialized `.synapse/index.json`.
* **Memory Footprint**: Less than 80 MB resident memory for codebases with up to 1,000,000 lines of code.
* **Zero External Runtime Dependencies**: Single static binary with no required CGO toolchains or Python interpreters.

---

## 4. Scope and System Boundaries

### In Scope
1. **Multi-Language AST Extraction**: First-class support for Go, TypeScript/JavaScript, Python, and Rust.
2. **Directed Multi-Edge Graph**: Nodes representing files and symbols; edges representing `CALLS`, `IMPORTS`, `DEFINES`, `IMPLEMENTS`, `EXTENDS`, and `REFERENCES`.
3. **Graph Ranking**: Personalized PageRank with configurable damping factor and power iteration.
4. **Token Management**: Strict token counting and Knapsack-style selection.
5. **Model Context Protocol (MCP)**: Full JSON-RPC 2.0 stdio transport implementation.
6. **Local Persistence**: Incremental cache serialization under `.synapse/`.

### Out of Scope
* Not a full compiler or type checker.
* Not an interactive text editor.
* Not a proprietary cloud-hosted SaaS.

---

## 5. Architectural Quality Attributes

* **Determinism**: Given the same codebase and task query, the output must be identical. Stable sorting is enforced on equal scores.
* **Decoupling**: Domain models and graph structures do not depend on transport layers or CLI frameworks.
* **Extensibility**: Adding new language support requires implementing a single `LanguageParser` interface without modifying the graph engine.
