package model

// Language identifies a programming language supported by SynapseCode.
type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "typescript"
	LangJavaScript Language = "javascript"
	LangPython     Language = "python"
	LangRust       Language = "rust"
	LangJava       Language = "java"
	LangPHP        Language = "php"
	LangUnknown    Language = "unknown"
)

// SymbolKind represents the semantic category of a code entity.
type SymbolKind string

const (
	KindFunction  SymbolKind = "function"
	KindMethod    SymbolKind = "method"
	KindClass     SymbolKind = "class"
	KindStruct    SymbolKind = "struct"
	KindInterface SymbolKind = "interface"
	KindTrait     SymbolKind = "trait"
	KindRecord    SymbolKind = "record"
	KindEnum      SymbolKind = "enum"
	KindType      SymbolKind = "type"
	KindVariable  SymbolKind = "variable"
	KindConstant  SymbolKind = "constant"
	KindModule    SymbolKind = "module"
	KindPackage   SymbolKind = "package"
)

// SymbolID is a unique identifier for a symbol (e.g. "path/file.go::FunctionName").
type SymbolID string

// Symbol represents an extracted code entity with its signature, location, and metadata.
type Symbol struct {
	ID            SymbolID       `json:"id"`
	Name          string         `json:"name"`
	QualifiedName string         `json:"qualified_name"`
	Kind          SymbolKind     `json:"kind"`
	Language      Language       `json:"language"`
	FileID        FileID         `json:"file_id"`
	Location      SourceLocation `json:"location"`
	Signature     string         `json:"signature"`
	Documentation string         `json:"documentation,omitempty"`
	Exported      bool           `json:"exported"`
	Body          string         `json:"body,omitempty"`
	Calls         []string       `json:"calls,omitempty"`
	Imports       []string       `json:"imports,omitempty"`
	Implements    []string       `json:"implements,omitempty"`
}
