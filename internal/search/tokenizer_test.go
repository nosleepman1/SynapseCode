package search

import (
	"testing"
)

func TestTokenizeQuery(t *testing.T) {
	query := "Fix JWT token refresh in AuthController"
	tokens := TokenizeQuery(query)

	expectedKeywords := map[string]bool{
		"jwt":            true,
		"token":          true,
		"refresh":        true,
		"auth":           true,
		"controller":     true,
		"authcontroller": true,
	}

	for kw := range expectedKeywords {
		found := false
		for _, tok := range tokens {
			if tok == kw {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected token '%s' in extracted tokens: %v", kw, tokens)
		}
	}
}
