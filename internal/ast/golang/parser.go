package golang

import (
	"bytes"
	"context"
	"fmt"
	goast "go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"strings"
	"unicode"

	"github.com/nosleepman1/synapse-code/internal/ast"
	"github.com/nosleepman1/synapse-code/pkg/model"
)

// Parser implements ast.Parser for Go source files using standard go/parser.
type Parser struct{}

// NewParser creates a new Go AST parser.
func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Language() model.Language {
	return model.LangGo
}

func (p *Parser) Extensions() []string {
	return []string{".go"}
}

func (p *Parser) Parse(ctx context.Context, fileID model.FileID, filePath string, content []byte) (*ast.ParsedFile, error) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, filePath, content, goparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", filePath, err)
	}

	result := &ast.ParsedFile{
		FileID:   fileID,
		FilePath: filePath,
		Language: model.LangGo,
		Symbols:  make([]model.Symbol, 0),
		Imports:  make([]string, 0),
	}

	// 1. Extract imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		result.Imports = append(result.Imports, path)
	}

	// 2. Extract declarations
	for _, decl := range file.Decls {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		switch d := decl.(type) {
		case *goast.FuncDecl:
			sym := p.extractFunc(fset, fileID, filePath, content, d)
			result.Symbols = append(result.Symbols, sym)

		case *goast.GenDecl:
			syms := p.extractGenDecl(fset, fileID, filePath, content, d)
			result.Symbols = append(result.Symbols, syms...)
		}
	}

	return result, nil
}

func (p *Parser) extractFunc(fset *token.FileSet, fileID model.FileID, filePath string, content []byte, fn *goast.FuncDecl) model.Symbol {
	startPos := fset.Position(fn.Pos())
	endPos := fset.Position(fn.End())

	name := fn.Name.Name
	kind := model.KindFunction
	qualifiedName := name

	// Receiver indicates method
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		kind = model.KindMethod
		recvType := formatNode(fset, fn.Recv.List[0].Type)
		qualifiedName = fmt.Sprintf("(%s).%s", recvType, name)
	}

	// Signature
	sigNode := &goast.FuncDecl{
		Doc:  fn.Doc,
		Recv: fn.Recv,
		Name: fn.Name,
		Type: fn.Type,
	}
	signature := formatNode(fset, sigNode)

	// Body extraction
	var body string
	if fn.Body != nil {
		bodyStart := fset.Position(fn.Body.Pos()).Offset
		bodyEnd := fset.Position(fn.Body.End()).Offset
		if bodyStart >= 0 && bodyEnd <= len(content) && bodyStart < bodyEnd {
			body = string(content[bodyStart:bodyEnd])
		}
	}

	// Calls inside function body
	var calls []string
	if fn.Body != nil {
		goast.Inspect(fn.Body, func(n goast.Node) bool {
			if call, ok := n.(*goast.CallExpr); ok {
				calledName := formatNode(fset, call.Fun)
				if calledName != "" {
					calls = append(calls, calledName)
				}
			}
			return true
		})
	}

	doc := ""
	if fn.Doc != nil {
		doc = fn.Doc.Text()
	}

	return model.Symbol{
		ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, qualifiedName)),
		Name:          name,
		QualifiedName: qualifiedName,
		Kind:          kind,
		Language:      model.LangGo,
		FileID:        fileID,
		Location: model.SourceLocation{
			FileID:      fileID,
			FilePath:    filePath,
			StartLine:   startPos.Line,
			StartColumn: startPos.Column,
			EndLine:     endPos.Line,
			EndColumn:   endPos.Column,
		},
		Signature:     signature,
		Documentation: doc,
		Exported:      unicode.IsUpper([]rune(name)[0]),
		Body:          body,
		Calls:         calls,
	}
}

func (p *Parser) extractGenDecl(fset *token.FileSet, fileID model.FileID, filePath string, content []byte, gen *goast.GenDecl) []model.Symbol {
	var symbols []model.Symbol

	for _, spec := range gen.Specs {
		switch s := spec.(type) {
		case *goast.TypeSpec:
			startPos := fset.Position(s.Pos())
			endPos := fset.Position(s.End())
			name := s.Name.Name

			kind := model.KindType
			switch s.Type.(type) {
			case *goast.StructType:
				kind = model.KindStruct
			case *goast.InterfaceType:
				kind = model.KindInterface
			}

			sig := fmt.Sprintf("type %s %s", name, formatNode(fset, s.Type))
			doc := ""
			if gen.Doc != nil {
				doc = gen.Doc.Text()
			} else if s.Doc != nil {
				doc = s.Doc.Text()
			}

			symbols = append(symbols, model.Symbol{
				ID:            model.SymbolID(fmt.Sprintf("%s::%s", filePath, name)),
				Name:          name,
				QualifiedName: name,
				Kind:          kind,
				Language:      model.LangGo,
				FileID:        fileID,
				Location: model.SourceLocation{
					FileID:      fileID,
					FilePath:    filePath,
					StartLine:   startPos.Line,
					StartColumn: startPos.Column,
					EndLine:     endPos.Line,
					EndColumn:   endPos.Column,
				},
				Signature:     sig,
				Documentation: doc,
				Exported:      unicode.IsUpper([]rune(name)[0]),
			})
		}
	}

	return symbols
}

func formatNode(fset *token.FileSet, node interface{}) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}
