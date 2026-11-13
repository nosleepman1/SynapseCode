package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestStorageSaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synapse-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	g := graph.NewGraph()
	node := &model.Node{
		ID:       "node1",
		Type:     model.NodeFile,
		FilePath: "main.go",
	}
	g.AddNode(node)
	g.AddEdge(model.Edge{
		Source: "node1",
		Target: "node2",
		Type:   model.EdgeCalls,
		Weight: 1.0,
	})

	store := NewStore(tempDir)
	if err := store.SaveIndex(g, tempDir); err != nil {
		t.Fatalf("failed to save index: %v", err)
	}

	loadedGraph, meta, err := store.LoadIndex()
	if err != nil {
		t.Fatalf("failed to load index: %v", err)
	}

	if meta.FileCount != 1 {
		t.Errorf("expected 1 file in metadata, got %d", meta.FileCount)
	}

	loadedNode, ok := loadedGraph.GetNode("node1")
	if !ok || loadedNode.FilePath != "main.go" {
		t.Errorf("failed to retrieve loaded node1")
	}

	// Verify file exists on disk
	if _, err := os.Stat(filepath.Join(tempDir, ".synapse", "index.json")); os.IsNotExist(err) {
		t.Errorf("expected index.json to exist on disk")
	}
}
