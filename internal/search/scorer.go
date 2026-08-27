package search

import (
	"sort"
	"strings"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// MatchResult represents a scored symbol or file matching a search query.
type MatchResult struct {
	NodeID model.NodeID
	Score  float64
	Reason string
}

// ScoreNodes scores all graph nodes against the tokenized user query.
func ScoreNodes(nodes []*model.Node, query string) []MatchResult {
	tokens := TokenizeQuery(query)
	if len(tokens) == 0 {
		return nil
	}

	var results []MatchResult

	for _, node := range nodes {
		score := 0.0
		var reasons []string

		targetText := strings.ToLower(node.FilePath)
		symbolName := ""
		signature := ""

		if node.Symbol != nil {
			symbolName = strings.ToLower(node.Symbol.Name)
			signature = strings.ToLower(node.Symbol.Signature)
			targetText += " " + symbolName + " " + signature + " " + strings.ToLower(node.Symbol.Documentation)
		}

		for _, token := range tokens {
			// Exact symbol name match (highest value)
			if symbolName == token {
				score += 10.0
				reasons = append(reasons, "exact_symbol_name")
			} else if strings.Contains(symbolName, token) {
				score += 4.0
				reasons = append(reasons, "partial_symbol_name")
			}

			// Signature match
			if strings.Contains(signature, token) {
				score += 2.0
				reasons = append(reasons, "signature_match")
			}

			// File path match
			if strings.Contains(targetText, token) {
				score += 1.0
				reasons = append(reasons, "path_or_doc_match")
			}
		}

		if score > 0 {
			results = append(results, MatchResult{
				NodeID: node.ID,
				Score:  score,
				Reason: strings.Join(reasons, ","),
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].NodeID < results[j].NodeID
		}
		return results[i].Score > results[j].Score
	})

	return results
}
