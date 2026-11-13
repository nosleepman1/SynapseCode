package storage

import (
	"os"
	"testing"
	"time"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

func TestFileCacheOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synapse_cache_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewFileCache()
	if cache.Len() != 0 {
		t.Errorf("expected empty cache, got %d", cache.Len())
	}

	sym := model.Symbol{
		Name:      "CalculateHash",
		Kind:      model.KindFunction,
		Signature: "func CalculateHash(data []byte) string",
	}

	content := []byte("func CalculateHash(data []byte) string { return \"\" }")
	checksum := ComputeChecksum(content)
	now := time.Now().UnixNano()

	entry := &CachedFileEntry{
		RelPath:     "pkg/hash/hash.go",
		SizeBytes:   int64(len(content)),
		ModTimeUnix: now,
		Checksum:    checksum,
		Symbols:     []model.Symbol{sym},
		Imports:     []string{"crypto/sha256"},
	}

	cache.Put(entry)
	if cache.Len() != 1 {
		t.Errorf("expected 1 entry, got %d", cache.Len())
	}

	// Test IsFresh
	if !cache.IsFresh("pkg/hash/hash.go", int64(len(content)), now, checksum) {
		t.Errorf("expected cache entry to be fresh")
	}

	// Test Invalidation on different size
	if cache.IsFresh("pkg/hash/hash.go", 999, now, "different_checksum") {
		t.Errorf("expected cache entry to be stale due to size/checksum mismatch")
	}

	// Test Save & Load
	if err := cache.Save(tempDir); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	loaded, err := LoadCache(tempDir)
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	if loaded.Len() != 1 {
		t.Fatalf("expected 1 loaded entry, got %d", loaded.Len())
	}

	loadedEntry, ok := loaded.Get("pkg/hash/hash.go")
	if !ok {
		t.Fatalf("expected to get pkg/hash/hash.go from loaded cache")
	}
	if len(loadedEntry.Symbols) != 1 || loadedEntry.Symbols[0].Name != "CalculateHash" {
		t.Errorf("loaded symbol mismatch: %+v", loadedEntry.Symbols)
	}

	// Test Remove
	cache.Remove("pkg/hash/hash.go")
	if cache.Len() != 0 {
		t.Errorf("expected 0 entries after removal, got %d", cache.Len())
	}
}
