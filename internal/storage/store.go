package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// SerializableIndex contains everything required to persist and restore an in-memory graph.
type SerializableIndex struct {
	Metadata IndexMetadata   `json:"metadata"`
	Nodes    []*model.Node   `json:"nodes"`
	Edges    []model.Edge    `json:"edges"`
}

// Store handles local on-disk persistence of the SynapseCode graph index.
type Store struct {
	baseDir string
}

// NewStore creates a new storage manager rooted at the target repository path.
func NewStore(repoPath string) *Store {
	return &Store{
		baseDir: filepath.Join(repoPath, ".synapse"),
	}
}

// SaveIndex serializes the graph and its metadata to `.synapse/index.json`.
func (s *Store) SaveIndex(g *graph.Graph, repoPath string) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory %s: %w", s.baseDir, err)
	}

	nodes := g.AllNodes()
	var edges []model.Edge
	for _, n := range nodes {
		edges = append(edges, g.GetOutgoing(n.ID)...)
	}

	summary := g.Summary()
	index := SerializableIndex{
		Metadata: IndexMetadata{
			Version:     "1.0.0",
			IndexedAt:   time.Now().UTC(),
			RepoPath:    repoPath,
			FileCount:   summary.TotalFiles,
			SymbolCount: summary.TotalSymbols,
			EdgeCount:   summary.TotalEdges,
		},
		Nodes: nodes,
		Edges: edges,
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize index to JSON: %w", err)
	}

	indexPath := filepath.Join(s.baseDir, "index.json")
	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file %s: %w", indexPath, err)
	}

	return nil
}

// LoadIndex deserializes the index from `.synapse/index.json` into a live in-memory Graph.
func (s *Store) LoadIndex() (*graph.Graph, *IndexMetadata, error) {
	indexPath := filepath.Join(s.baseDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read index file %s: %w", indexPath, err)
	}

	var index SerializableIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, nil, fmt.Errorf("failed to parse index JSON: %w", err)
	}

	g := graph.NewGraph()
	for _, node := range index.Nodes {
		g.AddNode(node)
	}
	for _, edge := range index.Edges {
		g.AddEdge(edge)
	}

	return g, &index.Metadata, nil
}
