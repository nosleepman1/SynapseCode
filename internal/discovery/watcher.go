package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchCallback is invoked with a deduplicated list of modified file paths after debouncing.
type WatchCallback func(changedPaths []string)

// FileWatcher monitors the codebase filesystem for changes and triggers debounced updates.
type FileWatcher struct {
	rootDir         string
	matcher         *IgnoreMatcher
	watcher         *fsnotify.Watcher
	debounceWindow  time.Duration
	callback        WatchCallback
	mu              sync.Mutex
	pendingChanges  map[string]bool
	debounceTimer   *time.Timer
	stopChan        chan struct{}
	stopped         bool
}

// NewFileWatcher initializes a recursive file watcher with custom ignore filters and debounce delay.
func NewFileWatcher(rootDir string, matcher *IgnoreMatcher, debounceWindow time.Duration, callback WatchCallback) (*FileWatcher, error) {
	if debounceWindow <= 0 {
		debounceWindow = 250 * time.Millisecond
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		rootDir:        rootDir,
		matcher:        matcher,
		watcher:        fsWatcher,
		debounceWindow: debounceWindow,
		callback:       callback,
		pendingChanges: make(map[string]bool),
		stopChan:       make(chan struct{}),
	}

	if err := fw.addRecursive(rootDir); err != nil {
		_ = fsWatcher.Close()
		return nil, err
	}

	return fw, nil
}

func (fw *FileWatcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fw.matcher != nil && fw.matcher.ShouldIgnore(path, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch directory %s: %w", path, err)
			}
		}

		return nil
	})
}

// Start begins listening for filesystem events in the background until ctx is cancelled or Close is called.
func (fw *FileWatcher) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fw.Close()
			return

		case <-fw.stopChan:
			return

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			_ = err // Ignore transient FS read errors

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// If a new directory is created, automatically watch it
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if fw.matcher == nil || !fw.matcher.ShouldIgnore(event.Name, info) {
						_ = fw.addRecursive(event.Name)
					}
				}
			}

			// Filter ignored files and directories
			if info, err := os.Stat(event.Name); err == nil {
				if fw.matcher != nil && fw.matcher.ShouldIgnore(event.Name, info) {
					continue
				}
			}

			// Queue change and reset debounce timer
			fw.mu.Lock()
			fw.pendingChanges[event.Name] = true

			if fw.debounceTimer != nil {
				fw.debounceTimer.Stop()
			}

			fw.debounceTimer = time.AfterFunc(fw.debounceWindow, fw.flushChanges)
			fw.mu.Unlock()
		}
	}
}

func (fw *FileWatcher) flushChanges() {
	fw.mu.Lock()
	if len(fw.pendingChanges) == 0 {
		fw.mu.Unlock()
		return
	}

	changedList := make([]string, 0, len(fw.pendingChanges))
	for path := range fw.pendingChanges {
		changedList = append(changedList, path)
	}
	fw.pendingChanges = make(map[string]bool)
	fw.mu.Unlock()

	if fw.callback != nil {
		fw.callback(changedList)
	}
}

// Close gracefully terminates the watcher and releases OS resources.
func (fw *FileWatcher) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.stopped {
		return nil
	}
	fw.stopped = true

	close(fw.stopChan)
	if fw.debounceTimer != nil {
		fw.debounceTimer.Stop()
	}

	return fw.watcher.Close()
}
