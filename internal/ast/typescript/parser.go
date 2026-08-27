package typescript

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

var (
	importRegex   = regexp.MustCompile(`(?:import\s+.*?\s+from\s+['"]([^'"]+)['"]|require\(['"]([^'"]+)['"]\))`)
	funcRegex     = regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z0-9_$]+)\s*\((.*?)\)`)
	classRegex    = regexp.MustCompile(`(?:export\s+)?(?:abstract\s+)?class\s+([a-zA-Z0-9_$]+)(?:\s+extends\s+([a-zA-Z0-9_$]+))?(?:\s+implements\s+([a-zA-Z0-9_$,\s]+))?`)
	interfaceRegex = regexp.MustCompile(`(?:export\s+)?interface\s+([a-zA-Z0-9_$]+)`)
	typeRegex     = regexp.MustCompile(`(?:export\s+)?type\s+([a-zA-Z0-9_$]+)\s*=`)
	callRegex     = regexp.MustCompile(`([a-zA-Z0-9_$]+(?:\.[a-zA-Z0-9_$]+)*)\s*\(`)
)

// Parser implements ast.Parser for TypeScript and JavaScript files.
type Parser struct{}

// NewParser creates a new TypeScript/JavaScript parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangTypeScript
}

func (p *Parser) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangTypeScript,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		// Imports
		if matches := importRegex.FindStringSubmatch(line); len(matches) > 1 {
			imp := matches[1]
			if imp == "" && len(matches) > 2 {
				imp = matches[2]
			}
			if imp != "" {
				result.Imports = append(result.Imports, imp)
			}
		}

		// Functions
		if matches := funcRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			sig := trimmed
			if idx := strings.Index(sig, "{"); idx != -1 {
				sig = strings.TrimSpace(sig[:idx])
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindFunction,
				Language:      model.LangTypeScript,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: sig,
				Exported:  strings.HasPrefix(trimmed, "export"),
			})
		}

		// Classes
		if matches := classRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindClass,
				Language:      model.LangTypeScript,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  strings.HasPrefix(trimmed, "export"),
			})
		}

		// Interfaces
		if matches := interfaceRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindInterface,
				Language:      model.LangTypeScript,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  strings.HasPrefix(trimmed, "export"),
			})
		}

		// Types
		if matches := typeRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindType,
				Language:      model.LangTypeScript,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  strings.HasPrefix(trimmed, "export"),
			})
		}
	}

	return result, nil
}
