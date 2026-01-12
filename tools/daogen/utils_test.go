package main

import "testing"

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserDetails", "user_details"},
		{"ID", "id"},
		{"UserID", "user_id"},
		{"UserFirstName", "user_first_name"},
		{"SQLServer", "sql_server"},
		{"My4Struct", "my4_struct"},
	}

	for _, tt := range tests {
		result := ToSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("ToSnakeCase(%s): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}
