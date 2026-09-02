package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestWorkspace(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "synapse_mcp_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// File 1: core/engine.go
	coreDir := filepath.Join(tempDir, "core")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		t.Fatalf("failed to create core dir: %v", err)
	}
	engineCode := `package core

// ProcessData transforms input payload into normalized output.
func ProcessData(data string) string {
	return "processed:" + data
}
`
	if err := os.WriteFile(filepath.Join(coreDir, "engine.go"), []byte(engineCode), 0644); err != nil {
		t.Fatalf("failed to write engine.go: %v", err)
	}

	// File 2: api/handler.go
	apiDir := filepath.Join(tempDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	handlerCode := `package api

import "core"

func HandleRequest(req string) string {
	return core.ProcessData(req)
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "handler.go"), []byte(handlerCode), 0644); err != nil {
		t.Fatalf("failed to write handler.go: %v", err)
	}

	// File 3: api/handler_test.go
	testCode := `package api

import "testing"

func TestHandleRequest(t *testing.T) {
	HandleRequest("test")
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "handler_test.go"), []byte(testCode), 0644); err != nil {
		t.Fatalf("failed to write handler_test.go: %v", err)
	}

	return tempDir
}

func TestMCPServerLifecycle(t *testing.T) {
	tempDir := createTestWorkspace(t)
	defer os.RemoveAll(tempDir)

	server := NewServer(tempDir)

	// 1. Test Initialize
	initReq := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	initResp := server.handleRequest(initReq)
	if initResp == nil || initResp.Error != nil {
		t.Fatalf("expected initialize success, got error: %v", initResp)
	}

	// 2. Test notifications/initialized
	notifyReq := &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	notifyResp := server.handleRequest(notifyReq)
	if notifyResp != nil {
		t.Errorf("expected nil response for notification, got: %v", notifyResp)
	}

	// 3. Test Unknown method
	unknownReq := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "unknown/method",
	}
	unknownResp := server.handleRequest(unknownReq)
	if unknownResp == nil || unknownResp.Error == nil {
		t.Fatalf("expected error for unknown method")
	}

	// 4. Test tools/list
	toolsListReq := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/list",
	}
	toolsListResp := server.handleRequest(toolsListReq)
	if toolsListResp == nil || toolsListResp.Error != nil {
		t.Fatalf("expected tools/list success")
	}

	resultMap, ok := toolsListResp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map in tools/list")
	}
	tools, ok := resultMap["tools"].([]map[string]interface{})
	if !ok || len(tools) < 6 {
		t.Fatalf("expected at least 6 tools in tools/list, got %d", len(tools))
	}
}

func TestMCPToolCalls(t *testing.T) {
	tempDir := createTestWorkspace(t)
	defer os.RemoveAll(tempDir)

	server := NewServer(tempDir)

	callTool := func(name string, args map[string]interface{}) string {
		params := map[string]interface{}{
			"name":      name,
			"arguments": args,
		}
		paramBytes, _ := json.Marshal(params)
		req := &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      42,
			Method:  "tools/call",
			Params:  paramBytes,
		}
		resp := server.handleRequest(req)
		if resp == nil {
			t.Fatalf("got nil response for tool %s", name)
		}
		if resp.Error != nil {
			t.Fatalf("got error for tool %s: %v", name, resp.Error)
		}
		resMap, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("invalid result map for %s", name)
		}
		contentList, ok := resMap["content"].([]map[string]interface{})
		if !ok || len(contentList) == 0 {
			t.Fatalf("empty content list for %s", name)
		}
		text, _ := contentList[0]["text"].(string)
		return text
	}

	// 1. Test get_repo_map
	repoMap := callTool("get_repo_map", nil)
	if !strings.Contains(repoMap, "Architectural Overview") {
		t.Errorf("get_repo_map missing header, got: %s", repoMap)
	}

	// 2. Test get_context_for_task
	taskContext := callTool("get_context_for_task", map[string]interface{}{
		"task_description": "Process data in engine",
		"budget_tokens":    2000,
	})
	if !strings.Contains(taskContext, "ProcessData") && !strings.Contains(taskContext, "Context Pack") {
		t.Errorf("get_context_for_task expected relevant context, got: %s", taskContext)
	}

	// 3. Test get_symbol_definition
	symDef := callTool("get_symbol_definition", map[string]interface{}{
		"symbol_name": "ProcessData",
	})
	if !strings.Contains(symDef, "ProcessData") || !strings.Contains(symDef, "Signature") {
		t.Errorf("get_symbol_definition missing expected details, got: %s", symDef)
	}

	// 4. Test get_symbol_callers
	callers := callTool("get_symbol_callers", map[string]interface{}{
		"symbol_name": "ProcessData",
	})
	if !strings.Contains(callers, "ProcessData") {
		t.Errorf("get_symbol_callers failed, got: %s", callers)
	}

	// 5. Test get_impact_analysis
	impact := callTool("get_impact_analysis", map[string]interface{}{
		"symbol_name": "ProcessData",
		"max_depth":   3,
	})
	if !strings.Contains(impact, "Blast Radius Analysis") || !strings.Contains(impact, "Risk Level") {
		t.Errorf("get_impact_analysis missing impact details, got: %s", impact)
	}

	// 6. Test get_file_outline
	outline := callTool("get_file_outline", map[string]interface{}{
		"file_path": "core/engine.go",
	})
	if !strings.Contains(outline, "File Outline") || !strings.Contains(outline, "ProcessData") {
		t.Errorf("get_file_outline missing symbol outline, got: %s", outline)
	}
}
