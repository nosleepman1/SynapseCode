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
	importRegex    = regexp.MustCompile(`(?:import\s+.*?\s+from\s+['"]([^'"]+)['"]|require\(['"]([^'"]+)['"]\))`)
	funcRegex      = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s+([a-zA-Z0-9_$]+)\s*\((.*?)\)`)
	arrowFuncRegex = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([a-zA-Z0-9_$]+)\s*=\s*(?:async\s+)?(?:\((.*?)\)|[a-zA-Z0-9_$]+)\s*(?::\s*[^=]+)?\s*=>`)
	classRegex     = regexp.MustCompile(`^(?:export\s+)?(?:abstract\s+)?class\s+([a-zA-Z0-9_$]+)(?:\s+extends\s+([a-zA-Z0-9_$]+))?(?:\s+implements\s+([a-zA-Z0-9_$,\s]+))?`)
	interfaceRegex = regexp.MustCompile(`^(?:export\s+)?interface\s+([a-zA-Z0-9_$]+)(?:\s+extends\s+([a-zA-Z0-9_$,\s]+))?`)
	typeRegex      = regexp.MustCompile(`^(?:export\s+)?type\s+([a-zA-Z0-9_$]+)\s*=`)
	enumRegex      = regexp.MustCompile(`^(?:export\s+)?enum\s+([a-zA-Z0-9_$]+)`)
	methodRegex    = regexp.MustCompile(`^\s*(?:public|private|protected|static|async|\*)*\s*([a-zA-Z0-9_$]+)\s*\((.*?)\)(?::\s*[^;{]+)?\s*\{?`)
	callRegex      = regexp.MustCompile(`\b([a-zA-Z0-9_$]+(?:\.[a-zA-Z0-9_$]+)*)\s*\(`)
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
	currentClass := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || trimmed == "" {
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

		// Extract function calls in current line
		var lineCalls []string
		for _, cm := range callRegex.FindAllStringSubmatch(line, -1) {
			if len(cm) > 1 && cm[1] != "function" && cm[1] != "if" && cm[1] != "for" && cm[1] != "while" && cm[1] != "switch" {
				lineCalls = append(lineCalls, cm[1])
			}
		}

		// 1. Arrow Functions: const foo = () => ...
		if matches := arrowFuncRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			name := matches[1]
			sig := trimmed
			if idx := strings.Index(sig, "=>"); idx != -1 {
				sig = strings.TrimSpace(sig[:idx+2])
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
				Calls:     lineCalls,
			})
			continue
		}

		// 2. Standard Functions: function foo() ...
		if matches := funcRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
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
				Calls:     lineCalls,
			})
			continue
		}

		// 3. Classes
		if matches := classRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			name := matches[1]
			currentClass = name
			var implements []string
			if len(matches) > 2 && matches[2] != "" {
				implements = append(implements, matches[2])
			}
			if len(matches) > 3 && matches[3] != "" {
				for _, impl := range strings.Split(matches[3], ",") {
					impl = strings.TrimSpace(impl)
					if impl != "" {
						implements = append(implements, impl)
					}
				}
			}

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
				Signature:  trimmed,
				Exported:   strings.HasPrefix(trimmed, "export"),
				Implements: implements,
			})
			continue
		}

		// 4. Interfaces
		if matches := interfaceRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
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
			continue
		}

		// 5. Types
		if matches := typeRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
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
			continue
		}

		// 6. Enums
		if matches := enumRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
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
			continue
		}

		// 7. Class Methods (when inside class)
		if currentClass != "" && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			if matches := methodRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
				methodName := matches[1]
				if methodName != "if" && methodName != "for" && methodName != "switch" && methodName != "return" {
					qualifiedName := fmt.Sprintf("%s.%s", currentClass, methodName)
					sig := trimmed
					if idx := strings.Index(sig, "{"); idx != -1 {
						sig = strings.TrimSpace(sig[:idx])
					}

					result.Symbols = append(result.Symbols, model.Symbol{
						ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
						Name:          methodName,
						QualifiedName: qualifiedName,
						Kind:          model.KindMethod,
						Language:      model.LangTypeScript,
						FileID:        fileID,
						Location: model.SourceLocation{
							FileID:    fileID,
							FilePath:  filePath,
							StartLine: lineNum,
							EndLine:   lineNum,
						},
						Signature: sig,
						Exported:  !strings.HasPrefix(methodName, "_") && !strings.HasPrefix(methodName, "#"),
						Calls:     lineCalls,
					})
				}
			}
		} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			currentClass = "" // Reset class context on unindented line
		}
	}

	return result, nil
}
