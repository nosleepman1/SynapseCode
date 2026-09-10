package search

import (
	"sort"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// FusedNodeResult represents a combined ranking result from BM25 and PageRank.
type FusedNodeResult struct {
	NodeID        model.NodeID
	RRFScore      float64
	LexicalRank   int
	LexicalScore  float64
	PageRankRank  int
	PageRankScore float64
}

// ComputeRRF combines lexical search results (BM25) and structural PageRank scores
// using Reciprocal Rank Fusion (RRF) with default k=60.
func ComputeRRF(lexicalResults []MatchResult, pagerankScores map[model.NodeID]float64, k int) []FusedNodeResult {
	if k <= 0 {
		k = 60
	}

	// 1. Sort PageRank entries by descending score to establish ranks
	type prEntry struct {
		id    model.NodeID
		score float64
	}
	var prList []prEntry
	for id, s := range pagerankScores {
		prList = append(prList, prEntry{id: id, score: s})
	}
	sort.Slice(prList, func(i, j int) bool {
		if prList[i].score == prList[j].score {
			return prList[i].id < prList[j].id
		}
		return prList[i].score > prList[j].score
	})

	prRanks := make(map[model.NodeID]int, len(prList))
	for rank, item := range prList {
		prRanks[item.id] = rank + 1 // 1-based rank
	}

	// 2. Track Lexical Ranks
	lexRanks := make(map[model.NodeID]int, len(lexicalResults))
	lexScores := make(map[model.NodeID]float64, len(lexicalResults))
	for rank, item := range lexicalResults {
		lexRanks[item.NodeID] = rank + 1
		lexScores[item.NodeID] = item.Score
	}

	// 3. Compute RRF across all unique candidate nodes
	fusedMap := make(map[model.NodeID]*FusedNodeResult)

	for id, lexRank := range lexRanks {
		prRank, hasPR := prRanks[id]
		prScore := 0.0
		if hasPR {
			prScore = pagerankScores[id]
		}

		rrf := 1.0 / float64(k+lexRank)
		if hasPR {
			rrf += 1.0 / float64(k+prRank)
		}

		fusedMap[id] = &FusedNodeResult{
			NodeID:        id,
			RRFScore:      rrf,
			LexicalRank:   lexRank,
			LexicalScore:  lexScores[id],
			PageRankRank:  prRank,
			PageRankScore: prScore,
		}
	}

	// Also include top PageRank nodes if not already matched
	for id, prRank := range prRanks {
		if _, exists := fusedMap[id]; !exists {
			rrf := 1.0 / float64(k+prRank)
			fusedMap[id] = &FusedNodeResult{
				NodeID:        id,
				RRFScore:      rrf,
				LexicalRank:   0,
				LexicalScore:  0,
				PageRankRank:  prRank,
				PageRankScore: pagerankScores[id],
			}
		}
	}

	var results []FusedNodeResult
	for _, res := range fusedMap {
		results = append(results, *res)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].RRFScore == results[j].RRFScore {
			return results[i].NodeID < results[j].NodeID
		}
		return results[i].RRFScore > results[j].RRFScore
	})

	return results
}
