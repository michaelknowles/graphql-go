package packer

import (
	"reflect"
	"testing"
)

func TestBuildArgFieldMap_BasicCases(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		expected  map[string][]int
		shouldErr bool
		errMsg    string
	}{
		{
			name: "simple struct with graphql tags",
			input: struct {
				Field1 string `graphql:"arg1"`
				Field2 string `graphql:"arg2"`
			}{},
			expected: map[string][]int{
				"arg1": {0},
				"arg2": {1},
			},
		},
		{
			name: "struct with name-based matching",
			input: struct {
				SessionID string
				UserID    string
			}{},
			expected: map[string][]int{
				"SessionID": {0},
				"UserID":    {1},
			},
		},
		{
			name: "mixed tags and names",
			input: struct {
				Tagged   string `graphql:"special"`
				UserName string // will use name-based matching
			}{},
			expected: map[string][]int{
				"special":  {0},
				"UserName": {1},
			},
		},
		{
			name:     "empty struct",
			input:    struct{}{},
			expected: map[string][]int{},
		},
		{
			name: "single field with tag",
			input: struct {
				ID string `graphql:"userId"`
			}{},
			expected: map[string][]int{
				"userId": {0},
			},
		},
		{
			name: "non-struct type string",
			input: "",
			expected: nil,
		},
		{
			name: "non-struct type int", 
			input: 42,
			expected: nil,
		},
		{
			name: "struct with unexported field",
			input: struct {
				exported   string `graphql:"exp"`
				unexported string `graphql:"unexp"`
			}{},
			expected: map[string][]int{
				"exp":   {0},
				"unexp": {1},
			},
		},
		{
			name: "pointer to struct",
			input: &struct{ Field string }{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputType reflect.Type
			if tt.input == nil {
				t.Fatal("input cannot be nil")
			}
			
			// Handle non-struct types
			switch v := tt.input.(type) {
			case string, int:
				inputType = reflect.TypeOf(v)
			case *struct{ Field string }:
				inputType = reflect.TypeOf(v)
			default:
				inputType = reflect.TypeOf(tt.input)
			}
			
			result, err := buildArgFieldMap(inputType)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q but got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expected == nil && result != nil {
				t.Errorf("expected nil result but got %+v", result)
			}

			if tt.expected != nil {
				if len(result) != len(tt.expected) {
					t.Errorf("expected %d fields but got %d", len(tt.expected), len(result))
				}

				for key, expectedIndex := range tt.expected {
					actualIndex, exists := result[key]
					if !exists {
						t.Errorf("expected key %q not found in result", key)
						continue
					}
					if len(actualIndex) != len(expectedIndex) {
						t.Errorf("for key %q: expected index length %d but got %d", key, len(expectedIndex), len(actualIndex))
						continue
					}
					for i, exp := range expectedIndex {
						if actualIndex[i] != exp {
							t.Errorf("for key %q at position %d: expected index %d but got %d", key, i, exp, actualIndex[i])
						}
					}
				}
			}
		})
	}
}

func TestBuildArgFieldMap_NestedStructures(t *testing.T) {
	type EmbeddedStruct struct {
		EmbeddedField string `graphql:"embedded"`
	}

	type NestedStruct struct {
		DeepField string `graphql:"deep"`
	}

	type NamedFieldStruct struct {
		NamedField string
	}

	tests := []struct {
		name      string
		input     any
		expected  map[string][]int
		shouldErr bool
		errMsg    string
	}{
		{
			name: "simple embedded struct",
			input: struct {
				EmbeddedStruct
				DirectField string `graphql:"direct"`
			}{},
			expected: map[string][]int{
				"embedded": {0, 0}, // First 0 is for EmbeddedStruct, second 0 is for EmbeddedField
				"direct":   {1},
			},
		},
		{
			name: "anonymous nested struct",
			input: struct {
				NestedStruct
				TopField string `graphql:"top"`
			}{},
			expected: map[string][]int{
				"deep": {0, 0},
				"top":  {1},
			},
		},
		{
			name: "anonymous struct with name-based field",
			input: struct {
				NamedFieldStruct
				Other string `graphql:"other"`
			}{},
			expected: map[string][]int{
				"NamedField": {0, 0},
				"other":      {1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildArgFieldMap(reflect.TypeOf(tt.input))

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q but got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for key, expectedIndex := range tt.expected {
				actualIndex, exists := result[key]
				if !exists {
					t.Errorf("expected key %q not found in result", key)
					continue
				}
				if len(actualIndex) != len(expectedIndex) {
					t.Errorf("for key %q: expected index length %d but got %d", key, len(expectedIndex), len(actualIndex))
					continue
				}
				for i, exp := range expectedIndex {
					if actualIndex[i] != exp {
						t.Errorf("for key %q at position %d: expected index %d but got %d", key, i, exp, actualIndex[i])
					}
				}
			}
		})
	}
}

func TestBuildArgFieldMap_ErrorCases(t *testing.T) {
	type DuplicateTagStruct struct {
		Field2 string `graphql:"duplicate"`
	}
	tests := []struct {
		name   string
		input  any
		errMsg string
	}{
		{
			name: "duplicate graphql tags",
			input: struct {
				Field1 string `graphql:"same"`
				Field2 string `graphql:"same"`
			}{},
			errMsg: `multiple fields have a graphql reflect tag "same"`,
		},
		{
			name: "ambiguous field names",
			input: struct {
				sessionid string
				SessionId string
			}{},
			errMsg: `ambiguous field "sessionid"`,
		},
		{
			name: "ambiguous field names with underscore",
			input: struct {
				user_id string
				UserId  string
			}{},
			errMsg: `ambiguous field "user_id"`,
		},
		{
			name: "duplicate tags in nested struct",
			input: struct {
				Field1 string `graphql:"duplicate"`
				DuplicateTagStruct
			}{},
			errMsg: `multiple fields have a graphql reflect tag "duplicate"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildArgFieldMap(reflect.TypeOf(tt.input))

			if err == nil {
				t.Errorf("expected error but got none")
			} else if tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("expected error %q but got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestBuildArgFieldMap_TagPriority(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected map[string][]int
	}{
		{
			name: "tag takes priority over name match",
			input: struct {
				UserID    string `graphql:"sessionId"` // tag wins over name
				SessionId string // name-based, but conflicts with tag
			}{},
			expected: map[string][]int{
				"sessionId": {0}, // Only the tagged field should be mapped
				"SessionId": {1}, // The other field gets name-based mapping
			},
		},
		{
			name: "tag overrides case-insensitive name match",
			input: struct {
				Field   string `graphql:"SPECIAL"`
				SpeciaL string // would match "special" case-insensitively
			}{},
			expected: map[string][]int{
				"SPECIAL": {0},
				"SpeciaL": {1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildArgFieldMap(reflect.TypeOf(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields but got %d", len(tt.expected), len(result))
				return
			}

			for key, expectedIndex := range tt.expected {
				actualIndex, exists := result[key]
				if !exists {
					t.Errorf("expected key %q not found in result", key)
					continue
				}
				for i, exp := range expectedIndex {
					if actualIndex[i] != exp {
						t.Errorf("for key %q at position %d: expected index %d but got %d", key, i, exp, actualIndex[i])
					}
				}
			}
		})
	}
}

func TestBuildArgFieldMap_CaseSensitivity(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected map[string][]int
	}{
		{
			name: "case sensitive tag matching",
			input: struct {
				Field1 string `graphql:"Lower"`
				Field2 string `graphql:"lower"` // Different from "Lower"
			}{},
			expected: map[string][]int{
				"Lower": {0},
				"lower": {1},
			},
		},
		{
			name: "name-based matching with different cases",
			input: struct {
				UserID string
				UserId string
			}{},
			// This should error due to ambiguity since both normalize to "userid"
			expected: nil, // Will be handled in error test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildArgFieldMap(reflect.TypeOf(tt.input))

			if tt.expected == nil {
				// Expected to error for ambiguous case
				if err == nil {
					t.Errorf("expected error for ambiguous case but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for key, expectedIndex := range tt.expected {
				actualIndex, exists := result[key]
				if !exists {
					t.Errorf("expected key %q not found in result", key)
					continue
				}
				for i, exp := range expectedIndex {
					if actualIndex[i] != exp {
						t.Errorf("for key %q at position %d: expected index %d but got %d", key, i, exp, actualIndex[i])
					}
				}
			}
		})
	}
}
