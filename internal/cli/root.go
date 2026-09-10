package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/ast/golang"
	"github.com/nosleepman1/synapse-code/internal/ast/java"
	"github.com/nosleepman1/synapse-code/internal/ast/php"
	"github.com/nosleepman1/synapse-code/internal/ast/python"
	"github.com/nosleepman1/synapse-code/internal/ast/rust"
	"github.com/nosleepman1/synapse-code/internal/ast/typescript"
	appcontext "github.com/nosleepman1/synapse-code/internal/context"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/internal/indexer"
	"github.com/nosleepman1/synapse-code/internal/mcp"
	"github.com/nosleepman1/synapse-code/internal/search"
	"github.com/nosleepman1/synapse-code/internal/storage"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

var (
	repoPath   string
	budget     int
	watchMode  bool
	forceIndex bool
)

var RootCmd = &cobra.Command{
	Use:   "synapse",
	Short: "SynapseCode - High Performance AST & Code-Graph MCP Server",
	Long:  `SynapseCode indexes source code into an in-memory dependency graph and optimizes context for LLMs via MCP, saving 75%-90% tokens.`,
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol (MCP) server for Claude Desktop / AI Agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := getTargetRepoPath()
		server := mcp.NewServerWithOptions(path, watchMode)
		defer server.Close()
		return server.StartStdio()
	},
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Scan, parse, and persist codebase graph index to .synapse/cache.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := getTargetRepoPath()
		g := graph.NewGraph()
		reg := ast.NewRegistry()
		reg.Register(golang.NewParser())
		reg.Register(typescript.NewParser())
		reg.Register(python.NewParser())
		reg.Register(rust.NewParser())
		reg.Register(java.NewParser())
		reg.Register(php.NewParser())

		matcher := discovery.NewIgnoreMatcher(nil, 1024)
		scanner := discovery.NewScanner(matcher)
		idx := indexer.NewIndexer(scanner, reg)

		var cache *storage.FileCache
		if forceIndex {
			cache = storage.NewFileCache()
		} else {
			cache, _ = storage.LoadCache(path)
		}

		stats, err := idx.IndexIncremental(context.Background(), path, g, cache)
		if err != nil {
			return fmt.Errorf("indexing failed: %w", err)
		}

		summary := g.Summary()
		fmt.Printf("✓ Codebase indexed successfully in %v\n", stats.Duration)
		fmt.Printf("  - Discovered files: %d\n", stats.TotalDiscovered)
		fmt.Printf("  - Cached hits:     %d\n", stats.Cached)
		fmt.Printf("  - Freshly parsed:  %d (Added: %d, Modified: %d)\n", stats.Added+stats.Modified, stats.Added, stats.Modified)
		fmt.Printf("  - Pruned files:    %d\n", stats.Deleted)
		fmt.Printf("  - In-Memory Graph: %d symbols, %d dependency edges\n", summary.TotalSymbols, summary.TotalEdges)
		fmt.Printf("  - Cache location:  %s\n", storage.CacheFilePath(path))
		return nil
	},
}

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Generate a condensed architectural map of the codebase",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := getTargetRepoPath()
		g, err := buildGraph(path)
		if err != nil {
			return err
		}

		summary := g.Summary()
		fmt.Printf("# SynapseCode Architectural Overview (%s)\n", path)
		fmt.Printf("- **Total Files**: %d\n", summary.TotalFiles)
		fmt.Printf("- **Total Symbols**: %d\n", summary.TotalSymbols)
		fmt.Printf("- **Total Dependency Edges**: %d\n\n", summary.TotalEdges)

		fmt.Println("## Top Indexed Symbols:")
		nodes := g.AllNodes()
		count := 0
		for _, n := range nodes {
			if n.Symbol != nil && count < 25 {
				fmt.Printf("- `%s` (%s) in `%s`\n", n.Symbol.Name, n.Symbol.Kind, n.FilePath)
				count++
			}
		}
		return nil
	},
}

var contextCmd = &cobra.Command{
	Use:   "context [task description]",
	Short: "Extract targeted context for a coding task under a token budget",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := getTargetRepoPath()
		g, err := buildGraph(path)
		if err != nil {
			return err
		}

		task := args[0]
		builder := appcontext.NewBuilder()
		pack, err := builder.BuildContextPack(context.Background(), g, task, budget)
		if err != nil {
			return fmt.Errorf("failed to build context pack: %w", err)
		}

		fmt.Println(pack.FormattedText)
		return nil
	},
}

var queryCmd = &cobra.Command{
	Use:   "query [search terms]",
	Short: "Search codebase symbols using Okapi BM25 and Reciprocal Rank Fusion",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := getTargetRepoPath()
		g, err := buildGraph(path)
		if err != nil {
			return err
		}

		query := strings.Join(args, " ")
		allNodes := g.AllNodes()
		matches := search.ScoreNodes(allNodes, query)

		fmt.Printf("# BM25 & RRF Search Results for \"%s\" (%s)\n\n", query, path)
		if len(matches) == 0 {
			fmt.Println("No matching symbols or files found.")
			return nil
		}

		// Calculate PageRank and RRF
		seeds := make(map[model.NodeID]float64)
		for _, m := range matches {
			seeds[m.NodeID] = m.Score
		}
		rankedNodes := g.PersonalizedPageRank(seeds, graph.DefaultPageRankConfig())
		prScores := make(map[model.NodeID]float64, len(rankedNodes))
		for _, rn := range rankedNodes {
			prScores[rn.Node.ID] = rn.Score
		}
		fused := search.ComputeRRF(matches, prScores, 60)

		nodeMap := make(map[model.NodeID]*model.Node, len(allNodes))
		for _, n := range allNodes {
			nodeMap[n.ID] = n
		}

		count := 0
		for _, item := range fused {
			if count >= 15 {
				break
			}
			node := nodeMap[item.NodeID]
			if node == nil || node.Symbol == nil {
				continue
			}
			sym := node.Symbol
			fmt.Printf("%2d. [%.4f RRF | BM25: #%d (%.2f) | PR: #%d (%.4f)] `%s` (%s)\n",
				count+1, item.RRFScore, item.LexicalRank, item.LexicalScore, item.PageRankRank, item.PageRankScore, sym.QualifiedName, sym.Kind)
			fmt.Printf("    File: %s:%d\n", node.FilePath, sym.Location.StartLine)
			if sym.Documentation != "" {
				fmt.Printf("    Doc:  %s\n", sym.Documentation)
			}
			count++
		}
		return nil
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&repoPath, "path", "p", "", "Target repository path (default: current directory)")
	contextCmd.Flags().IntVarP(&budget, "budget", "b", 3500, "Maximum token budget for context")
	mcpCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Enable live event-driven filesystem watcher")
	indexCmd.Flags().BoolVarP(&forceIndex, "force", "f", false, "Force re-indexing all files, bypassing existing cache")

	RootCmd.AddCommand(mcpCmd)
	RootCmd.AddCommand(indexCmd)
	RootCmd.AddCommand(mapCmd)
	RootCmd.AddCommand(contextCmd)
	RootCmd.AddCommand(queryCmd)
}

func getTargetRepoPath() string {
	if repoPath != "" {
		return repoPath
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func buildGraph(targetPath string) (*graph.Graph, error) {
	g := graph.NewGraph()
	reg := ast.NewRegistry()
	reg.Register(golang.NewParser())
	reg.Register(typescript.NewParser())
	reg.Register(python.NewParser())
	reg.Register(rust.NewParser())
	reg.Register(java.NewParser())
	reg.Register(php.NewParser())

	matcher := discovery.NewIgnoreMatcher(nil, 1024)
	scanner := discovery.NewScanner(matcher)
	idx := indexer.NewIndexer(scanner, reg)

	cache, _ := storage.LoadCache(targetPath)
	_, err := idx.IndexIncremental(context.Background(), targetPath, g, cache)
	if err != nil {
		return nil, fmt.Errorf("indexing failed: %w", err)
	}
	return g, nil
}
