# Changelog

All notable changes to **SynapseCode** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-08-27

### Added
- Multi-language AST parsing support for **Go**, **TypeScript/JavaScript**, **Python**, and **Rust**.
- High-speed repository file discovery with intelligent `.gitignore` and binary filtering.
- In-memory directed multi-graph representation of files, functions, classes, interfaces, and methods.
- Personalized PageRank (PPR) ranking engine with custom damping and power iteration.
- Exact Knapsack token budgeting pruner to guarantee strict context limits.
- Full Model Context Protocol (MCP) server implementation over JSON-RPC 2.0 stdio.
- Command-line interface with `synapse map`, `synapse context`, and `synapse mcp`.
- Local on-disk index persistence in `.synapse/index.json`.
