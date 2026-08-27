package context

import (
	"strings"
	"unicode/utf8"
)

// TokenEstimator provides fast, deterministic token counts for LLMs (BPE approximation: ~3.8 chars per token).
type TokenEstimator struct{}

// NewTokenEstimator creates a new token estimator.
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{}
}

// Count returns the estimated token count of a given text block.
func (e *TokenEstimator) Count(text string) int {
	charCount := utf8.RuneCountInString(text)
	if charCount == 0 {
		return 0
	}

	// Code typically has higher token-to-character ratio due to indentation and symbols
	words := len(strings.Fields(text))
	charBased := (charCount + 3) / 4

	if words > charBased {
		return words
	}
	return charBased
}
