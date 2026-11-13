package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// DiscoveredFile contains metadata about a candidate source file discovered on disk.
type DiscoveredFile struct {
	ID       model.FileID
	Path     string
	RelPath  string
	Size     int64
	Language model.Language
}

// Scanner walks the repository filesystem and produces candidate source files.
type Scanner struct {
	matcher *IgnoreMatcher
}

// NewScanner creates a new directory scanner.
func NewScanner(matcher *IgnoreMatcher) *Scanner {
	return &Scanner{matcher: matcher}
}

// Scan walks the root directory and returns all valid source files.
func (s *Scanner) Scan(ctx context.Context, rootDir string) ([]DiscoveredFile, error) {
	var files []DiscoveredFile

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("invalid root path %s: %w", rootDir, err)
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip unreadable paths safely
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if s.matcher.ShouldIgnore(path, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			relPath = path
		}

		files = append(files, DiscoveredFile{
			ID:      model.FileID(filepath.ToSlash(relPath)),
			Path:    path,
			RelPath: filepath.ToSlash(relPath),
			Size:    info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("filesystem walk error: %w", err)
	}

	return files, nil
}
