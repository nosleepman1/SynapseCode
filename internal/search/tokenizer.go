package search

import (
	"regexp"
	"strings"
	"unicode"
)

var wordRegex = regexp.MustCompile(`[a-zA-Z0-9_$#@]+`)

// TokenizeQuery extracts clean, normalized keywords from a query, symbol name, or code snippet.
func TokenizeQuery(query string) []string {
	rawTokens := wordRegex.FindAllString(query, -1)
	var tokens []string
	seen := make(map[string]bool)

	addToken := func(t string) {
		t = strings.ToLower(strings.TrimSpace(t))
		t = strings.Trim(t, "_$#@.-/")
		if len(t) > 1 && !isStopWord(t) && !seen[t] {
			tokens = append(tokens, t)
			seen[t] = true

			// Also add light-stemmed version if distinct
			stemmed := StemWord(t)
			if len(stemmed) > 1 && stemmed != t && !isStopWord(stemmed) && !seen[stemmed] {
				tokens = append(tokens, stemmed)
				seen[stemmed] = true
			}
		}
	}

	for _, raw := range rawTokens {
		addToken(raw)

		// Split camelCase, PascalCase, snake_case, and acronyms
		subWords := SplitIdentifier(raw)
		for _, sw := range subWords {
			addToken(sw)
		}
	}

	return tokens
}

// SplitIdentifier splits code identifiers (e.g., 'JSONParser', 'user_id', 'getHTTPResponse', 'com.example.service').
func SplitIdentifier(s string) []string {
	var words []string
	var current strings.Builder

	runes := []rune(s)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]

		// Separators
		if r == '_' || r == '-' || r == '.' || r == '/' || r == '\\' || r == ':' || r == '#' || r == '$' || r == '@' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}

		if unicode.IsUpper(r) {
			// Case 1: Transition from lower/digit to Upper (e.g. user|Id, get2|D)
			if i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
				if current.Len() > 0 {
					words = append(words, current.String())
					current.Reset()
				}
			} else if i+1 < n && unicode.IsLower(runes[i+1]) && current.Len() > 1 {
				// Case 2: Transition from Acronym to Word (e.g. JSON|Parser -> 'JSON', 'Parser')
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

// StemWord applies lightweight suffix stripping for common English and code variations.
func StemWord(w string) string {
	lower := strings.ToLower(w)
	if len(lower) <= 3 {
		return lower
	}

	if strings.HasSuffix(lower, "ing") && len(lower) > 5 {
		return strings.TrimSuffix(lower, "ing")
	}
	if strings.HasSuffix(lower, "tion") && len(lower) > 6 {
		return strings.TrimSuffix(lower, "tion")
	}
	if strings.HasSuffix(lower, "tions") && len(lower) > 7 {
		return strings.TrimSuffix(lower, "tions")
	}
	if strings.HasSuffix(lower, "able") && len(lower) > 6 {
		return strings.TrimSuffix(lower, "able")
	}
	if strings.HasSuffix(lower, "ed") && len(lower) > 4 {
		return strings.TrimSuffix(lower, "ed")
	}
	if strings.HasSuffix(lower, "es") && len(lower) > 4 {
		return strings.TrimSuffix(lower, "es")
	}
	if strings.HasSuffix(lower, "s") && len(lower) > 3 && !strings.HasSuffix(lower, "ss") {
		return strings.TrimSuffix(lower, "s")
	}

	return lower
}

func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "a": true, "an": true, "in": true, "on": true, "at": true,
		"for": true, "to": true, "of": true, "and": true, "or": true, "with": true,
		"is": true, "are": true, "it": true, "this": true, "that": true, "fix": true,
		"how": true, "what": true, "where": true, "does": true, "do": true, "by": true,
		"from": true, "into": true, "all": true, "any": true, "some": true,
	}
	return stops[w]
}
