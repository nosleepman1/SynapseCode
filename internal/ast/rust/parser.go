package rust

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
	useRegex    = regexp.MustCompile(`^\s*(?:pub\s+)?use\s+([^;]+);`)
	fnRegex     = regexp.MustCompile(`^\s*(?:pub(?:\(.*?\))?\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+([a-zA-Z0-9_]+)\s*(?:<.*?>)?\s*\((.*?)\)`)
	structRegex = regexp.MustCompile(`^\s*(?:pub(?:\(.*?\))?\s+)?struct\s+([a-zA-Z0-9_]+)`)
	enumRegex   = regexp.MustCompile(`^\s*(?:pub(?:\(.*?\))?\s+)?enum\s+([a-zA-Z0-9_]+)`)
	traitRegex  = regexp.MustCompile(`^\s*(?:pub(?:\(.*?\))?\s+)?trait\s+([a-zA-Z0-9_]+)`)
	implRegex   = regexp.MustCompile(`^\s*impl(?:\s*<.*?>)?\s+(?:([a-zA-Z0-9_]+)\s+for\s+)?([a-zA-Z0-9_]+)`)
)

// Parser implements ast.Parser for Rust source files.
type Parser struct{}

// NewParser creates a new Rust AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangRust
}

func (p *Parser) Extensions() []string {
	return []string{".rs"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangRust,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	currentImpl := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		// Imports / Use statements
		if matches := useRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Imports = append(result.Imports, strings.TrimSpace(matches[1]))
		}

		// Impl blocks
		if matches := implRegex.FindStringSubmatch(line); len(matches) > 2 {
			currentImpl = matches[2]
			if currentImpl == "" && len(matches) > 1 {
				currentImpl = matches[1]
			}
			continue
		}

		// Structs
		if matches := structRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindStruct,
				Language:      model.LangRust,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  strings.HasPrefix(trimmed, "pub"),
			})
			continue
		}

		// Enums
		if matches := enumRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindType,
				Language:      model.LangRust,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  strings.HasPrefix(trimmed, "pub"),
			})
			continue
		}

		// Traits
		if matches := traitRegex.FindStringSubmatch(line); len(matches) > 1 {
			name := matches[1]
			result.Symbols = append(result.Symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          model.KindInterface,
				Language:      model.LangRust,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: trimmed,
				Exported:  strings.HasPrefix(trimmed, "pub"),
			})
			continue
		}

		// Functions / Methods
		if matches := fnRegex.FindStringSubmatch(line); len(matches) > 1 {
			funcName := matches[1]
			kind := model.KindFunction
			qualifiedName := funcName

			if currentImpl != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
				kind = model.KindMethod
				qualifiedName = fmt.Sprintf("%s::%s", currentImpl, funcName)
			} else {
				currentImpl = ""
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
				Language:      model.LangRust,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:    fileID,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum,
				},
				Signature: sig,
				Exported:  strings.HasPrefix(trimmed, "pub"),
			})
		}
	}

	return result, nil
}
