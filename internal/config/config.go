package config

// Config holds the runtime configuration parameters for SynapseCode.
type Config struct {
	RepoPath         string   `json:"repo_path"`
	MaxFileSizeKB    int64    `json:"max_file_size_kb"`
	ExcludedDirs     []string `json:"excluded_dirs"`
	DefaultBudget    int      `json:"default_budget"`
	MaxTraversalHop  int      `json:"max_traversal_hop"`
	PageRankDamping  float64  `json:"pagerank_damping"`
	PageRankMaxIter  int      `json:"pagerank_max_iter"`
	LogLevel         string   `json:"log_level"`
}

// DefaultConfig returns production-ready default settings.
func DefaultConfig(repoPath string) *Config {
	return &Config{
		RepoPath:        repoPath,
		MaxFileSizeKB:   1024, // 1 MB limit
		ExcludedDirs:    []string{".git", "node_modules", "vendor", "dist", "bin", "build", ".next", ".cache", "target"},
		DefaultBudget:   3500,
		MaxTraversalHop: 2,
		PageRankDamping: 0.85,
		PageRankMaxIter: 20,
		LogLevel:        "info",
	}
}
