package graph

import (
	"math"
	"sort"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// PageRankConfig configures the Personalized PageRank algorithm.
type PageRankConfig struct {
	DampingFactor float64
	MaxIterations int
	Tolerance     float64
}

// DefaultPageRankConfig returns balanced default parameters.
func DefaultPageRankConfig() PageRankConfig {
	return PageRankConfig{
		DampingFactor: 0.85,
		MaxIterations: 25,
		Tolerance:     1e-6,
	}
}

// RankedNode pairs a graph node with its computed PageRank score.
type RankedNode struct {
	Node  *model.Node
	Score float64
}

// PersonalizedPageRank executes the PPR algorithm biased towards seeds.
func (g *Graph) PersonalizedPageRank(seeds map[model.NodeID]float64, cfg PageRankConfig) []RankedNode {
	nodes := g.AllNodes()
	n := len(nodes)
	if n == 0 {
		return nil
	}

	nodeIndices := make(map[model.NodeID]int, n)
	for i, node := range nodes {
		nodeIndices[node.ID] = i
	}

	// 1. Build teleport / personalization vector (p0)
	p0 := make([]float64, n)
	seedSum := 0.0
	for id, weight := range seeds {
		if idx, ok := nodeIndices[id]; ok {
			p0[idx] = weight
			seedSum += weight
		}
	}

	// If no valid seeds, distribute uniformly
	if seedSum == 0 {
		uniform := 1.0 / float64(n)
		for i := range p0 {
			p0[i] = uniform
		}
	} else {
		for i := range p0 {
			p0[i] /= seedSum
		}
	}

	// 2. Initialize current rank vector
	rank := make([]float64, n)
	copy(rank, p0)

	// Pre-calculate outgoing weights
	outWeights := make([]float64, n)
	for i, node := range nodes {
		edges := g.GetOutgoing(node.ID)
		sum := 0.0
		for _, e := range edges {
			sum += e.Weight
		}
		outWeights[i] = sum
	}

	d := cfg.DampingFactor
	if d <= 0 || d >= 1 {
		d = 0.85
	}
	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = 25
	}
	tol := cfg.Tolerance
	if tol <= 0 {
		tol = 1e-6
	}

	// 3. Power Iterations
	nextRank := make([]float64, n)
	for iter := 0; iter < maxIter; iter++ {
		danglingSum := 0.0
		for i := range nextRank {
			nextRank[i] = 0.0
		}

		for i, node := range nodes {
			if outWeights[i] == 0 {
				danglingSum += rank[i]
				continue
			}

			edges := g.GetOutgoing(node.ID)
			for _, e := range edges {
				if targetIdx, ok := nodeIndices[e.Target]; ok {
					nextRank[targetIdx] += rank[i] * (e.Weight / outWeights[i])
				}
			}
		}

		// Apply damping, teleportation, and dangling node redistribution
		diff := 0.0
		for i := range nextRank {
			nextRank[i] = d*(nextRank[i]+danglingSum*p0[i]) + (1.0-d)*p0[i]
			diff += math.Abs(nextRank[i] - rank[i])
		}

		copy(rank, nextRank)
		if diff < tol {
			break
		}
	}

	// 4. Collect and sort results stably
	results := make([]RankedNode, n)
	for i, node := range nodes {
		results[i] = RankedNode{
			Node:  node,
			Score: rank[i],
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Node.ID < results[j].Node.ID // Deterministic tie-breaking
		}
		return results[i].Score > results[j].Score
	})

	return results
}
