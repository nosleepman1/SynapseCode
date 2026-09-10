package csharp

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
	nsBlockRegex   = regexp.MustCompile(`^namespace\s+([a-zA-Z0-9_.]+)`)
	nsFileRegex    = regexp.MustCompile(`^namespace\s+([a-zA-Z0-9_.]+);`)
	usingRegex     = regexp.MustCompile(`^using\s+(?:static\s+)?([a-zA-Z0-9_.]+);`)
	attrRegex      = regexp.MustCompile(`^\[.*?\]`)
	typeRegex      = regexp.MustCompile(`^(?:(?:public|protected|private|internal|abstract|static|sealed|readonly|partial)\s+)*(class|interface|struct|record(?:\s+struct|\s+class)?|enum)\s+([a-zA-Z0-9_$]+)(?:<[^>]+>)?(?:\(.*?\))?(?:\s*:\s*([a-zA-Z0-9_$,\s<>]+?))?(?:\s*where\s+.*?)?(?:\s*\{|\s*;|\s*$)`)
	methodRegex    = regexp.MustCompile(`^(?:(?:public|protected|private|internal|static|async|override|virtual|abstract|sealed|extern)\s+)*(?:(?:Task|ValueTask|[a-zA-Z0-9_$<>,\[\]\?]+)\s+)+([a-zA-Z0-9_$]+)(?:<[^>]+>)?\s*\((.*?)\)(?:\s*where\s+.*?)?\s*\{?`)
	ctorRegex      = regexp.MustCompile(`^(?:(?:public|protected|private|internal|static)\s+)?([a-zA-Z0-9_$]+)\s*\((.*?)\)(?:\s*:\s*(?:base|this)\(.*?\))?\s*\{?`)
	callRegex      = regexp.MustCompile(`\b([a-zA-Z0-9_$]+(?:\.[a-zA-Z0-9_$]+)*)\s*\(`)
)

// Parser implements ast.Parser for C# and .NET / ASP.NET Core source files.
type Parser struct{}

// NewParser creates a new C# AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangCSharp
}

func (p *Parser) Extensions() []string {
	return []string{".cs"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangCSharp,
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

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || trimmed == "" {
			continue
		}

		// File-scoped or block namespace
		if matches := nsFileRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			currentNamespace = matches[1]
			continue
		}
		if matches := nsBlockRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			currentNamespace = matches[1]
			continue
		}

		// Using directives
		if matches := usingRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			result.Imports = append(result.Imports, matches[1])
			continue
		}

		// Attributes (e.g. [ApiController], [Route("api/[controller]")], [HttpGet])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") && attrRegex.MatchString(trimmed) {
			pendingAttributes = append(pendingAttributes, trimmed)
			continue
		}

		// Method calls extraction
		var lineCalls []string
		for _, cm := range callRegex.FindAllStringSubmatch(line, -1) {
			if len(cm) > 1 && cm[1] != "if" && cm[1] != "for" && cm[1] != "foreach" && cm[1] != "while" && cm[1] != "switch" && cm[1] != "catch" && cm[1] != "lock" && cm[1] != "typeof" && cm[1] != "sizeof" {
				lineCalls = append(lineCalls, cm[1])
			}
		}

		// 1. Classes, Interfaces, Structs, Records, Enums
		if matches := typeRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			kindStr := matches[1]
			name := matches[2]
			currentClass = name

			var kind model.SymbolKind
			switch {
			case kindStr == "interface":
				kind = model.KindInterface
			case kindStr == "struct":
				kind = model.KindStruct
			case kindStr == "enum":
				kind = model.KindEnum
			case strings.HasPrefix(kindStr, "record"):
				kind = model.KindRecord
			default:
				kind = model.KindClass
			}

			var implements []string
			if len(matches) > 3 && matches[3] != "" {
				for _, impl := range strings.Split(matches[3], ",") {
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
			sig = strings.TrimSuffix(sig, ";")
			sig = strings.TrimSpace(sig)
			if len(pendingAttributes) > 0 {
				sig = strings.Join(pendingAttributes, "\n") + "\n" + sig
				pendingAttributes = nil
			}

			qualifiedName := name
			if currentNamespace != "" {
				qualifiedName = fmt.Sprintf("%s.%s", currentNamespace, name)
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: qualifiedName,
				Kind:          kind,
				Language:      model.LangCSharp,
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
				if len(pendingAttributes) > 0 {
					sig = strings.Join(pendingAttributes, "\n") + "\n" + sig
					pendingAttributes = nil
				}

				qualifiedName := fmt.Sprintf("%s.%s", currentClass, name)
				result.Symbols = append(result.Symbols, model.Symbol{
					ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
					Name:          name,
					QualifiedName: qualifiedName,
					Kind:          model.KindMethod,
					Language:      model.LangCSharp,
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
		if matches := methodRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			methodName := matches[1]

			if methodName != "if" && methodName != "for" && methodName != "foreach" && methodName != "while" && methodName != "switch" && methodName != "return" && methodName != "throw" && methodName != "new" && methodName != "await" && methodName != "get" && methodName != "set" {
				sig := trimmed
				if idx := strings.Index(sig, "{"); idx != -1 {
					sig = strings.TrimSpace(sig[:idx])
				}
				if len(pendingAttributes) > 0 {
					sig = strings.Join(pendingAttributes, "\n") + "\n" + sig
					pendingAttributes = nil
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
					Language:      model.LangCSharp,
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

		// Reset unconsumed attributes
		pendingAttributes = nil
	}

	return result, nil
}
