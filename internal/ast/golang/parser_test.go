package golang

import (
	"context"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestGoParser(t *testing.T) {
	code := `package auth

import "context"

type AuthClaims struct {
	UserID string
}

func ValidateToken(ctx context.Context, token string) (*AuthClaims, error) {
	return &AuthClaims{UserID: "123"}, nil
}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "auth.go", "auth.go", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse Go code: %v", err)
	}

	if len(parsed.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(parsed.Symbols))
	}

	foundStruct := false
	foundFunc := false

	for _, s := range parsed.Symbols {
		if s.Name == "AuthClaims" && s.Kind == model.KindStruct {
			foundStruct = true
		}
		if s.Name == "ValidateToken" && s.Kind == model.KindFunction {
			foundFunc = true
		}
	}

	if !foundStruct {
		t.Errorf("AuthClaims struct was not extracted")
	}
	if !foundFunc {
		t.Errorf("ValidateToken function was not extracted")
	}
}
