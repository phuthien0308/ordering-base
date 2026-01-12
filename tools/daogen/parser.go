package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// FieldMetadata holds the extraction results for a struct field.
type FieldMetadata struct {
	FieldName    string
	FieldType    string
	ColumnName   string
	IsIdentifier bool // sql-identifier:"true"
	IsMandatory  bool // sql-insert:"true"
	IsUpdatable  bool // sql-update:"true"
	IsSkipped    bool // sql-skip:"true"
}

// StructMetadata holds the extraction results for the entire struct.
type StructMetadata struct {
	StructName string
	TableName  string
	Fields     []FieldMetadata
}

// Parser defines the interface for extracting metadata from Go structs.
type Parser interface {
	ParseStruct(ctx context.Context, filename, structName, tableName string) (*StructMetadata, error)
}

// ASTParser is a concrete implementation of Parser using go/ast.
type ASTParser struct{}

// ParseStruct parses the given Go file and searches for the specified struct using AST.
func (p *ASTParser) ParseStruct(ctx context.Context, filename, structName, tableName string) (*StructMetadata, error) {
	fset := token.NewFileSet()
	// Parse the file including comments
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var metadata *StructMetadata

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for type specifications
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		// Match struct name
		if ts.Name.Name != structName {
			return true
		}

		// Ensure it's a struct type
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		metadata = &StructMetadata{
			StructName: structName,
			TableName:  tableName,
		}

		// Default table name to snake_case struct name if not provided
		if metadata.TableName == "" {
			metadata.TableName = ToSnakeCase(structName)
		}

		// Iterate through fields
		for _, field := range st.Fields.List {
			// Skip anonymous/embedded fields for now unless requested
			if len(field.Names) == 0 {
				continue
			}

			// A field can have multiple names (e.g., A, B int)
			for _, name := range field.Names {
				fMeta := FieldMetadata{
					FieldName: name.Name,
				}

				// Extract Type as string
				fMeta.FieldType = extractTypeString(field.Type)

				// Parse Tags
				if field.Tag != nil {
					tagStr := strings.Trim(field.Tag.Value, "`")
					tags := reflect.StructTag(tagStr)

					fMeta.ColumnName = tags.Get("sql-col")
					fMeta.IsIdentifier = tags.Get("sql-identifier") == "true"
					fMeta.IsMandatory = tags.Get("sql-insert") == "true"
					fMeta.IsUpdatable = tags.Get("sql-update") == "true"
					fMeta.IsSkipped = tags.Get("sql-skip") == "true"
				}

				// Logic: if sql-col is missing, default to snake_case field name
				if fMeta.ColumnName == "" {
					fMeta.ColumnName = ToSnakeCase(name.Name)
				}

				// Only add if not explicitly skipped
				if !fMeta.IsSkipped {
					metadata.Fields = append(metadata.Fields, fMeta)
				}
			}
		}

		return false // Found the struct, stop inspecting
	})

	if metadata == nil {
		return nil, fmt.Errorf("struct %s not found in file %s", structName, filename)
	}

	return metadata, nil
}

// extractTypeString converts an ast.Expr to its string representation (simplified).
func extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + extractTypeString(t.X)
	case *ast.SelectorExpr:
		return extractTypeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + extractTypeString(t.Elt)
	default:
		return fmt.Sprintf("%T", expr) // Fallback for complex types
	}
}
