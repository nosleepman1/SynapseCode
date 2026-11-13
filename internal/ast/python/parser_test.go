package python

import (
	"context"
	"strings"
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

@dataclass
class PostgresPool(DatabasePool):
    @property
    def is_connected(self) -> bool:
        return True

    @classmethod
    async def create_pool(cls, host: str):
        connect(host)

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
	hasPostgresClass := false
	hasMethod := false
	hasAsyncMethod := false
	hasProperty := false
	hasFunction := false

	for _, s := range parsed.Symbols {
		if s.Name == "DatabasePool" && s.Kind == model.KindClass {
			hasClass = true
		}
		if s.Name == "PostgresPool" && s.Kind == model.KindClass {
			hasPostgresClass = true
			if len(s.Implements) == 0 || s.Implements[0] != "DatabasePool" {
				t.Errorf("expected PostgresPool to inherit from DatabasePool, got: %v", s.Implements)
			}
			if !strings.Contains(s.Signature, "@dataclass") {
				t.Errorf("expected @dataclass decorator in signature, got: %s", s.Signature)
			}
		}
		if s.Name == "query" && s.Kind == model.KindMethod {
			hasMethod = true
		}
		if s.Name == "is_connected" && s.Kind == model.KindMethod {
			hasProperty = true
			if !strings.Contains(s.Signature, "@property") {
				t.Errorf("expected @property decorator in signature, got: %s", s.Signature)
			}
		}
		if s.Name == "create_pool" && s.Kind == model.KindMethod {
			hasAsyncMethod = true
		}
		if s.Name == "main" && s.Kind == model.KindFunction {
			hasFunction = true
		}
	}

	if !hasClass {
		t.Errorf("expected DatabasePool class")
	}
	if !hasPostgresClass {
		t.Errorf("expected PostgresPool class")
	}
	if !hasMethod {
		t.Errorf("expected query method under DatabasePool")
	}
	if !hasProperty {
		t.Errorf("expected is_connected property")
	}
	if !hasAsyncMethod {
		t.Errorf("expected create_pool async method")
	}
	if !hasFunction {
		t.Errorf("expected main function")
	}
}
