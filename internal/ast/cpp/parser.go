package cpp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

var (
	includeRegex   = regexp.MustCompile(`^#include\s+[<"]([^>"]+)[>"]`)
	namespaceRegex = regexp.MustCompile(`^namespace\s+([a-zA-Z0-9_:]+)\s*\{?`)
	typeRegex      = regexp.MustCompile(`^(?:(?:template\s*<.*?>\s+)?(?:alignas\(.*?\)\s+)?)*(class|struct|union|enum(?:\s+class)?)\s+([a-zA-Z0-9_$]+)(?:\s*:\s*([a-zA-Z0-9_$,\s<>:]+?))?(?:\s*\{|\s*$)`)
	funcRegex      = regexp.MustCompile(`^(?:(?:virtual|static|inline|explicit|constexpr|friend)\s+)*(?:[a-zA-Z0-9_$<>,*&:]+\s+)+([a-zA-Z0-9_$~]+)\s*\((.*?)\)(?:\s*const)?(?:\s*override)?(?:\s*final)?(?:\s*noexcept)?(?:\s*=\s*(?:0|default|delete))?\s*\{?`)
	callRegex      = regexp.MustCompile(`\b([a-zA-Z0-9_$]+(?:::[a-zA-Z0-9_$]+|\.[a-zA-Z0-9_$]+|->[a-zA-Z0-9_$]+)?)\s*\(`)
)

// Parser implements ast.Parser for C and C++ source and header files.
type Parser struct{}

// NewParser creates a new C/C++ AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangCpp
}

func (p *Parser) Extensions() []string {
	return []string{".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	lang := model.LangCpp
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".c" || ext == ".h" {
		lang = model.LangC
	}

	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: lang,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	currentNamespace := ""
	currentClass := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || trimmed == "" {
			continue
		}

		// #include directives
		if matches := includeRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			result.Imports = append(result.Imports, matches[1])
			continue
		}

		// Namespace
		if matches := namespaceRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			currentNamespace = matches[1]
			continue
		}

		// Function/Method calls extraction
		var lineCalls []string
		for _, cm := range callRegex.FindAllStringSubmatch(line, -1) {
			if len(cm) > 1 && cm[1] != "if" && cm[1] != "for" && cm[1] != "while" && cm[1] != "switch" && cm[1] != "catch" && cm[1] != "sizeof" && cm[1] != "typeof" && cm[1] != "alignas" {
				lineCalls = append(lineCalls, cm[1])
			}
		}

		// 1. Classes, Structs, Enums, Unions
		if matches := typeRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			kindStr := matches[1]
			name := matches[2]
			currentClass = name

			var kind model.SymbolKind
			switch {
			case kindStr == "struct":
				kind = model.KindStruct
			case strings.HasPrefix(kindStr, "enum"):
				kind = model.KindEnum
			default:
				kind = model.KindClass
			}

			var implements []string
			if len(matches) > 3 && matches[3] != "" {
				for _, base := range strings.Split(matches[3], ",") {
					base = strings.TrimSpace(base)
					base = strings.TrimPrefix(base, "public ")
					base = strings.TrimPrefix(base, "protected ")
					base = strings.TrimPrefix(base, "private ")
					base = strings.TrimPrefix(base, "virtual ")
					base = strings.TrimSpace(base)
					if base != "" {
						implements = append(implements, base)
					}
				}
			}

			sig := trimmed
			if idx := strings.Index(sig, "{"); idx != -1 {
				sig = strings.TrimSpace(sig[:idx])
			}

			qualifiedName := name
			if currentNamespace != "" {
				qualifiedName = fmt.Sprintf("%s::%s", currentNamespace, name)
			}

			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: qualifiedName,
				Kind:          kind,
				Language:      lang,
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

		// 2. Functions & Methods
		if matches := funcRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			funcName := matches[1]

			if funcName != "if" && funcName != "for" && funcName != "while" && funcName != "switch" && funcName != "return" && funcName != "throw" && funcName != "delete" && funcName != "new" && !strings.HasPrefix(trimmed, "#") {
				kind := model.KindFunction
				qualifiedName := funcName

				if currentClass != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "  ")) {
					kind = model.KindMethod
					qualifiedName = fmt.Sprintf("%s::%s", currentClass, funcName)
				} else if currentNamespace != "" {
					qualifiedName = fmt.Sprintf("%s::%s", currentNamespace, funcName)
				}

				sig := trimmed
				if idx := strings.Index(sig, "{"); idx != -1 {
					sig = strings.TrimSpace(sig[:idx])
				}

				result.Symbols = append(result.Symbols, model.Symbol{
					ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
					Name:          funcName,
					QualifiedName: qualifiedName,
					Kind:          kind,
					Language:      lang,
					FileID:        fileID,
					Location: model.SourceLocation{
						FileID:    fileID,
						FilePath:  filePath,
						StartLine: lineNum,
						EndLine:   lineNum,
					},
					Signature: sig,
					Exported:  true,
					Calls:     lineCalls,
				})
				continue
			}
		}
	}

	return result, nil
}
