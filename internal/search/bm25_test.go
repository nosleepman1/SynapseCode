package search

import (
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestBM25SearchAndFieldWeighting(t *testing.T) {
	nodes := []*model.Node{
		{
			ID:       "internal/auth/jwt.go::JWTTokenHandler",
			FilePath: "internal/auth/jwt.go",
			Symbol: &model.Symbol{
				Name:          "JWTTokenHandler",
				QualifiedName: "auth.JWTTokenHandler",
				Signature:     "type JWTTokenHandler struct",
				Documentation: "Handles generation and verification of JWT access tokens.",
			},
		},
		{
			ID:       "internal/auth/service.go::AuthService",
			FilePath: "internal/auth/service.go",
			Symbol: &model.Symbol{
				Name:          "AuthService",
				QualifiedName: "auth.AuthService",
				Signature:     "type AuthService struct",
				Documentation: "Coordinates authentication workflows.",
			},
		},
		{
			ID:       "internal/logger/log.go::LogDebug",
			FilePath: "internal/logger/log.go",
			Symbol: &model.Symbol{
				Name:          "LogDebug",
				QualifiedName: "logger.LogDebug",
				Signature:     "func LogDebug(msg string)",
				Documentation: "Logs debug messages with timestamps.",
			},
		},
	}

	index := NewBM25Index(nodes)
	results := index.Search("jwt token verification")

	if len(results) == 0 {
		t.Fatalf("expected BM25 search results, got none")
	}

	// Top result should be JWTTokenHandler because its name and doc match 'jwt', 'token', 'verification'
	top := results[0]
	if top.NodeID != "internal/auth/jwt.go::JWTTokenHandler" {
		t.Errorf("expected top result to be JWTTokenHandler, got: %s (score: %f)", top.NodeID, top.Score)
	}

	// Test ScoreNodes wrapper
	scored := ScoreNodes(nodes, "AuthService")
	if len(scored) == 0 || scored[0].NodeID != "internal/auth/service.go::AuthService" {
		t.Errorf("expected AuthService to be top scored node")
	}
}

func TestComputeRRF(t *testing.T) {
	lexicalResults := []MatchResult{
		{NodeID: "nodeA", Score: 10.0},
		{NodeID: "nodeB", Score: 5.0},
		{NodeID: "nodeC", Score: 2.0},
	}

	pagerankScores := map[model.NodeID]float64{
		"nodeB": 0.50, // PR rank 1
		"nodeA": 0.30, // PR rank 2
		"nodeD": 0.20, // PR rank 3
	}

	fused := ComputeRRF(lexicalResults, pagerankScores, 60)

	if len(fused) < 4 {
		t.Fatalf("expected at least 4 fused results, got: %d", len(fused))
	}

	// nodeA has lexRank=1 (1/61) and prRank=2 (1/62) -> 0.01639 + 0.01612 = 0.03251
	// nodeB has lexRank=2 (1/62) and prRank=1 (1/61) -> 0.01612 + 0.01639 = 0.03251
	// Both nodeA and nodeB should be higher than nodeC and nodeD
	topScores := map[model.NodeID]bool{
		fused[0].NodeID: true,
		fused[1].NodeID: true,
	}

	if !topScores["nodeA"] || !topScores["nodeB"] {
		t.Errorf("expected nodeA and nodeB to be top 2 in RRF, got %s and %s", fused[0].NodeID, fused[1].NodeID)
	}
}
