package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/ast/golang"
	"github.com/nosleepman1/synapse-code/internal/ast/python"
	"github.com/nosleepman1/synapse-code/internal/ast/rust"
	"github.com/nosleepman1/synapse-code/internal/ast/typescript"
	appcontext "github.com/nosleepman1/synapse-code/internal/context"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/internal/indexer"
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
}

// NewServer initializes the MCP server and triggers codebase indexing.
func NewServer(repoPath string) *Server {
	g := graph.NewGraph()
	reg := ast.NewRegistry()
	reg.Register(golang.NewParser())
	reg.Register(typescript.NewParser())
	reg.Register(python.NewParser())
	reg.Register(rust.NewParser())

	matcher := discovery.NewIgnoreMatcher(nil, 1024)
	scanner := discovery.NewScanner(matcher)
	idx := indexer.NewIndexer(scanner, reg)

	_ = idx.IndexRepository(context.Background(), repoPath, g)

	return &Server{
		repoPath:       repoPath,
		graph:          g,
		contextBuilder: appcontext.NewBuilder(),
	}
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
			"description": "Returns a condensed architectural map of the codebase under a token budget.",
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
			"description": "Extracts targeted code implementations and 1-hop AST signatures for a specific task.",
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
			"description": "Finds all callers of a function/method across the codebase to prevent regressions.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the symbol to inspect",
					},
				},
				"required": []string{"symbol_name"},
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
		summary := s.graph.Summary()
		resultText = fmt.Sprintf("# SynapseCode Architectural Map\n- **Total Indexed Files**: %d\n- **Total Extracted Symbols**: %d\n- **Total Graph Relations**: %d\n", summary.TotalFiles, summary.TotalSymbols, summary.TotalEdges)

	case "get_context_for_task":
		taskDesc, _ := params.Arguments["task_description"].(string)
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
		callers := s.findSymbolCallers(symbolName)
		resultText = fmt.Sprintf("# Callers for symbol '%s'\n%s", symbolName, callers)

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

func (s *Server) findSymbolCallers(symbolName string) string {
	var sb strings.Builder
	found := false

	for _, node := range s.graph.AllNodes() {
		if node.Symbol != nil && (node.Symbol.Name == symbolName || node.Symbol.QualifiedName == symbolName) {
			found = true
			incoming := s.graph.GetIncoming(node.ID)
			if len(incoming) == 0 {
				sb.WriteString(fmt.Sprintf("- Defined at `%s` (No incoming callers found in index)\n", node.FilePath))
			} else {
				for _, edge := range incoming {
					sb.WriteString(fmt.Sprintf("- Called by `%s` (Edge: %s)\n", edge.Source, edge.Type))
				}
			}
		}
	}

	if !found {
		return fmt.Sprintf("Symbol '%s' was not found in the codebase index.", symbolName)
	}

	return sb.String()
}
