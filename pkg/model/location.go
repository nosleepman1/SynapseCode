package model

import "fmt"

// FileID represents a unique, deterministic identifier for a file in the repository.
type FileID string

// SourceLocation represents a 1-based line/column position in a source file.
type SourceLocation struct {
	FileID      FileID `json:"file_id"`
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	StartColumn int    `json:"start_column"`
	EndLine     int    `json:"end_line"`
	EndColumn   int    `json:"end_column"`
}

// String returns a human-readable string representation of the source location.
func (l SourceLocation) String() string {
	return fmt.Sprintf("%s:%d:%d-%d:%d", l.FilePath, l.StartLine, l.StartColumn, l.EndLine, l.EndColumn)
}

// IsValid checks whether the source location has valid positive line bounds.
func (l SourceLocation) IsValid() bool {
	return l.StartLine > 0 && l.EndLine >= l.StartLine
}
