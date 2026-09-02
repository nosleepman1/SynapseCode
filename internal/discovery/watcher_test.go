package discovery

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileWatcherDebounceAndIgnore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "synapse_watcher_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}

	testFile := filepath.Join(subDir, "service.go")
	if err := os.WriteFile(testFile, []byte("package pkg\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var callCount int32
	matcher := NewIgnoreMatcher(nil, 1024)

	watcher, err := NewFileWatcher(tempDir, matcher, 100*time.Millisecond, func(changedPaths []string) {
		atomic.AddInt32(&callCount, 1)
	})
	if err != nil {
		t.Fatalf("failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Start(ctx)

	// Wait for watcher to register
	time.Sleep(50 * time.Millisecond)

	// Perform 3 rapid writes
	for i := 0; i < 3; i++ {
		_ = os.WriteFile(testFile, []byte("package pkg\n// edit\n"), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce window (100ms) to flush
	time.Sleep(250 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	if count != 1 {
		t.Errorf("expected exactly 1 debounced callback execution, got %d", count)
	}
}
