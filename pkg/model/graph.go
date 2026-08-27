package model

// NodeID is the unique identifier for a vertex in the code graph.
type NodeID string

// NodeType indicates whether the node represents a file or an inner symbol.
type NodeType string

const (
	NodeFile   NodeType = "file"
	NodeSymbol NodeType = "symbol"
)

// EdgeType represents the directional relationship between two graph nodes.
type EdgeType string

const (
	EdgeCalls      EdgeType = "CALLS"
	EdgeImports    EdgeType = "IMPORTS"
	EdgeDefines    EdgeType = "DEFINES"
	EdgeImplements EdgeType = "IMPLEMENTS"
	EdgeExtends    EdgeType = "EXTENDS"
	EdgeReferences EdgeType = "REFERENCES"
	EdgeDependsOn  EdgeType = "DEPENDS_ON"
)

// Node represents a vertex in the SynapseCode graph.
type Node struct {
	ID        NodeID     `json:"id"`
	Type      NodeType   `json:"type"`
	FileID    FileID     `json:"file_id"`
	FilePath  string     `json:"file_path"`
	Symbol    *Symbol    `json:"symbol,omitempty"`
	Score     float64    `json:"score,omitempty"`
	TokenCost int        `json:"token_cost"`
	SkelCost  int        `json:"skel_cost"`
}

// Edge represents a directed relation from Source to Target.
type Edge struct {
	ID     string   `json:"id"`
	Source NodeID   `json:"source"`
	Target NodeID   `json:"target"`
	Type   EdgeType `json:"type"`
	Weight float64  `json:"weight"`
}

// GraphSummary provides high-level statistics about the indexed codebase.
type GraphSummary struct {
	TotalFiles   int `json:"total_files"`
	TotalSymbols int `json:"total_symbols"`
	TotalEdges   int `json:"total_edges"`
}
