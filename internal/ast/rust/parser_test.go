package rust

import (
	"context"
	"testing"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestRustParser(t *testing.T) {
	code := `
use std::collections::HashMap;

pub struct CodeGraph {
    nodes: HashMap<String, Node>,
}

impl CodeGraph {
    pub fn new() -> Self {
        Self { nodes: HashMap::new() }
    }

    pub fn add_node(&mut self, id: String) {
    }
}

pub trait Indexer {
    fn index_file(&self, path: &str);
}
`

	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), "graph.rs", "graph.rs", []byte(code))
	if err != nil {
		t.Fatalf("failed to parse Rust code: %v", err)
	}

	if len(parsed.Symbols) < 4 {
		t.Fatalf("expected at least 4 symbols, got %d", len(parsed.Symbols))
	}

	hasStruct := false
	hasTrait := false
	hasMethod := false

	for _, s := range parsed.Symbols {
		if s.Name == "CodeGraph" && s.Kind == model.KindStruct {
			hasStruct = true
		}
		if s.Name == "Indexer" && s.Kind == model.KindInterface {
			hasTrait = true
		}
		if s.Name == "new" && s.Kind == model.KindMethod {
			hasMethod = true
		}
	}

	if !hasStruct {
		t.Errorf("expected CodeGraph struct")
	}
	if !hasTrait {
		t.Errorf("expected Indexer trait")
	}
	if !hasMethod {
		t.Errorf("expected new method under CodeGraph")
	}
}
