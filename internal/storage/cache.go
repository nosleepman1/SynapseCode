package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// CachedFileEntry represents the cached parsing results and verification metadata for a file.
type CachedFileEntry struct {
	RelPath      string         `json:"rel_path"`
	SizeBytes    int64          `json:"size_bytes"`
	ModTimeUnix  int64          `json:"mod_time_unix"`
	Checksum     string         `json:"checksum"`
	Symbols      []model.Symbol `json:"symbols"`
	Imports      []string       `json:"imports"`
}

// FileCache manages on-disk and in-memory persistence of parsed file entries.
type FileCache struct {
	mu      sync.RWMutex
	Version string                      `json:"version"`
	Entries map[string]*CachedFileEntry `json:"entries"`
}

// NewFileCache creates a new empty file cache container.
func NewFileCache() *FileCache {
	return &FileCache{
		Version: "1.0.0",
		Entries: make(map[string]*CachedFileEntry),
	}
}

// ComputeChecksum calculates the SHA-256 checksum of raw content bytes.
func ComputeChecksum(content []byte) string {
	hasher := sha256.New()
	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

// CacheFilePath returns the standard cache path: `<repoPath>/.synapse/cache.json`.
func CacheFilePath(repoPath string) string {
	return filepath.Join(repoPath, ".synapse", "cache.json")
}

// LoadCache reads and deserializes `.synapse/cache.json` if it exists.
func LoadCache(repoPath string) (*FileCache, error) {
	cachePath := CacheFilePath(repoPath)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewFileCache(), nil // Return fresh cache if none exists
		}
		return nil, fmt.Errorf("failed to read cache file %s: %w", cachePath, err)
	}

	var cache FileCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return NewFileCache(), nil // If corrupted, safely start with a fresh cache
	}

	if cache.Entries == nil {
		cache.Entries = make(map[string]*CachedFileEntry)
	}

	return &cache, nil
}

// Save writes the cache atomically to `.synapse/cache.json` using a temporary file.
func (fc *FileCache) Save(repoPath string) error {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	dir := filepath.Join(repoPath, ".synapse")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize file cache: %w", err)
	}

	targetPath := CacheFilePath(repoPath)
	tempPath := targetPath + ".tmp"

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary cache file %s: %w", tempPath, err)
	}

	// Atomic replace
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to atomically rename cache file to %s: %w", targetPath, err)
	}

	return nil
}

// IsFresh checks whether a cached entry matches given size, modtime, and content checksum.
func (fc *FileCache) IsFresh(relPath string, sizeBytes int64, modTimeUnix int64, checksum string) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	entry, exists := fc.Entries[relPath]
	if !exists {
		return false
	}

	// Quick check: size and modtime match
	if entry.SizeBytes == sizeBytes && entry.ModTimeUnix == modTimeUnix {
		return true
	}

	// Secondary check: checksum matches (handles git checkouts where mtime changes but content is identical)
	if checksum != "" && entry.Checksum == checksum {
		return true
	}

	return false
}

// Get retrieves a cached entry if available.
func (fc *FileCache) Get(relPath string) (*CachedFileEntry, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	entry, ok := fc.Entries[relPath]
	return entry, ok
}

// Put adds or replaces a cache entry.
func (fc *FileCache) Put(entry *CachedFileEntry) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.Entries == nil {
		fc.Entries = make(map[string]*CachedFileEntry)
	}
	fc.Entries[entry.RelPath] = entry
}

// Remove deletes an entry from the cache.
func (fc *FileCache) Remove(relPath string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	delete(fc.Entries, relPath)
}

// Len returns the count of cached entries.
func (fc *FileCache) Len() int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	return len(fc.Entries)
}
