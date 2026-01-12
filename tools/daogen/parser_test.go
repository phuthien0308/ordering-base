package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseStruct(t *testing.T) {
	// Create a temporary file with a test struct
	content := `package test
type TestUser struct {
	ID    int64  ` + "`" + `sql-col:"id" sql-identifier:"true"` + "`" + `
	Name  string ` + "`" + `sql-col:"name" sql-insert:"true"` + "`" + `
	Email string ` + "`" + `sql-col:"email" sql-update:"true"` + "`" + `
	Age   int    ` + "`" + `sql-skip:"true"` + "`" + `
    OtherField string
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "models.go")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var p Parser = &ASTParser{}
	metadata, err := p.ParseStruct(context.Background(), tmpFile, "TestUser", "users")
	if err != nil {
		t.Fatalf("ParseStruct failed: %v", err)
	}

	if metadata.StructName != "TestUser" {
		t.Errorf("Expected StructName TestUser, got %s", metadata.StructName)
	}

	if metadata.TableName != "users" {
		t.Errorf("Expected TableName users, got %s", metadata.TableName)
	}

	expectedFields := map[string]struct {
		col  string
		id   bool
		mand bool
		upd  bool
	}{
		"ID":         {"id", true, false, false},
		"Name":       {"name", false, true, false},
		"Email":      {"email", false, false, true},
		"OtherField": {"other_field", false, false, false},
	}

	foundFields := 0
	for _, f := range metadata.Fields {
		exp, ok := expectedFields[f.FieldName]
		if !ok {
			t.Errorf("Unexpected field found: %s", f.FieldName)
			continue
		}
		foundFields++
		if f.ColumnName != exp.col {
			t.Errorf("Field %s: expected column %s, got %s", f.FieldName, exp.col, f.ColumnName)
		}
		if f.IsIdentifier != exp.id {
			t.Errorf("Field %s: expected identifier %v, got %v", f.FieldName, exp.id, f.IsIdentifier)
		}
		if f.IsMandatory != exp.mand {
			t.Errorf("Field %s: expected mandatory %v, got %v", f.FieldName, exp.mand, f.IsMandatory)
		}
		if f.IsUpdatable != exp.upd {
			t.Errorf("Field %s: expected updatable %v, got %v", f.FieldName, exp.upd, f.IsUpdatable)
		}
	}

	if foundFields != len(expectedFields) {
		t.Errorf("Expected %d fields, found %d", len(expectedFields), foundFields)
	}
}

func TestDefaultNaming(t *testing.T) {
	content := `package test
type UserDetails struct {
	FirstName string
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "models.go")
	os.WriteFile(tmpFile, []byte(content), 0644)

	var p Parser = &ASTParser{}
	metadata, err := p.ParseStruct(context.Background(), tmpFile, "UserDetails", "")
	if err != nil {
		t.Fatal(err)
	}

	if metadata.TableName != "user_details" {
		t.Errorf("Expected snake_case table name user_details, got %s", metadata.TableName)
	}

	if len(metadata.Fields) != 1 || metadata.Fields[0].ColumnName != "first_name" {
		t.Errorf("Expected snake_case column name first_name, got %s", metadata.Fields[0].ColumnName)
	}
}
