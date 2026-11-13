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
	"github.com/nosleepman1/synapse-code/internal/storage"
)

func TestIncrementalIndexingLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synapse_inc_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// File 1: helper.go
	f1 := filepath.Join(tempDir, "helper.go")
	if err := os.WriteFile(f1, []byte("package main\n\nfunc Helper() string { return \"ok\" }\n"), 0644); err != nil {
		t.Fatalf("failed to write helper.go: %v", err)
	}

	// File 2: main.go
	f2 := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(f2, []byte("package main\n\nfunc main() { Helper() }\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	reg := ast.NewRegistry()
	reg.Register(golang.NewParser())

	matcher := discovery.NewIgnoreMatcher(nil, 1024)
	scanner := discovery.NewScanner(matcher)
	idx := NewIndexer(scanner, reg)

	g := graph.NewGraph()
	cache := storage.NewFileCache()

	// 1. Cold Run
	stats1, err := idx.IndexIncremental(context.Background(), tempDir, g, cache)
	if err != nil {
		t.Fatalf("cold index failed: %v", err)
	}

	if stats1.Added != 2 || stats1.Cached != 0 {
		t.Errorf("expected 2 added, 0 cached in cold run, got %+v", stats1)
	}
	if g.Summary().TotalSymbols != 2 {
		t.Errorf("expected 2 symbols in graph, got %d", g.Summary().TotalSymbols)
	}

	// 2. Warm Run (Zero changes)
	stats2, err := idx.IndexIncremental(context.Background(), tempDir, g, cache)
	if err != nil {
		t.Fatalf("warm index failed: %v", err)
	}

	if stats2.Cached != 2 || stats2.Added != 0 || stats2.Modified != 0 {
		t.Errorf("expected 2 cached, 0 added in warm run, got %+v", stats2)
	}

	// 3. Modify Run (Edit main.go)
	if err := os.WriteFile(f2, []byte("package main\n\nfunc main() { Helper(); Helper() }\nfunc NewFunc() {}\n"), 0644); err != nil {
		t.Fatalf("failed to edit main.go: %v", err)
	}

	stats3, err := idx.IndexIncremental(context.Background(), tempDir, g, cache)
	if err != nil {
		t.Fatalf("modify index failed: %v", err)
	}

	if stats3.Modified != 1 || stats3.Cached != 1 {
		t.Errorf("expected 1 modified, 1 cached, got %+v", stats3)
	}
	if g.Summary().TotalSymbols != 3 {
		t.Errorf("expected 3 symbols after adding NewFunc, got %d", g.Summary().TotalSymbols)
	}

	// 4. Delete Run (Remove helper.go)
	if err := os.Remove(f1); err != nil {
		t.Fatalf("failed to remove helper.go: %v", err)
	}

	stats4, err := idx.IndexIncremental(context.Background(), tempDir, g, cache)
	if err != nil {
		t.Fatalf("delete index failed: %v", err)
	}

	if stats4.Deleted != 1 {
		t.Errorf("expected 1 deleted file, got %+v", stats4)
	}
}
