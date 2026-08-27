package context

import (
	"context"
	"fmt"

	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/internal/search"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// Builder coordinates search matching, PageRank scoring, and token budgeting.
type Builder struct {
	estimator *TokenEstimator
}

// NewBuilder creates a new context builder.
func NewBuilder() *Builder {
	return &Builder{
		estimator: NewTokenEstimator(),
	}
}

// BuildContextPack generates an optimized context pack under the requested token budget.
func (b *Builder) BuildContextPack(ctx context.Context, g *graph.Graph, taskQuery string, budgetTokens int) (*model.ContextPack, error) {
	if budgetTokens <= 0 {
		budgetTokens = 3500
	}

	allNodes := g.AllNodes()
	if len(allNodes) == 0 {
		return &model.ContextPack{
			TaskDescription: taskQuery,
			BudgetTokens:    budgetTokens,
			UsedTokens:      0,
			FormattedText:   "No code indexed in repository.",
		}, nil
	}

	// 1. Lexical search to find seed nodes
	matches := search.ScoreNodes(allNodes, taskQuery)
	seeds := make(map[model.NodeID]float64)
	for _, m := range matches {
		seeds[m.NodeID] = m.Score
	}

	// 2. Personalized PageRank
	rankedNodes := g.PersonalizedPageRank(seeds, graph.DefaultPageRankConfig())

	// 3. Selection under token budget (Knapsack)
	usedBudget := 0
	var selectedItems []model.ContextItem

	// Reserve 200 tokens for markdown structure
	usableBudget := budgetTokens - 200
	if usableBudget < 500 {
		usableBudget = budgetTokens
	}

	for i, rn := range rankedNodes {
		node := rn.Node
		if node.Symbol == nil {
			continue // Skip raw file nodes for detailed item extraction
		}

		// Top 3 nodes get full body if available, others get signatures
		mode := model.ContextSkeleton
		content := node.Symbol.Signature
		if i < 3 && node.Symbol.Body != "" {
			mode = model.ContextFullCode
			content = fmt.Sprintf("%s {\n%s\n}", node.Symbol.Signature, node.Symbol.Body)
		}

		cost := b.estimator.Count(content)
		if usedBudget+cost <= usableBudget {
			usedBudget += cost
			selectedItems = append(selectedItems, model.ContextItem{
				FilePath:   node.FilePath,
				SymbolName: node.Symbol.Name,
				Kind:       node.Symbol.Kind,
				Mode:       mode,
				Content:    content,
				TokenCount: cost,
				Relevance:  rn.Score,
				Reason:     fmt.Sprintf("PageRank score: %.4f", rn.Score),
			})
		}
	}

	pack := &model.ContextPack{
		TaskDescription: taskQuery,
		BudgetTokens:    budgetTokens,
		UsedTokens:      usedBudget,
		Items:           selectedItems,
	}

	pack.FormattedText = FormatContextPack(pack)
	return pack, nil
}
