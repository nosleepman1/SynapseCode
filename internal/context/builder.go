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

	// 3. Selection under token budget (Adaptive Knapsack)
	usedBudget := 0
	var selectedItems []model.ContextItem
	includedSymbols := make(map[model.NodeID]bool)

	// Reserve 200 tokens for markdown wrapper and headings
	usableBudget := budgetTokens - 200
	if usableBudget < 400 {
		usableBudget = budgetTokens
	}

	// Pass 3.1: Primary items (High-relevance nodes with adaptive full-body expansion)
	for _, rn := range rankedNodes {
		node := rn.Node
		if node.Symbol == nil {
			continue // Skip file nodes
		}

		sym := node.Symbol
		skelContent := sym.Signature
		if sym.Documentation != "" {
			skelContent = fmt.Sprintf("/* %s */\n%s", sym.Documentation, sym.Signature)
		}
		skelCost := b.estimator.Count(skelContent)

		fullContent := skelContent
		fullCost := skelCost
		hasBody := sym.Body != ""
		if hasBody {
			fullContent = fmt.Sprintf("%s {\n%s\n}", sym.Signature, sym.Body)
			if sym.Documentation != "" {
				fullContent = fmt.Sprintf("/* %s */\n%s", sym.Documentation, fullContent)
			}
			fullCost = b.estimator.Count(fullContent)
		}

		// Try to pack full body if relevance is high and budget allows
		if hasBody && rn.Score > 0.01 && usedBudget+fullCost <= usableBudget {
			usedBudget += fullCost
			includedSymbols[node.ID] = true
			selectedItems = append(selectedItems, model.ContextItem{
				FilePath:   node.FilePath,
				SymbolName: sym.Name,
				Kind:       sym.Kind,
				Mode:       model.ContextFullCode,
				Content:    fullContent,
				TokenCount: fullCost,
				Relevance:  rn.Score,
				Reason:     fmt.Sprintf("Primary target (PageRank: %.4f)", rn.Score),
			})
		} else if usedBudget+skelCost <= usableBudget {
			// Otherwise fallback gracefully to signature skeleton
			usedBudget += skelCost
			includedSymbols[node.ID] = true
			selectedItems = append(selectedItems, model.ContextItem{
				FilePath:   node.FilePath,
				SymbolName: sym.Name,
				Kind:       sym.Kind,
				Mode:       model.ContextSkeleton,
				Content:    skelContent,
				TokenCount: skelCost,
				Relevance:  rn.Score,
				Reason:     fmt.Sprintf("Skeleton reference (PageRank: %.4f)", rn.Score),
			})
		}
	}

	// Pass 3.2: 1-hop Dependency Neighbors (Enrich with direct caller/callee contracts)
	for _, item := range selectedItems {
		if usedBudget >= usableBudget {
			break
		}

		// Find symbol node in graph
		for _, n := range allNodes {
			if n.Symbol != nil && n.Symbol.Name == item.SymbolName && n.FilePath == item.FilePath {
				neighbors := g.KHopNeighbors(n.ID, graph.TraversalOpts{
					MaxHops:   1,
					Direction: "outgoing",
				})

				for _, neighbor := range neighbors {
					if neighbor.Node.Symbol == nil || includedSymbols[neighbor.Node.ID] {
						continue
					}

					nsym := neighbor.Node.Symbol
					skelContent := nsym.Signature
					skelCost := b.estimator.Count(skelContent)

					if usedBudget+skelCost <= usableBudget {
						usedBudget += skelCost
						includedSymbols[neighbor.Node.ID] = true
						selectedItems = append(selectedItems, model.ContextItem{
							FilePath:   neighbor.Node.FilePath,
							SymbolName: nsym.Name,
							Kind:       nsym.Kind,
							Mode:       model.ContextSkeleton,
							Content:    skelContent,
							TokenCount: skelCost,
							Relevance:  neighbor.Node.Score,
							Reason:     fmt.Sprintf("1-hop dependency of %s", item.SymbolName),
						})
					}
				}
				break
			}
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
