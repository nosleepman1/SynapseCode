package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/ast/golang"
	"github.com/nosleepman1/synapse-code/internal/ast/python"
	"github.com/nosleepman1/synapse-code/internal/ast/rust"
	"github.com/nosleepman1/synapse-code/internal/ast/typescript"
	appcontext "github.com/nosleepman1/synapse-code/internal/context"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/internal/indexer"
	"github.com/nosleepman1/synapse-code/internal/mcp"
)

var (
	repoPath string
	budget   int
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
		server := mcp.NewServer(path)
		return server.StartStdio()
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

func init() {
	RootCmd.PersistentFlags().StringVarP(&repoPath, "path", "p", "", "Target repository path (default: current directory)")
	contextCmd.Flags().IntVarP(&budget, "budget", "b", 3500, "Maximum token budget for context")

	RootCmd.AddCommand(mcpCmd)
	RootCmd.AddCommand(mapCmd)
	RootCmd.AddCommand(contextCmd)
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

	matcher := discovery.NewIgnoreMatcher(nil, 1024)
	scanner := discovery.NewScanner(matcher)
	idx := indexer.NewIndexer(scanner, reg)

	err := idx.IndexRepository(context.Background(), targetPath, g)
	if err != nil {
		return nil, fmt.Errorf("indexing failed: %w", err)
	}
	return g, nil
}
