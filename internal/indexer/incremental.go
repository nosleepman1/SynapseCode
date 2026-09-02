package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/internal/discovery"
	"github.com/nosleepman1/synapse-code/internal/graph"
	"github.com/nosleepman1/synapse-code/internal/storage"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// IncrementalStats tracks the efficiency metrics of an incremental indexing run.
type IncrementalStats struct {
	TotalDiscovered int           `json:"total_discovered"`
	Added           int           `json:"added"`
	Modified        int           `json:"modified"`
	Deleted         int           `json:"deleted"`
	Cached          int           `json:"cached"`
	Duration        time.Duration `json:"duration"`
}

// IndexIncremental indexes only added/modified files while leveraging the disk cache for unchanged files.
func (idx *Indexer) IndexIncremental(ctx context.Context, repoPath string, g *graph.Graph, cache *storage.FileCache) (*IncrementalStats, error) {
	start := time.Now()

	files, err := idx.scanner.Scan(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan repository files: %w", err)
	}

	stats := &IncrementalStats{
		TotalDiscovered: len(files),
	}

	discoveredMap := make(map[string]discovery.DiscoveredFile)
	for _, f := range files {
		discoveredMap[f.RelPath] = f
	}

	// 1. Detect and prune Deleted files
	var deletedPaths []string
	if cache != nil {
		for relPath := range cache.Entries {
			if _, exists := discoveredMap[relPath]; !exists {
				deletedPaths = append(deletedPaths, relPath)
			}
		}
	}

	for _, delPath := range deletedPaths {
		g.RemoveFileNodes(delPath)
		if cache != nil {
			cache.Remove(delPath)
		}
		stats.Deleted++
	}

	// 2. Classify files: Cached vs NeedsParsing
	type parseJob struct {
		file     discovery.DiscoveredFile
		content  []byte
		checksum string
		modTime  int64
		isNew    bool
	}

	var jobs []parseJob
	var parsedFiles []*ast.ParsedFile

	for _, f := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		info, err := os.Stat(f.Path)
		if err != nil {
			continue
		}

		modTimeUnix := info.ModTime().UnixNano()
		content, err := os.ReadFile(f.Path)
		if err != nil {
			continue
		}

		checksum := storage.ComputeChecksum(content)

		if cache != nil && cache.IsFresh(f.RelPath, f.Size, modTimeUnix, checksum) {
			// Cache Hit: reconstruct ParsedFile from cache without running AST parser
			cachedEntry, ok := cache.Get(f.RelPath)
			if ok {
				stats.Cached++
				parsedFiles = append(parsedFiles, &ast.ParsedFile{
					FileID:   f.ID,
					FilePath: f.RelPath,
					Symbols:  cachedEntry.Symbols,
					Imports:  cachedEntry.Imports,
				})
				continue
			}
		}

		// Cache Miss: queue for concurrent parsing
		isNew := true
		if cache != nil {
			if _, exists := cache.Get(f.RelPath); exists {
				isNew = false
				stats.Modified++
			} else {
				stats.Added++
			}
		} else {
			stats.Added++
		}

		jobs = append(jobs, parseJob{
			file:     f,
			content:  content,
			checksum: checksum,
			modTime:  modTimeUnix,
			isNew:    isNew,
		})
	}

	// 3. Concurrently parse modified / added files
	numWorkers := 8
	if len(jobs) < numWorkers {
		numWorkers = len(jobs)
	}
	if numWorkers <= 0 {
		numWorkers = 1
	}

	jobChan := make(chan parseJob, len(jobs))
	for _, j := range jobs {
		jobChan <- j
	}
	close(jobChan)

	type parseResult struct {
		parsed   *ast.ParsedFile
		job      parseJob
		err      error
	}

	resultChan := make(chan parseResult, len(jobs))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobChan {
				parser, err := idx.registry.ForFile(j.file.Path)
				if err != nil {
					continue
				}

				parsed, err := parser.Parse(ctx, j.file.ID, j.file.RelPath, j.content)
				if err != nil {
					continue
				}

				resultChan <- parseResult{
					parsed: parsed,
					job:    j,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect freshly parsed results and update cache
	for res := range resultChan {
		if !res.job.isNew {
			g.RemoveFileNodes(res.job.file.RelPath)
		}

		parsedFiles = append(parsedFiles, res.parsed)

		if cache != nil {
			cache.Put(&storage.CachedFileEntry{
				RelPath:     res.job.file.RelPath,
				SizeBytes:   res.job.file.Size,
				ModTimeUnix: res.job.modTime,
				Checksum:    res.job.checksum,
				Symbols:     res.parsed.Symbols,
				Imports:     res.parsed.Imports,
			})
		}
	}

	// 4. Reconstruct Graph & Inter-symbol Edge Resolution
	fileNodes := make(map[string]model.NodeID)
	symbolsByFileAndName := make(map[string]map[string]model.NodeID)
	symbolsByName := make(map[string][]model.NodeID)

	for _, parsed := range parsedFiles {
		fileNodeID := model.NodeID(fmt.Sprintf("file:%s", parsed.FilePath))
		fileNodes[parsed.FilePath] = fileNodeID

		g.AddNode(&model.Node{
			ID:       fileNodeID,
			Type:     model.NodeFile,
			FileID:   parsed.FileID,
			FilePath: parsed.FilePath,
		})

		if symbolsByFileAndName[parsed.FilePath] == nil {
			symbolsByFileAndName[parsed.FilePath] = make(map[string]model.NodeID)
		}

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

			symbolsByFileAndName[parsed.FilePath][sym.Name] = symNodeID
			symbolsByFileAndName[parsed.FilePath][sym.QualifiedName] = symNodeID
			symbolsByName[sym.Name] = append(symbolsByName[sym.Name], symNodeID)

			g.AddEdge(model.Edge{
				Source: fileNodeID,
				Target: symNodeID,
				Type:   model.EdgeDefines,
				Weight: 1.0,
			})
		}
	}

	// Resolve CALLS, IMPORTS, and IMPLEMENTS
	for _, parsed := range parsedFiles {
		fileNodeID := fileNodes[parsed.FilePath]

		for _, imp := range parsed.Imports {
			for targetPath, targetNodeID := range fileNodes {
				if targetPath == parsed.FilePath {
					continue
				}
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

		for _, sym := range parsed.Symbols {
			sourceSymNodeID := symbolsByFileAndName[parsed.FilePath][sym.Name]

			for _, callName := range sym.Calls {
				cleanCall := strings.TrimSpace(callName)
				if cleanCall == "" {
					continue
				}

				baseName := cleanCall
				if dotIdx := strings.LastIndex(cleanCall, "."); dotIdx != -1 && dotIdx < len(cleanCall)-1 {
					baseName = cleanCall[dotIdx+1:]
				}

				resolved := false

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

				if !resolved {
					if targets, ok := symbolsByName[baseName]; ok && len(targets) > 0 {
						for _, targetID := range targets {
							if targetID == sourceSymNodeID {
								continue
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

	// 5. Persist updated cache to disk
	if cache != nil {
		_ = cache.Save(repoPath)
	}

	stats.Duration = time.Since(start)
	return stats, nil
}
