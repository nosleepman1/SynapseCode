# Changelog

All notable changes to SynapseCode will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
