package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/ast/golang"
	"github.com/nosleepman1/synapse-code/internal/ast/java"
	"github.com/nosleepman1/synapse-code/internal/ast/php"
	"github.com/nosleepman1/synapse-code/internal/ast/python"
	"github.com/nosleepman1/synapse-code/internal/ast/rust"
	"github.com/nosleepman1/synapse-code/internal/ast/typescript"
	appcontext "github.com/nosleepman1/synapse-code/internal/context"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/internal/indexer"
	"github.com/nosleepman1/synapse-code/internal/storage"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// JSONRPCMessage represents a standard JSON-RPC 2.0 frame.
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

// Server handles Model Context Protocol (MCP) interactions over stdio.
type Server struct {
	repoPath       string
	graph          *graph.Graph
	contextBuilder *appcontext.Builder
	cache          *storage.FileCache
	indexer        *indexer.Indexer
	watcher        *discovery.FileWatcher
}

// NewServer initializes the MCP server and triggers codebase indexing using the persistent cache.
func NewServer(repoPath string) *Server {
	return NewServerWithOptions(repoPath, false)
}

// NewServerWithOptions creates the MCP server with optional live background file watching.
func NewServerWithOptions(repoPath string, enableWatcher bool) *Server {
	g := graph.NewGraph()
	reg := ast.NewRegistry()
	reg.Register(golang.NewParser())
	reg.Register(typescript.NewParser())
	reg.Register(python.NewParser())
	reg.Register(rust.NewParser())
	reg.Register(java.NewParser())
	reg.Register(php.NewParser())

	matcher := discovery.NewIgnoreMatcher(nil, 1024)
	scanner := discovery.NewScanner(matcher)
	idx := indexer.NewIndexer(scanner, reg)

	cache, _ := storage.LoadCache(repoPath)
	_, _ = idx.IndexIncremental(context.Background(), repoPath, g, cache)

	s := &Server{
		repoPath:       repoPath,
		graph:          g,
		contextBuilder: appcontext.NewBuilder(),
		cache:          cache,
		indexer:        idx,
	}

	if enableWatcher {
		s.startLiveWatcher(matcher)
	}

	return s
}

func (s *Server) startLiveWatcher(matcher *discovery.IgnoreMatcher) {
	watcher, err := discovery.NewFileWatcher(s.repoPath, matcher, 300*time.Millisecond, func(changedPaths []string) {
		_, _ = s.indexer.IndexIncremental(context.Background(), s.repoPath, s.graph, s.cache)
	})
	if err == nil {
		s.watcher = watcher
		go watcher.Start(context.Background())
	}
}

// Close releases any background resources such as the file watcher.
func (s *Server) Close() error {
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

// StartStdio starts reading JSON-RPC messages from stdin and replying on stdout.
func (s *Server) StartStdio() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("stdin read error: %w", err)
		}

		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var req JSONRPCMessage
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		resp := s.handleRequest(&req)
		if resp != nil {
			respBytes, err := json.Marshal(resp)
			if err == nil {
				os.Stdout.Write(respBytes)
				os.Stdout.WriteString("\n")
			}
		}
	}
}

func (s *Server) handleRequest(req *JSONRPCMessage) *JSONRPCMessage {
	switch req.Method {
	case "initialize":
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "synapse-code",
					"version": "1.0.0",
				},
			},
		}

	case "notifications/initialized":
		return nil // Notification, no response needed

	case "tools/list":
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.getToolsList(),
			},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: map[string]interface{}{
				"code":    -32601,
				"message": fmt.Sprintf("Method '%s' not found", req.Method),
			},
		}
	}
}

func (s *Server) getToolsList() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "get_repo_map",
			"description": "Returns a condensed architectural map and graph statistics of the codebase.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"budget_tokens": map[string]interface{}{
						"type":        "number",
						"description": "Max token budget for the map (default: 2000)",
					},
				},
			},
		},
		{
			"name":        "get_context_for_task",
			"description": "Extracts targeted code implementations and 1-hop AST signatures for a specific task under a token budget.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_description": map[string]interface{}{
						"type":        "string",
						"description": "The user task intent or query",
					},
					"budget_tokens": map[string]interface{}{
						"type":        "number",
						"description": "Max tokens allocated (default: 3500)",
					},
				},
				"required": []string{"task_description"},
			},
		},
		{
			"name":        "get_symbol_callers",
			"description": "Finds all callers and dependents of a function/method across the codebase.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol_name": map[string]interface{}{
						"type":        "string",
						"description": "Name or qualified name of the symbol to inspect",
					},
				},
				"required": []string{"symbol_name"},
			},
		},
		{
			"name":        "get_symbol_definition",
			"description": "Retrieves the full definition, signature, location, documentation, and body of a symbol.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol_name": map[string]interface{}{
						"type":        "string",
						"description": "Name or qualified name of the symbol",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Optional file path to disambiguate identical symbol names",
					},
				},
				"required": []string{"symbol_name"},
			},
		},
		{
			"name":        "get_impact_analysis",
			"description": "Calculates the blast radius of modifying a symbol (direct callers, transitive dependents, impacted files and test suites).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the symbol being modified or refactored",
					},
					"max_depth": map[string]interface{}{
						"type":        "number",
						"description": "Maximum reverse traversal depth for callers (default: 3)",
					},
				},
				"required": []string{"symbol_name"},
			},
		},
		{
			"name":        "get_file_outline",
			"description": "Returns a structured outline of all symbols, types, functions, and classes defined in a specific file.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative or absolute path of the file to inspect",
					},
				},
				"required": []string{"file_path"},
			},
		},
	}
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func (s *Server) handleToolCall(req *JSONRPCMessage) *JSONRPCMessage {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: map[string]interface{}{
				"code":    -32602,
				"message": "Invalid params",
			},
		}
	}

	var resultText string

	switch params.Name {
	case "get_repo_map":
		resultText = s.formatRepoMap()

	case "get_context_for_task":
		taskDesc, _ := params.Arguments["task_description"].(string)
		if strings.TrimSpace(taskDesc) == "" {
			return &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: map[string]interface{}{
					"code":    -32602,
					"message": "Missing required parameter 'task_description'",
				},
			}
		}
		budget := 3500
		if b, ok := params.Arguments["budget_tokens"].(float64); ok && b > 0 {
			budget = int(b)
		}

		pack, err := s.contextBuilder.BuildContextPack(context.Background(), s.graph, taskDesc, budget)
		if err != nil {
			resultText = fmt.Sprintf("Error extracting context: %v", err)
		} else {
			resultText = pack.FormattedText
		}

	case "get_symbol_callers":
		symbolName, _ := params.Arguments["symbol_name"].(string)
		if strings.TrimSpace(symbolName) == "" {
			return &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: map[string]interface{}{
					"code":    -32602,
					"message": "Missing required parameter 'symbol_name'",
				},
			}
		}
		resultText = s.findSymbolCallers(symbolName)

	case "get_symbol_definition":
		symbolName, _ := params.Arguments["symbol_name"].(string)
		if strings.TrimSpace(symbolName) == "" {
			return &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: map[string]interface{}{
					"code":    -32602,
					"message": "Missing required parameter 'symbol_name'",
				},
			}
		}
		filePath, _ := params.Arguments["file_path"].(string)
		resultText = s.findSymbolDefinition(symbolName, filePath)

	case "get_impact_analysis":
		symbolName, _ := params.Arguments["symbol_name"].(string)
		if strings.TrimSpace(symbolName) == "" {
			return &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: map[string]interface{}{
					"code":    -32602,
					"message": "Missing required parameter 'symbol_name'",
				},
			}
		}
		maxDepth := 3
		if d, ok := params.Arguments["max_depth"].(float64); ok && d > 0 {
			maxDepth = int(d)
		}
		resultText = s.calculateImpactAnalysis(symbolName, maxDepth)

	case "get_file_outline":
		filePath, _ := params.Arguments["file_path"].(string)
		if strings.TrimSpace(filePath) == "" {
			return &JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: map[string]interface{}{
					"code":    -32602,
					"message": "Missing required parameter 'file_path'",
				},
			}
		}
		resultText = s.getFileOutline(filePath)

	default:
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: map[string]interface{}{
				"code":    -32601,
				"message": fmt.Sprintf("Tool '%s' not recognized", params.Name),
			},
		}
	}

	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": resultText,
				},
			},
		},
	}
}

func (s *Server) formatRepoMap() string {
	summary := s.graph.Summary()
	var sb strings.Builder

	sb.WriteString("# SynapseCode Architectural Overview\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Indexed Files**: %d\n", summary.TotalFiles))
	sb.WriteString(fmt.Sprintf("- **Total Extracted Symbols**: %d\n", summary.TotalSymbols))
	sb.WriteString(fmt.Sprintf("- **Total Dependency Edges**: %d\n\n", summary.TotalEdges))

	// Language breakdown & Kind breakdown
	langCount := make(map[string]int)
	kindCount := make(map[string]int)

	for _, node := range s.graph.AllNodes() {
		if node.Symbol != nil {
			langCount[string(node.Symbol.Language)]++
			kindCount[string(node.Symbol.Kind)]++
		}
	}

	if len(langCount) > 0 {
		sb.WriteString("## Language Distribution\n")
		for lang, count := range langCount {
			sb.WriteString(fmt.Sprintf("- **%s**: %d symbols\n", lang, count))
		}
		sb.WriteString("\n")
	}

	if len(kindCount) > 0 {
		sb.WriteString("## Symbol Categories\n")
		for kind, count := range kindCount {
			sb.WriteString(fmt.Sprintf("- **%s**: %d\n", kind, count))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Server) findSymbolCallers(symbolName string) string {
	var sb strings.Builder
	found := false

	for _, node := range s.graph.AllNodes() {
		if node.Symbol != nil && (node.Symbol.Name == symbolName || node.Symbol.QualifiedName == symbolName) {
			found = true
			incoming := s.graph.GetIncoming(node.ID)

			var callerEdges []string
			for _, edge := range incoming {
				if edge.Type == "CALLS" || edge.Type == "REFERENCES" {
					callerEdges = append(callerEdges, fmt.Sprintf("- Called by `%s` (Edge: %s)", edge.Source, edge.Type))
				}
			}

			sb.WriteString(fmt.Sprintf("### Symbol `%s` in `%s`\n", node.Symbol.Name, node.FilePath))
			if len(callerEdges) == 0 {
				sb.WriteString("- *No incoming callers found in index.*\n\n")
			} else {
				for _, ce := range callerEdges {
					sb.WriteString(ce + "\n")
				}
				sb.WriteString("\n")
			}
		}
	}

	if !found {
		return fmt.Sprintf("Symbol '%s' was not found in the codebase index.", symbolName)
	}

	return sb.String()
}

func (s *Server) findSymbolDefinition(symbolName string, filePath string) string {
	var matches []*model.Node

	for _, node := range s.graph.AllNodes() {
		if node.Symbol != nil && (node.Symbol.Name == symbolName || node.Symbol.QualifiedName == symbolName) {
			if filePath == "" || strings.HasSuffix(node.FilePath, filePath) {
				matches = append(matches, node)
			}
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("Symbol '%s' not found in codebase.", symbolName)
	}

	var sb strings.Builder
	for _, node := range matches {
		sym := node.Symbol
		sb.WriteString(fmt.Sprintf("## Symbol: `%s`\n", sym.Name))
		sb.WriteString(fmt.Sprintf("- **File**: `%s` (Lines %d-%d)\n", node.FilePath, sym.Location.StartLine, sym.Location.EndLine))
		sb.WriteString(fmt.Sprintf("- **Kind**: %s\n", sym.Kind))
		sb.WriteString(fmt.Sprintf("- **Language**: %s\n", sym.Language))
		sb.WriteString(fmt.Sprintf("- **Exported**: %v\n", sym.Exported))

		if sym.Documentation != "" {
			sb.WriteString(fmt.Sprintf("\n### Documentation\n```\n%s\n```\n", sym.Documentation))
		}

		sb.WriteString(fmt.Sprintf("\n### Signature\n```%s\n%s\n```\n", sym.Language, sym.Signature))

		if sym.Body != "" {
			sb.WriteString(fmt.Sprintf("\n### Implementation\n```%s\n%s\n```\n", sym.Language, sym.Body))
		}

		if len(sym.Calls) > 0 {
			sb.WriteString("\n### Outgoing Calls\n")
			for _, c := range sym.Calls {
				sb.WriteString(fmt.Sprintf("- `%s`\n", c))
			}
		}
		sb.WriteString("\n---\n")
	}

	return sb.String()
}

func (s *Server) calculateImpactAnalysis(symbolName string, maxDepth int) string {
	var targetNodes []*model.Node
	for _, node := range s.graph.AllNodes() {
		if node.Symbol != nil && (node.Symbol.Name == symbolName || node.Symbol.QualifiedName == symbolName) {
			targetNodes = append(targetNodes, node)
		}
	}

	if len(targetNodes) == 0 {
		return fmt.Sprintf("Symbol '%s' was not found in the codebase index.", symbolName)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Impact & Blast Radius Analysis for `%s`\n\n", symbolName))

	for _, target := range targetNodes {
		sb.WriteString(fmt.Sprintf("## Target: `%s` (Defined in `%s`)\n\n", target.Symbol.Name, target.FilePath))

		// Reverse BFS traversal along incoming CALLS & REFERENCES edges
		visited := make(map[model.NodeID]int) // nodeID -> hop distance
		visited[target.ID] = 0
		queue := []model.NodeID{target.ID}

		impactedFiles := make(map[string]bool)
		impactedTestFiles := make(map[string]bool)
		directCallers := make([]string, 0)
		transitiveCallers := make([]string, 0)

		for len(queue) > 0 {
			currID := queue[0]
			queue = queue[1:]
			currDist := visited[currID]

			if currDist >= maxDepth {
				continue
			}

			incoming := s.graph.GetIncoming(currID)
			for _, edge := range incoming {
				if edge.Type != "CALLS" && edge.Type != "REFERENCES" {
					continue
				}

				callerID := edge.Source
				if _, seen := visited[callerID]; !seen {
					visited[callerID] = currDist + 1
					queue = append(queue, callerID)

					if callerNode, ok := s.graph.GetNode(callerID); ok {
						filePath := callerNode.FilePath
						impactedFiles[filePath] = true

						// Check if test file
						if isTestFile(filePath) {
							impactedTestFiles[filePath] = true
						}

						callerDesc := fmt.Sprintf("`%s` in `%s`", callerNode.ID, filePath)
						if callerNode.Symbol != nil {
							callerDesc = fmt.Sprintf("`%s` in `%s:%d`", callerNode.Symbol.Name, filePath, callerNode.Symbol.Location.StartLine)
						}

						if currDist+1 == 1 {
							directCallers = append(directCallers, callerDesc)
						} else {
							transitiveCallers = append(transitiveCallers, fmt.Sprintf("%s (hop %d)", callerDesc, currDist+1))
						}
					}
				}
			}
		}

		// Calculate Risk Severity
		totalAffected := len(visited) - 1
		riskLevel := "LOW (Leaf / Isolated Symbol)"
		if totalAffected > 10 {
			riskLevel = "CRITICAL (Core Architectural Bottleneck)"
		} else if totalAffected > 4 {
			riskLevel = "HIGH (Heavily Depended On)"
		} else if totalAffected > 0 {
			riskLevel = "MEDIUM (Moderately Connected)"
		}

		sb.WriteString(fmt.Sprintf("- **Risk Level**: **%s**\n", riskLevel))
		sb.WriteString(fmt.Sprintf("- **Direct (1-Hop) Callers**: %d\n", len(directCallers)))
		sb.WriteString(fmt.Sprintf("- **Transitive Dependents**: %d\n", len(transitiveCallers)))
		sb.WriteString(fmt.Sprintf("- **Total Impacted Files**: %d\n", len(impactedFiles)))
		sb.WriteString(fmt.Sprintf("- **Impacted Test Suites**: %d\n\n", len(impactedTestFiles)))

		if len(directCallers) > 0 {
			sb.WriteString("### Direct Callers:\n")
			for _, dc := range directCallers {
				sb.WriteString(fmt.Sprintf("- %s\n", dc))
			}
			sb.WriteString("\n")
		}

		if len(transitiveCallers) > 0 {
			sb.WriteString("### Transitive Dependents:\n")
			for _, tc := range transitiveCallers {
				sb.WriteString(fmt.Sprintf("- %s\n", tc))
			}
			sb.WriteString("\n")
		}

		if len(impactedTestFiles) > 0 {
			sb.WriteString("### Suggested Tests to Run:\n")
			for tf := range impactedTestFiles {
				sb.WriteString(fmt.Sprintf("- `%s`\n", tf))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (s *Server) getFileOutline(filePath string) string {
	var symbols []*model.Node
	var matchedFile string

	for _, node := range s.graph.AllNodes() {
		if strings.HasSuffix(node.FilePath, filePath) || node.FilePath == filePath {
			matchedFile = node.FilePath
			if node.Symbol != nil {
				symbols = append(symbols, node)
			}
		}
	}

	if matchedFile == "" {
		return fmt.Sprintf("File '%s' not found in codebase index.", filePath)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# File Outline: `%s`\n\n", matchedFile))
	sb.WriteString(fmt.Sprintf("- **Total Symbols Defined**: %d\n\n", len(symbols)))

	if len(symbols) == 0 {
		sb.WriteString("*No structured symbols extracted for this file.*\n")
		return sb.String()
	}

	for _, n := range symbols {
		sym := n.Symbol
		exportStr := ""
		if sym.Exported {
			exportStr = " [Exported]"
		}
		sb.WriteString(fmt.Sprintf("### `%s` (%s)%s\n", sym.Name, sym.Kind, exportStr))
		sb.WriteString(fmt.Sprintf("- **Line**: %d-%d\n", sym.Location.StartLine, sym.Location.EndLine))
		sb.WriteString(fmt.Sprintf("- **Signature**: `%s`\n", sym.Signature))
		if len(sym.Calls) > 0 {
			sb.WriteString(fmt.Sprintf("- **Calls**: `%s`\n", strings.Join(sym.Calls, "`, `")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasPrefix(filepath.Base(lower), "test_") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "\\test\\") ||
		strings.Contains(lower, "\\tests\\")
}
