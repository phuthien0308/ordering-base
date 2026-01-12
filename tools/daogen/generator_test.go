package main

import (
	"context"
	"strings"
	"testing"
)

func TestGenerator(t *testing.T) {
	metadata := &StructMetadata{
		StructName: "TestUser",
		TableName:  "users",
		Fields: []FieldMetadata{
			{FieldName: "ID", FieldType: "int64", ColumnName: "id", IsIdentifier: true},
			{FieldName: "Name", FieldType: "string", ColumnName: "name", IsMandatory: true},
			{FieldName: "Email", FieldType: "string", ColumnName: "email", IsUpdatable: true},
		},
	}

	g := &TemplateGenerator{}
	content, err := g.Generate(context.Background(), "models", metadata, "mysql")
	if err != nil {
		t.Fatal(err)
	}

	code := string(content)

	// Check package and struct
	if !strings.Contains(code, "package models") {
		t.Error("Generated code missing package name")
	}
	if !strings.Contains(code, "type TestUserDAO struct") {
		t.Error("Generated code missing TestUserDAO struct")
	}

	// Check method signatures
	if !strings.Contains(code, "func (dao *TestUserDAO) Get(ctx context.Context, id int64) (*TestUser, error)") {
		t.Errorf("Generated code missing or incorrect Get method signature. Code:\n%s", code)
	}
	if !strings.Contains(code, "func (dao *TestUserDAO) Insert(ctx context.Context, m *TestUser) error") {
		t.Error("Generated code missing Insert method")
	}
	if !strings.Contains(code, "func (dao *TestUserDAO) Update(ctx context.Context, m *TestUser) error") {
		t.Error("Generated code missing Update method")
	}

	// Check SQL queries (MySQL style)
	if !strings.Contains(code, "SELECT id, name, email FROM users WHERE id = ?") {
		t.Error("Incorrect SELECT query generated for mysql")
	}
	if !strings.Contains(code, "INSERT INTO users (name, email) VALUES (?, ?)") {
		t.Error("Incorrect INSERT query generated for mysql")
	}
	if !strings.Contains(code, "UPDATE users SET email = ? WHERE id = ?") {
		t.Error("Incorrect UPDATE query generated for mysql")
	}
}

func TestGeneratorWithExclusions(t *testing.T) {
	metadata := &StructMetadata{
		StructName: "Order",
		TableName:  "orders",
		Fields: []FieldMetadata{
			{FieldName: "ID", FieldType: "int64", ColumnName: "id", IsIdentifier: true},
			{FieldName: "Code", FieldType: "string", ColumnName: "code", IsMandatory: true, IsUpdatable: false},
			{FieldName: "Status", FieldType: "string", ColumnName: "status", IsUpdatable: true},
		},
	}

	g := &TemplateGenerator{}
	content, err := g.Generate(context.Background(), "models", metadata, "mysql")
	if err != nil {
		t.Fatal(err)
	}

	code := string(content)

	// Verify that 'code' is NOT in the SET clause, but 'status' IS
	if !strings.Contains(code, "UPDATE orders SET status = ? WHERE id = ?") {
		t.Errorf("Expected status in SET clause, but not found. Code:\n%s", code)
	}
	if strings.Contains(code, "code = ?") {
		t.Errorf("Field 'code' should be excluded from UPDATE statement as IsUpdatable=false. Code:\n%s", code)
	}
}

func TestPostgresDialect(t *testing.T) {
	metadata := &StructMetadata{
		StructName: "TestUser",
		TableName:  "users",
		Fields: []FieldMetadata{
			{FieldName: "ID", FieldType: "int64", ColumnName: "id", IsIdentifier: true},
			{FieldName: "Name", FieldType: "string", ColumnName: "name", IsMandatory: true, IsUpdatable: true},
			{FieldName: "Email", FieldType: "string", ColumnName: "email", IsUpdatable: true},
		},
	}

	g := &TemplateGenerator{}
	content, err := g.Generate(context.Background(), "models", metadata, "postgres")
	if err != nil {
		t.Fatal(err)
	}

	code := string(content)

	// Check placeholders (Postgres style: $1, $2, ...)
	if !strings.Contains(code, "SELECT id, name, email FROM users WHERE id = $1") {
		t.Errorf("Incorrect SELECT query for postgres. Code:\n%s", code)
	}
	if !strings.Contains(code, "INSERT INTO users (name, email) VALUES ($1, $2)") {
		t.Errorf("Incorrect INSERT query for postgres. Code:\n%s", code)
	}
	// Update has SET name=$1, email=$2 WHERE id=$3
	if !strings.Contains(code, "UPDATE users SET name = $1, email = $2 WHERE id = $3") {
		t.Errorf("Incorrect UPDATE query for postgres. Code:\n%s", code)
	}
}
