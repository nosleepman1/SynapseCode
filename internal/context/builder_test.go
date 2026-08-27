package context

import (
	"context"
	"testing"

	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestContextBuilder(t *testing.T) {
	g := graph.NewGraph()

	sym := model.Symbol{
		Name:      "ValidateToken",
		Kind:      model.KindFunction,
		Signature: "func ValidateToken(token string) bool",
		Body:      "return token != \"\"",
	}

	g.AddNode(&model.Node{
		ID:       "sym:auth.go::ValidateToken",
		Type:     model.NodeSymbol,
		FilePath: "auth.go",
		Symbol:   &sym,
	})

	builder := NewBuilder()
	pack, err := builder.BuildContextPack(context.Background(), g, "ValidateToken auth", 1000)
	if err != nil {
		t.Fatalf("failed to build context pack: %v", err)
	}

	if len(pack.Items) == 0 {
		t.Fatalf("expected at least 1 context item")
	}

	if pack.UsedTokens > pack.BudgetTokens {
		t.Errorf("used tokens (%d) exceeded budget (%d)", pack.UsedTokens, pack.BudgetTokens)
	}

	if pack.Items[0].SymbolName != "ValidateToken" {
		t.Errorf("expected ValidateToken symbol in context pack")
	}
}
