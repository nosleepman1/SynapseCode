package model

// ContextMode indicates whether a piece of context contains the full implementation or just the skeleton.
type ContextMode string

const (
	ContextFullCode ContextMode = "full"
	ContextSkeleton ContextMode = "skeleton"
)

// ContextItem represents a single extracted piece of code in the final LLM context.
type ContextItem struct {
	FilePath   string      `json:"file_path"`
	SymbolName string      `json:"symbol_name"`
	Kind       SymbolKind  `json:"kind"`
	Mode       ContextMode `json:"mode"`
	Content    string      `json:"content"`
	TokenCount int         `json:"token_count"`
	Relevance  float64     `json:"relevance"`
	Reason     string      `json:"reason"`
}

// ContextPack represents the assembled, token-budgeted payload ready for the LLM.
type ContextPack struct {
	TaskDescription string        `json:"task_description"`
	BudgetTokens    int           `json:"budget_tokens"`
	UsedTokens      int           `json:"used_tokens"`
	Items           []ContextItem `json:"items"`
	RepoSummary     string        `json:"repo_summary,omitempty"`
	FormattedText   string        `json:"formatted_text"`
}
