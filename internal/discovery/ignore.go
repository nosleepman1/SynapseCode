package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

// IgnoreMatcher evaluates whether a file or directory should be excluded from indexing.
type IgnoreMatcher struct {
	excludedNames map[string]bool
	maxFileSize   int64
}

// NewIgnoreMatcher initializes the matcher with standard exclusion rules.
func NewIgnoreMatcher(customExcluded []string, maxFileSizeKB int64) *IgnoreMatcher {
	defaults := map[string]bool{
		".git":         true,
		".synapse":     true,
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"bin":          true,
		"build":        true,
		".next":        true,
		".cache":       true,
		"target":       true,
		".gradle":      true,
		".mvn":         true,
		".idea":        true,
		".vscode":      true,
		"coverage":     true,
		"storage":      true, // Laravel storage folder
	}

	for _, name := range customExcluded {
		defaults[strings.ToLower(name)] = true
	}

	return &IgnoreMatcher{
		excludedNames: defaults,
		maxFileSize:   maxFileSizeKB * 1024,
	}
}

// ShouldIgnore checks if the given path should be skipped.
func (m *IgnoreMatcher) ShouldIgnore(path string, info os.FileInfo) bool {
	name := info.Name()
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		return true
	}

	if m.excludedNames[strings.ToLower(name)] {
		return true
	}

	if !info.IsDir() && m.maxFileSize > 0 && info.Size() > m.maxFileSize {
		return true
	}

	// Filter binary extensions
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".exe", ".bin", ".dll", ".so", ".dylib", ".o", ".a", ".png", ".jpg", ".jpeg", ".gif", ".zip", ".tar", ".gz", ".pdf", ".lock":
		return true
	}

	return false
}
