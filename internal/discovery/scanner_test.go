package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanner(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synapse-scan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "app.ts"), []byte("console.log('hi')"), 0644)

	// Create ignored directory
	nodeModules := filepath.Join(tempDir, "node_modules")
	_ = os.MkdirAll(nodeModules, 0755)
	_ = os.WriteFile(filepath.Join(nodeModules, "pkg.js"), []byte("// ignored"), 0644)

	matcher := NewIgnoreMatcher(nil, 1024)
	scanner := NewScanner(matcher)

	files, err := scanner.Scan(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected exactly 2 files (ignoring node_modules), got %d", len(files))
	}
}
