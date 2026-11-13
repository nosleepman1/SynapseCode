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

func TestSplitIdentifierAndStemming(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "JSONParser",
			expected: []string{"JSON", "Parser"},
		},
		{
			input:    "getHTTPResponseCode",
			expected: []string{"get", "HTTP", "Response", "Code"},
		},
		{
			input:    "UserControllerInterface",
			expected: []string{"User", "Controller", "Interface"},
		},
		{
			input:    "MAX_BUFFER_SIZE",
			expected: []string{"MAX", "BUFFER", "SIZE"},
		},
		{
			input:    "com.example.service.AuthHandler",
			expected: []string{"com", "example", "service", "Auth", "Handler"},
		},
	}

	for _, tc := range tests {
		result := SplitIdentifier(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("SplitIdentifier(%s) length mismatch: got %v, expected %v", tc.input, result, tc.expected)
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("SplitIdentifier(%s)[%d] mismatch: got %s, expected %s", tc.input, i, result[i], tc.expected[i])
			}
		}
	}

	// Test Stemming
	if StemWord("authenticating") != "authenticat" {
		t.Errorf("expected StemWord(authenticating) == authenticat, got %s", StemWord("authenticating"))
	}
	if StemWord("controllers") != "controller" {
		t.Errorf("expected StemWord(controllers) == controller, got %s", StemWord("controllers"))
	}
	if StemWord("validation") != "valida" {
		t.Errorf("expected StemWord(validation) == valida, got %s", StemWord("validation"))
	}
}
