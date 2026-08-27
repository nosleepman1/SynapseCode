package storage

import "time"

// IndexMetadata contains tracking and versioning information for the persisted code graph.
type IndexMetadata struct {
	Version      string    `json:"version"`
	IndexedAt    time.Time `json:"indexed_at"`
	RepoPath     string    `json:"repo_path"`
	FileCount    int       `json:"file_count"`
	SymbolCount  int       `json:"symbol_count"`
	EdgeCount    int       `json:"edge_count"`
	ContentHash  string    `json:"content_hash"`
}
