package ast

import (
	"path/filepath"
	"strings"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// DetectLanguage identifies the programming language from a file extension.
func DetectLanguage(filePath string) model.Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return model.LangGo
	case ".ts", ".tsx":
		return model.LangTypeScript
	case ".js", ".jsx", ".mjs", ".cjs":
		return model.LangJavaScript
	case ".py":
		return model.LangPython
	case ".rs":
		return model.LangRust
	default:
		return model.LangUnknown
	}
}
