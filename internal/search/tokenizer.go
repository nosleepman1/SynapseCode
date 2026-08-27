package search

import (
	"regexp"
	"strings"
	"unicode"
)

var wordRegex = regexp.MustCompile(`[a-zA-Z0-9_]+`)

// TokenizeQuery extracts clean keywords from a natural language query or identifier.
func TokenizeQuery(query string) []string {
	rawTokens := wordRegex.FindAllString(query, -1)
	var tokens []string
	seen := make(map[string]bool)

	for _, raw := range rawTokens {
		lower := strings.ToLower(raw)
		if len(lower) > 1 && !isStopWord(lower) && !seen[lower] {
			tokens = append(tokens, lower)
			seen[lower] = true
		}

		// Split camelCase and snake_case
		subWords := splitIdentifier(raw)
		for _, sw := range subWords {
			swLower := strings.ToLower(sw)
			if len(swLower) > 1 && !isStopWord(swLower) && !seen[swLower] {
				tokens = append(tokens, swLower)
				seen[swLower] = true
			}
		}
	}

	return tokens
}

func splitIdentifier(s string) []string {
	var words []string
	var current strings.Builder

	runes := []rune(s)
	for i, r := range runes {
		if r == '_' || r == '-' || r == '.' || r == '/' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}

		if unicode.IsUpper(r) && i > 0 && unicode.IsLower(runes[i-1]) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "a": true, "an": true, "in": true, "on": true, "at": true,
		"for": true, "to": true, "of": true, "and": true, "or": true, "with": true,
		"is": true, "are": true, "it": true, "this": true, "that": true, "fix": true,
		"how": true, "what": true, "where": true, "does": true, "do": true,
	}
	return stops[w]
}
