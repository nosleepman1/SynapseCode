package graph

import (
	"fmt"
	"sync"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// Graph represents a thread-safe, in-memory directed multi-graph of code entities.
type Graph struct {
	mu       sync.RWMutex
	nodes    map[model.NodeID]*model.Node
	outgoing map[model.NodeID][]model.Edge
	incoming map[model.NodeID][]model.Edge
}

// NewGraph initializes an empty in-memory code graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:    make(map[model.NodeID]*model.Node),
		outgoing: make(map[model.NodeID][]model.Edge),
		incoming: make(map[model.NodeID][]model.Edge),
	}
}

// AddNode adds or updates a vertex in the graph.
func (g *Graph) AddNode(node *model.Node) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes[node.ID] = node
}

// GetNode retrieves a node by its unique ID.
func (g *Graph) GetNode(id model.NodeID) (*model.Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	n, ok := g.nodes[id]
	return n, ok
}

// AddEdge inserts a directed edge from source to target.
func (g *Graph) AddEdge(edge model.Edge) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if edge.Weight == 0 {
		edge.Weight = 1.0
	}
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("%s->%s:%s", edge.Source, edge.Target, edge.Type)
	}

	g.outgoing[edge.Source] = append(g.outgoing[edge.Source], edge)
	g.incoming[edge.Target] = append(g.incoming[edge.Target], edge)
}

// GetOutgoing returns all edges originating from a node.
func (g *Graph) GetOutgoing(id model.NodeID) []model.Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := g.outgoing[id]
	result := make([]model.Edge, len(edges))
	copy(result, edges)
	return result
}

// GetIncoming returns all edges targeting a node (callers/dependents).
func (g *Graph) GetIncoming(id model.NodeID) []model.Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := g.incoming[id]
	result := make([]model.Edge, len(edges))
	copy(result, edges)
	return result
}

// AllNodes returns a slice of all nodes currently stored in the graph.
func (g *Graph) AllNodes() []*model.Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]*model.Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// NodeCount returns total number of vertices in the graph.
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// Summary returns high-level graph statistics.
func (g *Graph) Summary() model.GraphSummary {
	g.mu.RLock()
	defer g.mu.RUnlock()

	totalEdges := 0
	for _, edges := range g.outgoing {
		totalEdges += len(edges)
	}

	fileCount := 0
	symbolCount := 0
	for _, n := range g.nodes {
		if n.Type == model.NodeFile {
			fileCount++
		} else {
			symbolCount++
		}
	}

	return model.GraphSummary{
		TotalFiles:   fileCount,
		TotalSymbols: symbolCount,
		TotalEdges:   totalEdges,
	}
}
