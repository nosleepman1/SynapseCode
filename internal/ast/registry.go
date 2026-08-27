package ast

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Registry manages and routes registered language parsers by extension.
type Registry struct {
	mu      sync.RWMutex
	parsers map[string]Parser
}

// NewRegistry initializes an empty AST parser registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]Parser),
	}
}

// Register adds a parser for all its supported file extensions.
func (r *Registry) Register(p Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, ext := range p.Extensions() {
		normalized := strings.ToLower(ext)
		if !strings.HasPrefix(normalized, ".") {
			normalized = "." + normalized
		}
		r.parsers[normalized] = p
	}
}

// ForFile returns the appropriate language parser for a given file path.
func (r *Registry) ForFile(filePath string) (Parser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(filePath))
	p, ok := r.parsers[ext]
	if !ok {
		return nil, fmt.Errorf("no AST parser registered for extension %s", ext)
	}
	return p, nil
}
