package indexer

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// Indexer coordinates the entire pipeline from file discovery to graph construction.
type Indexer struct {
	scanner  *discovery.Scanner
	registry *ast.Registry
}

// NewIndexer creates a new repository indexer.
func NewIndexer(scanner *discovery.Scanner, registry *ast.Registry) *Indexer {
	return &Indexer{
		scanner:  scanner,
		registry: registry,
	}
}

// IndexRepository scans and parses all candidate source files into the code graph.
func (idx *Indexer) IndexRepository(ctx context.Context, repoPath string, g *graph.Graph) error {
	files, err := idx.scanner.Scan(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("failed to scan repository files: %w", err)
	}

	numWorkers := 8
	fileChan := make(chan discovery.DiscoveredFile, len(files))
	resultChan := make(chan *ast.ParsedFile, len(files))

	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				parser, err := idx.registry.ForFile(f.Path)
				if err != nil {
					continue // Skip unsupported extensions
				}

				content, err := os.ReadFile(f.Path)
				if err != nil {
					continue
				}

				parsed, err := parser.Parse(ctx, f.ID, f.RelPath, content)
				if err != nil {
					continue
				}

				resultChan <- parsed
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Construct graph nodes and edges
	symbolMap := make(map[string]model.NodeID) // name -> NodeID

	for parsed := range resultChan {
		fileNodeID := model.NodeID(fmt.Sprintf("file:%s", parsed.FilePath))

		// 1. Add File Node
		g.AddNode(&model.Node{
			ID:       fileNodeID,
			Type:     model.NodeFile,
			FileID:   parsed.FileID,
			FilePath: parsed.FilePath,
		})

		// 2. Add Symbol Nodes & DEFINES Edges
		for _, sym := range parsed.Symbols {
			symNodeID := model.NodeID(fmt.Sprintf("sym:%s::%s", parsed.FilePath, sym.Name))
			symbolCopy := sym

			g.AddNode(&model.Node{
				ID:        symNodeID,
				Type:      model.NodeSymbol,
				FileID:    parsed.FileID,
				FilePath:  parsed.FilePath,
				Symbol:    &symbolCopy,
				TokenCost: len(sym.Body) / 4,
				SkelCost:  len(sym.Signature) / 4,
			})

			symbolMap[sym.Name] = symNodeID
			symbolMap[sym.QualifiedName] = symNodeID

			// File DEFINES Symbol
			g.AddEdge(model.Edge{
				Source: fileNodeID,
				Target: symNodeID,
				Type:   model.EdgeDefines,
				Weight: 1.0,
			})
		}
	}

	return nil
}
