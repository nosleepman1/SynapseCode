package ast

import (
	"context"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// ParsedFile contains all extracted symbols, imports, and references from a single source file.
type ParsedFile struct {
	FileID     model.FileID    `json:"file_id"`
	FilePath   string          `json:"file_path"`
	Language   model.Language  `json:"language"`
	Symbols    []model.Symbol  `json:"symbols"`
	Imports    []string        `json:"imports"`
	References []string        `json:"references"`
}

// Parser defines the contract that language-specific AST analyzers must implement.
type Parser interface {
	Language() model.Language
	Extensions() []string
	Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ParsedFile, error)
}
