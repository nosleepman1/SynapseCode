package search

import (
	"strings"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// MatchResult represents a scored symbol or file matching a search query.
type MatchResult struct {
	NodeID model.NodeID
	Score  float64
	Reason string
}

// ScoreNodes scores all graph nodes against the tokenized user query using Okapi BM25.
func ScoreNodes(nodes []*model.Node, query string) []MatchResult {
	if len(nodes) == 0 || strings.TrimSpace(query) == "" {
		return nil
	}

	index := NewBM25Index(nodes)
	return index.Search(query)
}

