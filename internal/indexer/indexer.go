package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Collect all parsed results
	var parsedFiles []*ast.ParsedFile
	for parsed := range resultChan {
		parsedFiles = append(parsedFiles, parsed)
	}

	// Index mappings for fast resolution
	fileNodes := make(map[string]model.NodeID)                      // filePath -> fileNodeID
	symbolsByFileAndName := make(map[string]map[string]model.NodeID) // filePath -> symName -> symNodeID
	symbolsByName := make(map[string][]model.NodeID)                // symName -> []symNodeID
	symbolNodeMap := make(map[model.NodeID]*model.Node)             // nodeID -> Node

	// PASS 1: Register all File nodes, Symbol nodes, and DEFINES edges
	for _, parsed := range parsedFiles {
		fileNodeID := model.NodeID(fmt.Sprintf("file:%s", parsed.FilePath))
		fileNodes[parsed.FilePath] = fileNodeID

		fileNode := &model.Node{
			ID:       fileNodeID,
			Type:     model.NodeFile,
			FileID:   parsed.FileID,
			FilePath: parsed.FilePath,
		}
		g.AddNode(fileNode)
		symbolNodeMap[fileNodeID] = fileNode

		if symbolsByFileAndName[parsed.FilePath] == nil {
			symbolsByFileAndName[parsed.FilePath] = make(map[string]model.NodeID)
		}

		for _, sym := range parsed.Symbols {
			symNodeID := model.NodeID(fmt.Sprintf("sym:%s::%s", parsed.FilePath, sym.Name))
			symbolCopy := sym

			symNode := &model.Node{
				ID:        symNodeID,
				Type:      model.NodeSymbol,
				FileID:    parsed.FileID,
				FilePath:  parsed.FilePath,
				Symbol:    &symbolCopy,
				TokenCost: len(sym.Body) / 4,
				SkelCost:  len(sym.Signature) / 4,
			}
			g.AddNode(symNode)
			symbolNodeMap[symNodeID] = symNode

			symbolsByFileAndName[parsed.FilePath][sym.Name] = symNodeID
			symbolsByFileAndName[parsed.FilePath][sym.QualifiedName] = symNodeID
			symbolsByName[sym.Name] = append(symbolsByName[sym.Name], symNodeID)

			// File DEFINES Symbol
			g.AddEdge(model.Edge{
				Source: fileNodeID,
				Target: symNodeID,
				Type:   model.EdgeDefines,
				Weight: 1.0,
			})
		}
	}

	// PASS 2: Resolve CALLS, IMPORTS, and IMPLEMENTS edges
	for _, parsed := range parsedFiles {
		fileNodeID := fileNodes[parsed.FilePath]

		// 2.1 Resolve IMPORTS edges (File -> Imported File)
		for _, imp := range parsed.Imports {
			for targetPath, targetNodeID := range fileNodes {
				if targetPath == parsed.FilePath {
					continue
				}
				// Match relative paths, exact matches, or package path endings
				if strings.HasSuffix(targetPath, imp) ||
					strings.Contains(targetPath, strings.TrimPrefix(imp, "./")) ||
					strings.HasSuffix(imp, filepath.Base(filepath.Dir(targetPath))) {
					g.AddEdge(model.Edge{
						Source: fileNodeID,
						Target: targetNodeID,
						Type:   model.EdgeImports,
						Weight: 1.0,
					})
				}
			}
		}

		// 2.2 Resolve CALLS edges (Symbol -> Target Symbol)
		for _, sym := range parsed.Symbols {
			sourceSymNodeID := symbolsByFileAndName[parsed.FilePath][sym.Name]

			for _, callName := range sym.Calls {
				cleanCall := strings.TrimSpace(callName)
				if cleanCall == "" {
					continue
				}

				// If call is like obj.Method() or pkg.Func(), check both full and base name
				baseName := cleanCall
				if dotIdx := strings.LastIndex(cleanCall, "."); dotIdx != -1 && dotIdx < len(cleanCall)-1 {
					baseName = cleanCall[dotIdx+1:]
				}

				resolved := false

				// Priority 1: Intra-file symbol resolution
				if targetID, ok := symbolsByFileAndName[parsed.FilePath][cleanCall]; ok {
					g.AddEdge(model.Edge{
						Source: sourceSymNodeID,
						Target: targetID,
						Type:   model.EdgeCalls,
						Weight: 2.0,
					})
					resolved = true
				} else if targetID, ok := symbolsByFileAndName[parsed.FilePath][baseName]; ok {
					g.AddEdge(model.Edge{
						Source: sourceSymNodeID,
						Target: targetID,
						Type:   model.EdgeCalls,
						Weight: 2.0,
					})
					resolved = true
				}

				// Priority 2: Global workspace symbol resolution
				if !resolved {
					if targets, ok := symbolsByName[baseName]; ok && len(targets) > 0 {
						for _, targetID := range targets {
							if targetID == sourceSymNodeID {
								continue // Avoid trivial self-call if others exist
							}
							g.AddEdge(model.Edge{
								Source: sourceSymNodeID,
								Target: targetID,
								Type:   model.EdgeCalls,
								Weight: 1.0,
							})
							resolved = true
						}
					}
				}
			}

			// 2.3 Resolve IMPLEMENTS / EXTENDS edges
			for _, implName := range sym.Implements {
				if targets, ok := symbolsByName[implName]; ok {
					for _, targetID := range targets {
						g.AddEdge(model.Edge{
							Source: sourceSymNodeID,
							Target: targetID,
							Type:   model.EdgeImplements,
							Weight: 1.5,
						})
					}
				}
			}
		}
	}

	return nil
}
