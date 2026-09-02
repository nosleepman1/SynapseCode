package context

import (
	"context"
	"strings"
	"testing"

	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestContextBuilder(t *testing.T) {
	g := graph.NewGraph()

	sym1 := model.Symbol{
		Name:          "ValidateToken",
		Kind:          model.KindFunction,
		Signature:     "func ValidateToken(token string) bool",
		Documentation: "ValidateToken checks if JWT token is valid.",
		Body:          "return token != \"\"",
		Calls:         []string{"HashToken"},
	}

	node1ID := model.NodeID("sym:auth.go::ValidateToken")
	g.AddNode(&model.Node{
		ID:       node1ID,
		Type:     model.NodeSymbol,
		FilePath: "auth.go",
		Symbol:   &sym1,
	})

	sym2 := model.Symbol{
		Name:      "HashToken",
		Kind:      model.KindFunction,
		Signature: "func HashToken(s string) string",
		Body:      "return \"hashed:\" + s",
	}

	node2ID := model.NodeID("sym:crypto.go::HashToken")
	g.AddNode(&model.Node{
		ID:       node2ID,
		Type:     model.NodeSymbol,
		FilePath: "crypto.go",
		Symbol:   &sym2,
	})

	// Add CALLS edge from ValidateToken to HashToken
	g.AddEdge(model.Edge{
		Source: node1ID,
		Target: node2ID,
		Type:   model.EdgeCalls,
		Weight: 1.0,
	})

	builder := NewBuilder()

	// Test 1: Full Context Extraction with 1-hop neighbor
	pack, err := builder.BuildContextPack(context.Background(), g, "ValidateToken", 1000)
	if err != nil {
		t.Fatalf("failed to build context pack: %v", err)
	}

	if len(pack.Items) == 0 {
		t.Fatalf("expected at least 1 context item")
	}

	if pack.UsedTokens > pack.BudgetTokens {
		t.Errorf("used tokens (%d) exceeded budget (%d)", pack.UsedTokens, pack.BudgetTokens)
	}

	hasValidateToken := false
	hasHashToken := false
	for _, item := range pack.Items {
		if item.SymbolName == "ValidateToken" {
			hasValidateToken = true
			if item.Mode != model.ContextFullCode {
				t.Errorf("expected ValidateToken to be full code, got %s", item.Mode)
			}
		}
		if item.SymbolName == "HashToken" {
			hasHashToken = true
		}
	}

	if !hasValidateToken {
		t.Errorf("expected ValidateToken in pack")
	}
	if !hasHashToken {
		t.Errorf("expected 1-hop neighbor HashToken to be included in pack")
	}

	if !strings.Contains(pack.FormattedText, "ValidateToken") {
		t.Errorf("formatted text missing ValidateToken")
	}

	// Test 2: Tight budget constraint
	tightPack, err := builder.BuildContextPack(context.Background(), g, "ValidateToken", 50)
	if err != nil {
		t.Fatalf("failed tight budget build: %v", err)
	}
	if tightPack.UsedTokens > 50 {
		t.Errorf("tight budget exceeded: used %d, budget 50", tightPack.UsedTokens)
	}
}
