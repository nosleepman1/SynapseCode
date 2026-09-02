package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/ast/golang"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestIndexerResolution(t *testing.T) {
	// Create temporary mock repository
	tempDir, err := os.MkdirTemp("", "synapse_test_repo_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// File 1: utils/helper.go
	utilsDir := filepath.Join(tempDir, "utils")
	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		t.Fatalf("failed to create utils dir: %v", err)
	}
	helperCode := `package utils

func FormatString(s string) string {
	return s
}
`
	if err := os.WriteFile(filepath.Join(utilsDir, "helper.go"), []byte(helperCode), 0644); err != nil {
		t.Fatalf("failed to write helper.go: %v", err)
	}

	// File 2: service/service.go
	svcDir := filepath.Join(tempDir, "service")
	if err := os.MkdirAll(svcDir, 0755); err != nil {
		t.Fatalf("failed to create svc dir: %v", err)
	}
	serviceCode := `package service

import "utils"

func ExecuteTask(task string) string {
	return FormatString(task)
}
`
	if err := os.WriteFile(filepath.Join(svcDir, "service.go"), []byte(serviceCode), 0644); err != nil {
		t.Fatalf("failed to write service.go: %v", err)
	}

	// File 3: main.go
	mainCode := `package main

import "service"

func main() {
	service.ExecuteTask("hello")
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainCode), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Setup Indexer
	reg := ast.NewRegistry()
	reg.Register(golang.NewParser())

	matcher := discovery.NewIgnoreMatcher(nil, 1024)
	scanner := discovery.NewScanner(matcher)
	idx := NewIndexer(scanner, reg)

	g := graph.NewGraph()
	if err := idx.IndexRepository(context.Background(), tempDir, g); err != nil {
		t.Fatalf("IndexRepository failed: %v", err)
	}

	summary := g.Summary()
	if summary.TotalFiles < 3 {
		t.Errorf("expected at least 3 files, got %d", summary.TotalFiles)
	}
	if summary.TotalSymbols < 3 {
		t.Errorf("expected at least 3 symbols, got %d", summary.TotalSymbols)
	}

	// Verify CALLS edge from ExecuteTask to FormatString
	var executeTaskID model.NodeID
	var formatStringID model.NodeID

	for _, n := range g.AllNodes() {
		if n.Symbol != nil {
			if n.Symbol.Name == "ExecuteTask" {
				executeTaskID = n.ID
			}
			if n.Symbol.Name == "FormatString" {
				formatStringID = n.ID
			}
		}
	}

	if executeTaskID == "" {
		t.Fatalf("ExecuteTask symbol not found in graph")
	}
	if formatStringID == "" {
		t.Fatalf("FormatString symbol not found in graph")
	}

	// Check outgoing edges from ExecuteTask
	outgoing := g.GetOutgoing(executeTaskID)
	hasCallToFormat := false
	for _, edge := range outgoing {
		if edge.Target == formatStringID && edge.Type == model.EdgeCalls {
			hasCallToFormat = true
			break
		}
	}

	if !hasCallToFormat {
		t.Errorf("expected EdgeCalls from ExecuteTask (%s) to FormatString (%s)", executeTaskID, formatStringID)
	}

	// Check incoming edges to FormatString
	incoming := g.GetIncoming(formatStringID)
	hasCallerExecuteTask := false
	for _, edge := range incoming {
		if edge.Source == executeTaskID && edge.Type == model.EdgeCalls {
			hasCallerExecuteTask = true
			break
		}
	}

	if !hasCallerExecuteTask {
		t.Errorf("expected incoming caller ExecuteTask for FormatString")
	}
}
