package java

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
	pkgRegex        = regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+);`)
	importRegex     = regexp.MustCompile(`^import\s+(?:static\s+)?([a-zA-Z0-9_.*]+);`)
	typeRegex       = regexp.MustCompile(`^(?:(?:public|protected|private|abstract|static|final|sealed|non-sealed)\s+)*(class|interface|record|enum)\s+([a-zA-Z0-9_$]+)(?:<[^>]+>)?(?:\(.*?\))?(?:\s+extends\s+([a-zA-Z0-9_$,\s<>]+?))?(?:\s+implements\s+([a-zA-Z0-9_$,\s<>]+?))?(?:\s*\{|\s*$)`)
	methodRegex     = regexp.MustCompile(`^(?:(?:public|protected|private|static|final|synchronized|abstract|default|native)\s+)*(?:<[^>]+>\s+)?([a-zA-Z0-9_$<>,\[\]]+)\s+([a-zA-Z0-9_$]+)\s*\((.*?)\)(?:\s*throws\s+[a-zA-Z0-9_$,\s]+)?\s*\{?`)
	ctorRegex       = regexp.MustCompile(`^(?:(?:public|protected|private)\s+)?([a-zA-Z0-9_$]+)\s*\((.*?)\)(?:\s*throws\s+[a-zA-Z0-9_$,\s]+)?\s*\{?`)
	annotationRegex = regexp.MustCompile(`^@([a-zA-Z0-9_$]+)(?:\(.*?\))?`)
	callRegex       = regexp.MustCompile(`\b([a-zA-Z0-9_$]+(?:\.[a-zA-Z0-9_$]+)*)\s*\(`)
)

// Parser implements ast.Parser for Java source files.
type Parser struct{}

// NewParser creates a new Java AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangJava
}

func (p *Parser) Extensions() []string {
	return []string{".java"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangJava,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	currentPackage := ""
	currentClass := ""
	var pendingAnnotations []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || trimmed == "" {
			continue
		}

		// Package
		if matches := pkgRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			currentPackage = matches[1]
			continue
		}

		// Imports
		if matches := importRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			result.Imports = append(result.Imports, matches[1])
			continue
		}

		// Annotations (e.g., @RestController, @Service, @Autowired, @Override)
		if strings.HasPrefix(trimmed, "@") && annotationRegex.MatchString(trimmed) {
			pendingAnnotations = append(pendingAnnotations, trimmed)
			continue
		}

		// Function calls extraction
		var lineCalls []string
		for _, cm := range callRegex.FindAllStringSubmatch(line, -1) {
			if len(cm) > 1 && cm[1] != "if" && cm[1] != "for" && cm[1] != "while" && cm[1] != "switch" && cm[1] != "catch" && cm[1] != "synchronized" {
				lineCalls = append(lineCalls, cm[1])
			}
		}

		// 1. Classes, Interfaces, Records, Enums
		if matches := typeRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			kindStr := matches[1]
			name := matches[2]
			currentClass = name

			var kind model.SymbolKind
			switch kindStr {
			case "interface":
				kind = model.KindInterface
			case "enum":
				kind = model.KindEnum
			case "record":
				kind = model.KindRecord
			default:
				kind = model.KindClass
			}

			var implements []string
			if len(matches) > 3 && matches[3] != "" {
				for _, ext := range strings.Split(matches[3], ",") {
					ext = strings.TrimSpace(ext)
					if ext != "" {
						implements = append(implements, ext)
					}
				}
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
			if len(pendingAnnotations) > 0 {
				sig = strings.Join(pendingAnnotations, "\n") + "\n" + sig
				pendingAnnotations = nil
			}

			qualifiedName := name
			if currentPackage != "" {
				qualifiedName = fmt.Sprintf("%s.%s", currentPackage, name)
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: qualifiedName,
				Kind:          kind,
				Language:      model.LangJava,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature:  sig,
				Exported:   strings.HasPrefix(trimmed, "public"),
				Implements: implements,
				Calls:      lineCalls,
			})
			continue
		}

		// 2. Constructors
		if currentClass != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "  ")) {
			if matches := ctorRegex.FindStringSubmatch(trimmed); len(matches) > 1 && matches[1] == currentClass {
				name := matches[1]
				sig := trimmed
				if idx := strings.Index(sig, "{"); idx != -1 {
					sig = strings.TrimSpace(sig[:idx])
				}
				if len(pendingAnnotations) > 0 {
					sig = strings.Join(pendingAnnotations, "\n") + "\n" + sig
					pendingAnnotations = nil
				}

				qualifiedName := fmt.Sprintf("%s.%s", currentClass, name)
				result.Symbols = append(result.Symbols, model.Symbol{
					ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
					Name:          name,
					QualifiedName: qualifiedName,
					Kind:          model.KindMethod,
					Language:      model.LangJava,
					FileID:        fileID,
					Location: model.SourceLocation{
						FileID:    fileID,
						FilePath:  filePath,
						StartLine: lineNum,
						EndLine:   lineNum,
					},
					Signature: sig,
					Exported:  strings.HasPrefix(trimmed, "public"),
					Calls:     lineCalls,
				})
				continue
			}
		}

		// 3. Methods
		if matches := methodRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			retType := matches[1]
			methodName := matches[2]

			// Exclude control structures
			if methodName != "if" && methodName != "for" && methodName != "while" && methodName != "switch" && methodName != "catch" && retType != "return" && retType != "throw" && retType != "new" {
				sig := trimmed
				if idx := strings.Index(sig, "{"); idx != -1 {
					sig = strings.TrimSpace(sig[:idx])
				}
				if len(pendingAnnotations) > 0 {
					sig = strings.Join(pendingAnnotations, "\n") + "\n" + sig
					pendingAnnotations = nil
				}

				qualifiedName := methodName
				if currentClass != "" {
					qualifiedName = fmt.Sprintf("%s.%s", currentClass, methodName)
				}

				result.Symbols = append(result.Symbols, model.Symbol{
					ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
					Name:          methodName,
					QualifiedName: qualifiedName,
					Kind:          model.KindMethod,
					Language:      model.LangJava,
					FileID:        fileID,
					Location: model.SourceLocation{
						FileID:    fileID,
						FilePath:  filePath,
						StartLine: lineNum,
						EndLine:   lineNum,
					},
					Signature: sig,
					Exported:  strings.HasPrefix(trimmed, "public"),
					Calls:     lineCalls,
				})
				continue
			}
		}

		// Reset unconsumed annotations
		pendingAnnotations = nil
	}

	return result, nil
}
