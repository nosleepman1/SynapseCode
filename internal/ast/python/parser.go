package python

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
	pyImportRegex = regexp.MustCompile(`^(?:from\s+([a-zA-Z0-9_.]+)\s+import|import\s+([a-zA-Z0-9_.]+))`)
	pyDefRegex    = regexp.MustCompile(`^\s*(?:async\s+)?def\s+([a-zA-Z0-9_]+)\s*\((.*?)\)`)
	pyClassRegex  = regexp.MustCompile(`^\s*class\s+([a-zA-Z0-9_]+)(?:\((.*?)\))?:`)
)

// Parser implements ast.Parser for Python source files.
type Parser struct{}

// NewParser creates a new Python AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangPython
}

func (p *Parser) Extensions() []string {
	return []string{".py"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangPython,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	currentClass := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Imports
		if matches := pyImportRegex.FindStringSubmatch(line); len(matches) > 1 {
			imp := matches[1]
			if imp == "" && len(matches) > 2 {
				imp = matches[2]
			}
			if imp != "" {
				result.Imports = append(result.Imports, imp)
			}
		}

		// Classes
		if matches := pyClassRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentClass = matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, currentClass)),
				Name:          currentClass,
				QualifiedName: currentClass,
				Kind:          model.KindClass,
				Language:      model.LangPython,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  !strings.HasPrefix(currentClass, "_"),
			})
			continue
		}

		// Functions / Methods
		if matches := pyDefRegex.FindStringSubmatch(line); len(matches) > 1 {
			funcName := matches[1]
			kind := model.KindFunction
			qualifiedName := funcName

			// Indentation check: if indented under class, it's a method
			if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
				if currentClass != "" {
					kind = model.KindMethod
					qualifiedName = fmt.Sprintf("%s.%s", currentClass, funcName)
				}
			} else {
				currentClass = "" // Reset top-level class
			}

			sig := trimmed
			if strings.HasSuffix(sig, ":") {
				sig = strings.TrimSuffix(sig, ":")
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
				Name:          funcName,
				QualifiedName: qualifiedName,
				Kind:          kind,
				Language:      model.LangPython,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: sig,
				Exported:  !strings.HasPrefix(funcName, "_"),
			})
		}
	}

	return result, nil
}
