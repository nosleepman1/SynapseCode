package python

import (
	"context"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestPythonParser(t *testing.T) {
	code := `
from typing import Optional
import json

class DatabasePool:
    def __init__(self, dsn: str):
        self.dsn = dsn

    def query(self, sql: str) -> list:
        return []

def main():
    pool = DatabasePool("localhost")
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "db.py", "db.py", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse Python code: %v", err)
	}

	if len(parsed.Imports) < 2 {
		t.Errorf("expected at least 2 imports, got %v", parsed.Imports)
	}

	hasClass := false
	hasMethod := false
	hasFunction := false

	for _, s := range parsed.Symbols {
		if s.Name == "DatabasePool" && s.Kind == model.KindClass {
			hasClass = true
		}
		if s.Name == "query" && s.Kind == model.KindMethod {
			hasMethod = true
		}
		if s.Name == "main" && s.Kind == model.KindFunction {
			hasFunction = true
		}
	}

	if !hasClass {
		t.Errorf("expected DatabasePool class")
	}
	if !hasMethod {
		t.Errorf("expected query method under DatabasePool")
	}
	if !hasFunction {
		t.Errorf("expected main function")
	}
}
