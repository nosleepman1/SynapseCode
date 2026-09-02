package php

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
	namespaceRegex = regexp.MustCompile(`^namespace\s+([a-zA-Z0-9_\\]+);`)
	useRegex       = regexp.MustCompile(`^use\s+([a-zA-Z0-9_\\]+)(?:\s+as\s+([a-zA-Z0-9_$]+))?;`)
	traitUseRegex  = regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_$,\s\\]+);`)
	typeRegex      = regexp.MustCompile(`^(?:(?:abstract|final|readonly)\s+)*(class|interface|trait|enum)\s+([a-zA-Z0-9_$]+)(?:\s+extends\s+([a-zA-Z0-9_$\\]+))?(?:\s+implements\s+([a-zA-Z0-9_$,\s\\]+))?`)
	funcRegex      = regexp.MustCompile(`^(?:(?:public|protected|private|static|abstract|final)\s+)*function\s+([a-zA-Z0-9_$]+)\s*\((.*?)\)(?:\s*:\s*[^;{]+)?\s*\{?`)
	attributeRegex = regexp.MustCompile(`^#\[.*?\]`)
	callRegex      = regexp.MustCompile(`\b([a-zA-Z0-9_$]+(?:::[a-zA-Z0-9_$]+|->[a-zA-Z0-9_$]+)?)\s*\(`)
)

// Parser implements ast.Parser for PHP and Laravel source files.
type Parser struct{}

// NewParser creates a new PHP AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangPHP
}

func (p *Parser) Extensions() []string {
	return []string{".php", ".phtml"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangPHP,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	currentNamespace := ""
	currentClass := ""
	var pendingAttributes []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || trimmed == "" || trimmed == "<?php" || trimmed == "?>" {
			continue
		}

		// Namespace
		if matches := namespaceRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			currentNamespace = matches[1]
			continue
		}

		// Use / Imports (at top of file)
		if currentClass == "" && strings.HasPrefix(trimmed, "use ") {
			if matches := useRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
				result.Imports = append(result.Imports, matches[1])
				continue
			}
		}

		// PHP 8+ Attributes (e.g., #[Route("/api", methods: ["GET"])])
		if strings.HasPrefix(trimmed, "#[") {
			pendingAttributes = append(pendingAttributes, trimmed)
			continue
		}

		// Calls extraction
		var lineCalls []string
		for _, cm := range callRegex.FindAllStringSubmatch(line, -1) {
			if len(cm) > 1 && cm[1] != "if" && cm[1] != "for" && cm[1] != "foreach" && cm[1] != "while" && cm[1] != "switch" && cm[1] != "catch" && cm[1] != "array" && cm[1] != "isset" && cm[1] != "empty" {
				lineCalls = append(lineCalls, cm[1])
			}
		}

		// 1. Classes, Interfaces, Traits, Enums
		if matches := typeRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			kindStr := matches[1]
			name := matches[2]
			currentClass = name

			var kind model.SymbolKind
			switch kindStr {
			case "interface":
				kind = model.KindInterface
			case "trait":
				kind = model.KindTrait
			case "enum":
				kind = model.KindEnum
			default:
				kind = model.KindClass
			}

			var implements []string
			if len(matches) > 3 && matches[3] != "" {
				implements = append(implements, matches[3])
			}
			if len(matches) > 4 && matches[4] != "" {
				for _, impl := range strings.Split(matches[4], ",") {
					impl = strings.TrimSpace(impl)
					if impl != "" {
						implements = append(implements, impl)
					}
				}
			}

			sig := trimmed
			if idx := strings.Index(sig, "{"); idx != -1 {
				sig = strings.TrimSpace(sig[:idx])
			}
			if len(pendingAttributes) > 0 {
				sig = strings.Join(pendingAttributes, "\n") + "\n" + sig
				pendingAttributes = nil
			}

			qualifiedName := name
			if currentNamespace != "" {
				qualifiedName = fmt.Sprintf("%s\\%s", currentNamespace, name)
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: qualifiedName,
				Kind:          kind,
				Language:      model.LangPHP,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature:  sig,
				Exported:   true,
				Implements: implements,
				Calls:      lineCalls,
			})
			continue
		}

		// 2. Trait usages inside class (e.g., use HasFactory, Notifiable;)
		if currentClass != "" && strings.HasPrefix(trimmed, "use ") {
			if matches := traitUseRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
				for _, trait := range strings.Split(matches[1], ",") {
					trait = strings.TrimSpace(trait)
					if trait != "" {
						for i := range result.Symbols {
							if result.Symbols[i].Name == currentClass {
								result.Symbols[i].Implements = append(result.Symbols[i].Implements, trait)
								break
							}
						}
					}
				}
			}
			continue
		}

		// 3. Functions / Methods
		if matches := funcRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			methodName := matches[1]
			kind := model.KindFunction
			qualifiedName := methodName

			if currentClass != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "  ")) {
				kind = model.KindMethod
				qualifiedName = fmt.Sprintf("%s::%s", currentClass, methodName)
			} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				currentClass = "" // Reset class context if top-level function
			}

			sig := trimmed
			if idx := strings.Index(sig, "{"); idx != -1 {
				sig = strings.TrimSpace(sig[:idx])
			}
			if len(pendingAttributes) > 0 {
				sig = strings.Join(pendingAttributes, "\n") + "\n" + sig
				pendingAttributes = nil
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
				Name:          methodName,
				QualifiedName: qualifiedName,
				Kind:          kind,
				Language:      model.LangPHP,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: sig,
				Exported:  !strings.HasPrefix(methodName, "_") || methodName == "__construct",
				Calls:     lineCalls,
			})
			continue
		}

		// Reset attributes if unconsumed
		pendingAttributes = nil
	}

	return result, nil
}
