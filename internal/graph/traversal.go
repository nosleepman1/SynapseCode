package graph

import (
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// TraversalOpts specifies constraints on graph exploration.
type TraversalOpts struct {
	MaxHops   int
	Direction string // "outgoing", "incoming", "both"
	EdgeTypes []model.EdgeType
}

// HopNode pairs a graph node with the distance (number of hops) from starting seed.
type HopNode struct {
	Node *model.Node
	Hop  int
}

// KHopNeighbors extracts the ego-network around a starting node up to maxHops distance.
func (g *Graph) KHopNeighbors(startID model.NodeID, opts TraversalOpts) []HopNode {
	if opts.MaxHops <= 0 {
		opts.MaxHops = 1
	}

	visited := make(map[model.NodeID]int)
	visited[startID] = 0

	queue := []model.NodeID{startID}
	var results []HopNode

	allowedEdges := make(map[model.EdgeType]bool)
	for _, et := range opts.EdgeTypes {
		allowedEdges[et] = true
	}

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]
		currHop := visited[currID]

		if currHop >= opts.MaxHops {
			continue
		}

		var candidateEdges []model.Edge
		if opts.Direction == "incoming" || opts.Direction == "both" {
			candidateEdges = append(candidateEdges, g.GetIncoming(currID)...)
		}
		if opts.Direction == "outgoing" || opts.Direction == "both" || opts.Direction == "" {
			candidateEdges = append(candidateEdges, g.GetOutgoing(currID)...)
		}

		for _, e := range candidateEdges {
			if len(allowedEdges) > 0 && !allowedEdges[e.Type] {
				continue
			}

			neighborID := e.Target
			if neighborID == currID {
				neighborID = e.Source
			}

			if _, seen := visited[neighborID]; !seen {
				visited[neighborID] = currHop + 1
				if node, ok := g.GetNode(neighborID); ok {
					results = append(results, HopNode{
						Node: node,
						Hop:  currHop + 1,
					})
					queue = append(queue, neighborID)
				}
			}
		}
	}

	return results
}
