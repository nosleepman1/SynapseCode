package graph

import (
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestPageRank(t *testing.T) {
	g := NewGraph()

	nodeA := &model.Node{ID: "nodeA", FilePath: "a.go"}
	nodeB := &model.Node{ID: "nodeB", FilePath: "b.go"}
	nodeC := &model.Node{ID: "nodeC", FilePath: "c.go"}

	g.AddNode(nodeA)
	g.AddNode(nodeB)
	g.AddNode(nodeC)

	// A -> B, C -> B (B has high in-degree)
	g.AddEdge(model.Edge{Source: "nodeA", Target: "nodeB", Type: model.EdgeCalls, Weight: 1.0})
	g.AddEdge(model.Edge{Source: "nodeC", Target: "nodeB", Type: model.EdgeCalls, Weight: 1.0})

	seeds := map[model.NodeID]float64{
		"nodeA": 1.0,
		"nodeC": 1.0,
	}

	results := g.PersonalizedPageRank(seeds, DefaultPageRankConfig())

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Node B should be the highest ranked because both A and C call B
	if results[0].Node.ID != "nodeB" {
		t.Errorf("expected nodeB to be highest ranked, got %s", results[0].Node.ID)
	}
}
