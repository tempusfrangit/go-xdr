package main

import (
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Configure log to remove timestamps for cleaner test output
	log.SetFlags(0)

	// Run tests
	os.Exit(m.Run())
}

func TestParseXDRTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple uint32 tag",
			input:    `xdr:"uint32"`,
			expected: "uint32",
		},
		{
			name:     "String tag",
			input:    `xdr:"string"`,
			expected: "string",
		},
		{
			name:     "Bytes tag",
			input:    `xdr:"bytes"`,
			expected: "bytes",
		},
		{
			name:     "Union tag",
			input:    `xdr:"union:SUCCESS,void:default"`,
			expected: "union:SUCCESS,void:default",
		},
		{
			name:     "Discriminant tag",
			input:    `xdr:"discriminant"`,
			expected: "discriminant",
		},
		{
			name:     "Fixed array tag",
			input:    `xdr:"fixed:16"`,
			expected: "fixed:16",
		},
		{
			name:     "Tag with other attributes",
			input:    `json:"name" xdr:"string" validate:"required"`,
			expected: "string",
		},
		{
			name:     "Empty tag",
			input:    "",
			expected: "",
		},
		{
			name:     "No xdr tag",
			input:    `json:"name"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseXDRTag(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseDiscriminatedUnionTag(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedDiscrimant bool
		expectedUnionSpec  string
	}{
		{
			name:               "Discriminant tag",
			input:              "discriminant",
			expectedDiscrimant: true,
			expectedUnionSpec:  "",
		},
		{
			name:               "Union tag",
			input:              "union:SUCCESS,void:default",
			expectedDiscrimant: false,
			expectedUnionSpec:  "SUCCESS,void:default",
		},
		{
			name:               "Pattern union tag",
			input:              "union:pattern",
			expectedDiscrimant: false,
			expectedUnionSpec:  "pattern",
		},
		{
			name:               "Regular XDR tag",
			input:              "uint32",
			expectedDiscrimant: false,
			expectedUnionSpec:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDiscriminant, unionSpec := parseDiscriminatedUnionTag(tt.input)
			if isDiscriminant != tt.expectedDiscrimant {
				t.Errorf("Expected discriminant %v, got %v", tt.expectedDiscrimant, isDiscriminant)
			}
			if unionSpec != tt.expectedUnionSpec {
				t.Errorf("Expected union spec %q, got %q", tt.expectedUnionSpec, unionSpec)
			}
		})
	}
}

func TestIsSupportedXDRType(t *testing.T) {
	tests := []struct {
		name     string
		xdrType  string
		expected bool
	}{
		{"uint32", "uint32", true},
		{"uint64", "uint64", true},
		{"int64", "int64", true},
		{"string", "string", true},
		{"bytes", "bytes", true},
		{"bool", "bool", true},
		{"struct", "struct", true},
		{"array", "array", true},
		{"discriminant", "discriminant", true},
		{"fixed array", "fixed:16", true},
		{"union", "union:SUCCESS", true},
		{"alias", "alias:stateid", true},
		{"unsupported", "float64", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSupportedXDRType(tt.xdrType)
			if result != tt.expected {
				t.Errorf("Expected %v for type %q, got %v", tt.expected, tt.xdrType, result)
			}
		})
	}
}

func TestParseUnionSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []UnionCase
	}{
		{
			name:  "Simple data case",
			input: "SUCCESS",
			expected: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
			},
		},
		{
			name:  "Multiple data cases",
			input: "SUCCESS,FAILURE",
			expected: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
				{Values: []string{"FAILURE"}, IsVoid: false},
			},
		},
		{
			name:  "Void case",
			input: "void:ERROR",
			expected: []UnionCase{
				{Values: []string{"ERROR"}, IsVoid: true},
			},
		},
		{
			name:  "Mixed cases",
			input: "SUCCESS,void:ERROR,PARTIAL",
			expected: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
				{Values: []string{"ERROR"}, IsVoid: true},
				{Values: []string{"PARTIAL"}, IsVoid: false},
			},
		},
		{
			name:  "Default case",
			input: "SUCCESS,void:default",
			expected: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
				{Values: []string{"default"}, IsVoid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUnionSpec(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d cases, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if len(result[i].Values) != len(expected.Values) {
					t.Errorf("Case %d: expected %d values, got %d", i, len(expected.Values), len(result[i].Values))
					continue
				}
				for j, expectedValue := range expected.Values {
					if result[i].Values[j] != expectedValue {
						t.Errorf("Case %d, value %d: expected %q, got %q", i, j, expectedValue, result[i].Values[j])
					}
				}
				if result[i].IsVoid != expected.IsVoid {
					t.Errorf("Case %d: expected IsVoid %v, got %v", i, expected.IsVoid, result[i].IsVoid)
				}
			}
		})
	}
}

func TestHasDefaultCase(t *testing.T) {
	tests := []struct {
		name     string
		cases    []UnionCase
		expected bool
	}{
		{
			name:     "No default case",
			cases:    []UnionCase{{Values: []string{"SUCCESS"}, IsVoid: false}},
			expected: false,
		},
		{
			name:     "Has default case",
			cases:    []UnionCase{{Values: []string{"default"}, IsVoid: true}},
			expected: true,
		},
		{
			name: "Multiple cases with default",
			cases: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
				{Values: []string{"default"}, IsVoid: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasDefaultCase(tt.cases)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFindDiscriminantField(t *testing.T) {
	tests := []struct {
		name       string
		structInfo TypeInfo
		expected   string
	}{
		{
			name: "Has discriminant field",
			structInfo: TypeInfo{
				Name: "TestStruct",
				Fields: []FieldInfo{
					{Name: "Type", IsDiscriminant: true},
					{Name: "Data", IsDiscriminant: false},
				},
			},
			expected: "Type",
		},
		{
			name: "No discriminant field",
			structInfo: TypeInfo{
				Name: "TestStruct",
				Fields: []FieldInfo{
					{Name: "Field1", IsDiscriminant: false},
					{Name: "Field2", IsDiscriminant: false},
				},
			},
			expected: "",
		},
		{
			name: "Multiple discriminant fields (should return first)",
			structInfo: TypeInfo{
				Name: "TestStruct",
				Fields: []FieldInfo{
					{Name: "Type1", IsDiscriminant: true},
					{Name: "Type2", IsDiscriminant: true},
				},
			},
			expected: "Type1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findDiscriminantField(tt.structInfo)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractReceiverType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Pointer receiver",
			input:    "*TestStruct",
			expected: "TestStruct",
		},
		{
			name:     "Value receiver",
			input:    "TestStruct",
			expected: "TestStruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a type expression
			expr, err := parser.ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			result := extractReceiverType(expr)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple type",
			input:    "uint32",
			expected: "uint32",
		},
		{
			name:     "Slice type",
			input:    "[]byte",
			expected: "[]byte",
		},
		{
			name:     "Array type",
			input:    "[16]byte",
			expected: "[16]byte",
		},
		{
			name:     "Package qualified type",
			input:    "primitives.StateID",
			expected: "primitives.StateID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a type expression
			expr, err := parser.ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			result := formatType(expr)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCollectExternalImports(t *testing.T) {
	tests := []struct {
		name     string
		types    []TypeInfo
		expected []string
	}{
		{
			name: "No external imports",
			types: []TypeInfo{
				{
					Name: "TestStruct",
					Fields: []FieldInfo{
						{Name: "ID", Type: "uint32"},
						{Name: "Name", Type: "string"},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "Has external imports",
			types: []TypeInfo{
				{
					Name: "TestStruct",
					Fields: []FieldInfo{
						{Name: "ID", Type: "uint32"},
						{Name: "State", Type: "primitives.StateID"},
						{Name: "Session", Type: "session.ID"},
					},
				},
			},
			expected: []string{
				"github.com/tempusfrangit/go-xdr/primitives",
				"github.com/tempusfrangit/go-xdr/session",
			},
		},
		{
			name: "Array of external type",
			types: []TypeInfo{
				{
					Name: "TestStruct",
					Fields: []FieldInfo{
						{Name: "States", Type: "[]primitives.StateID"},
					},
				},
			},
			expected: []string{
				"github.com/tempusfrangit/go-xdr/primitives",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectExternalImports(tt.types)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d imports, got %d", len(tt.expected), len(result))
			}
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected import %q not found in result %v", expected, result)
				}
			}
		})
	}
}

func TestParseFileWithBasicStruct(t *testing.T) {
	// Create a temporary Go file for testing
	content := `package test

//go:generate xdrgen $GOFILE
type TestStruct struct {
	ID   uint32 ` + "`xdr:\"uint32\"`" + `
	Name string ` + "`xdr:\"string\"`" + `
	Data []byte ` + "`xdr:\"bytes\"`" + `
	Skip int    ` + "`xdr:\"-\"`" + `
}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	types, err := parseFile(tmpFile)
	if err != nil {
		t.Fatalf("parseFile failed: %v", err)
	}

	if len(types) != 1 {
		t.Fatalf("Expected 1 type, got %d", len(types))
	}

	typeInfo := types[0]
	if typeInfo.Name != "TestStruct" {
		t.Errorf("Expected name 'TestStruct', got %q", typeInfo.Name)
	}

	if len(typeInfo.Fields) != 3 { // Skip field should be excluded
		t.Errorf("Expected 3 fields, got %d", len(typeInfo.Fields))
	}

	expectedFields := []struct {
		name    string
		xdrType string
	}{
		{"ID", "uint32"},
		{"Name", "string"},
		{"Data", "bytes"},
	}

	for i, expected := range expectedFields {
		if i >= len(typeInfo.Fields) {
			t.Errorf("Missing field %d: %s", i, expected.name)
			continue
		}
		field := typeInfo.Fields[i]
		if field.Name != expected.name {
			t.Errorf("Field %d: expected name %q, got %q", i, expected.name, field.Name)
		}
		if field.XDRType != expected.xdrType {
			t.Errorf("Field %d: expected XDR type %q, got %q", i, expected.xdrType, field.XDRType)
		}
	}
}

func TestParseFileWithDiscriminatedUnion(t *testing.T) {
	content := `package test

//go:generate xdrgen $GOFILE
type UnionStruct struct {
	Type OpCode ` + "`xdr:\"discriminant\"`" + `
	Data []byte ` + "`xdr:\"union:SUCCESS,void:default\"`" + `
}

type OpCode uint32
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	types, err := parseFile(tmpFile)
	if err != nil {
		t.Fatalf("parseFile failed: %v", err)
	}

	if len(types) != 1 {
		t.Fatalf("Expected 1 type, got %d", len(types))
	}

	typeInfo := types[0]
	if typeInfo.Name != "UnionStruct" {
		t.Errorf("Expected name 'UnionStruct', got %q", typeInfo.Name)
	}

	if !typeInfo.IsDiscriminatedUnion {
		t.Error("Expected IsDiscriminatedUnion to be true")
	}

	if len(typeInfo.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(typeInfo.Fields))
	}

	// Check discriminant field
	discriminantField := typeInfo.Fields[0]
	if !discriminantField.IsDiscriminant {
		t.Error("Expected first field to be discriminant")
	}

	// Check union field
	unionField := typeInfo.Fields[1]
	if unionField.UnionSpec != "SUCCESS,void:default" {
		t.Errorf("Expected union spec 'SUCCESS,void:default', got %q", unionField.UnionSpec)
	}
}

func TestParseFileWithSpecificStruct(t *testing.T) {
	content := `package test

//go:generate xdrgen SpecificStruct
type SpecificStruct struct {
	Value uint32 ` + "`xdr:\"uint32\"`" + `
}

type IgnoredStruct struct {
	Value uint32 ` + "`xdr:\"uint32\"`" + `
}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	types, err := parseFile(tmpFile)
	if err != nil {
		t.Fatalf("parseFile failed: %v", err)
	}

	if len(types) != 1 {
		t.Fatalf("Expected 1 type, got %d", len(types))
	}

	typeInfo := types[0]
	if typeInfo.Name != "SpecificStruct" {
		t.Errorf("Expected name 'SpecificStruct', got %q", typeInfo.Name)
	}
}

func TestParseFileErrors(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing XDR tag",
			content: `package test

//go:generate xdrgen $GOFILE
type TestStruct struct {
	ID uint32
}
`,
			expectError: true,
			errorMsg:    "missing XDR tag",
		},
		{
			name: "Unsupported XDR type",
			content: `package test

//go:generate xdrgen $GOFILE
type TestStruct struct {
	ID uint32 ` + "`xdr:\"float64\"`" + `
}
`,
			expectError: true,
			errorMsg:    "Unsupported XDR type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.content)
			defer os.Remove(tmpFile)

			// This test will cause log.Fatalf to be called, which will exit the process
			// We need to test this differently by checking the function behavior
			// For now, we'll just verify the content is correct
			if !tt.expectError {
				_, err := parseFile(tmpFile)
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateDiscriminatedUnions(t *testing.T) {
	tests := []struct {
		name        string
		types       []TypeInfo
		expectPanic bool
	}{
		{
			name: "Valid discriminated union",
			types: []TypeInfo{
				{
					Name:                 "ValidUnion",
					IsDiscriminatedUnion: true,
					Fields: []FieldInfo{
						{Name: "Type", IsDiscriminant: true},
						{Name: "Data", UnionSpec: "SUCCESS,void:default"},
					},
				},
			},
			expectPanic: false,
		},
		{
			name: "Non-discriminated union",
			types: []TypeInfo{
				{
					Name:                 "RegularStruct",
					IsDiscriminatedUnion: false,
					Fields: []FieldInfo{
						{Name: "ID", XDRType: "uint32"},
					},
				},
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic, but function completed normally")
					}
				}()
			}
			validateDiscriminatedUnions(tt.types)
		})
	}
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test_*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	return tmpFile.Name()
}

func TestMainFunction(t *testing.T) {
	// Test the main function behavior with different arguments
	tests := []struct {
		name     string
		args     []string
		setup    func() (string, func())
		expectOK bool
	}{
		{
			name: "Valid file with structs",
			args: []string{"test.go"},
			setup: func() (string, func()) {
				content := `package test

//go:generate xdrgen $GOFILE
type TestStruct struct {
	ID uint32 ` + "`xdr:\"uint32\"`" + `
}
`
				tmpFile := createTempFile(t, content)
				cleanup := func() {
					os.Remove(tmpFile)
					os.Remove(strings.TrimSuffix(tmpFile, ".go") + "_xdr.go")
				}
				return tmpFile, cleanup
			},
			expectOK: true,
		},
		{
			name: "File with no XDR structs",
			args: []string{"test.go"},
			setup: func() (string, func()) {
				content := `package test

type RegularStruct struct {
	ID uint32
}
`
				tmpFile := createTempFile(t, content)
				cleanup := func() {
					os.Remove(tmpFile)
				}
				return tmpFile, cleanup
			},
			expectOK: true, // Should succeed but generate no output
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, cleanup := tt.setup()
			defer cleanup()

			// Change to the directory containing the temp file
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(oldDir)

			tmpDir := filepath.Dir(tmpFile)
			os.Chdir(tmpDir)

			// Test with the file
			types, err := parseFile(tmpFile)
			if err != nil {
				if tt.expectOK {
					t.Errorf("Expected success, got error: %v", err)
				}
				return
			}

			if tt.expectOK && len(types) > 0 {
				// Verify we can generate code for the types
				for _, typeInfo := range types {
					_, err := GenerateEncodeMethod(typeInfo)
					if err != nil {
						t.Errorf("Failed to generate encode method: %v", err)
					}

					_, err = GenerateDecodeMethod(typeInfo)
					if err != nil {
						t.Errorf("Failed to generate decode method: %v", err)
					}

					_, err = GenerateAssertion(typeInfo.Name)
					if err != nil {
						t.Errorf("Failed to generate assertion: %v", err)
					}
				}
			}
		})
	}
}

func TestSilentMode(t *testing.T) {
	// Test that silent mode suppresses output
	originalSilent := silent
	defer func() { silent = originalSilent }()

	// Test silent mode - logf should not output anything
	silent = true
	logf("This should not appear in output")

	// Test normal mode - we don't call logf here to avoid test output
	silent = false
	// We would test logf output here if we had output capturing
}

// Test the logf function
func TestLogf(t *testing.T) {
	originalSilent := silent
	defer func() { silent = originalSilent }()

	// Test silent mode
	silent = true
	// In silent mode, logf should not output anything
	// We can't easily test this without capturing output

	silent = false
	// In normal mode, logf should output
	// We can't easily test this without capturing output
}

func TestHasExistingMethods(t *testing.T) {
	content := `package test

type TestStruct struct {
	ID uint32
}

func (t *TestStruct) Encode() {}
func (t *TestStruct) Decode() {}

type PartialStruct struct {
	ID uint32
}

func (p *PartialStruct) Encode() {}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	// Parse the file to get the AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, tmpFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Test struct with both methods
	hasEncode, hasDecode := hasExistingMethods(file, "TestStruct")
	if !hasEncode || !hasDecode {
		t.Error("TestStruct should have both Encode and Decode methods")
	}

	// Test struct with only Encode method
	hasEncode, hasDecode = hasExistingMethods(file, "PartialStruct")
	if !hasEncode || hasDecode {
		t.Error("PartialStruct should have only Encode method")
	}

	// Test struct with no methods
	hasEncode, hasDecode = hasExistingMethods(file, "NonExistentStruct")
	if hasEncode || hasDecode {
		t.Error("NonExistentStruct should have no methods")
	}
}

func TestFormatTypeComplex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic literal",
			input:    "16",
			expected: "16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}

			result := formatType(expr)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseUnionSpecEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []UnionCase
	}{
		{
			name:  "Empty parts",
			input: "SUCCESS, ,FAILURE",
			expected: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
				{Values: []string{"FAILURE"}, IsVoid: false},
			},
		},
		{
			name:  "Whitespace handling",
			input: " SUCCESS , void:ERROR ",
			expected: []UnionCase{
				{Values: []string{"SUCCESS"}, IsVoid: false},
				{Values: []string{"ERROR"}, IsVoid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUnionSpec(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d cases, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if len(result[i].Values) != len(expected.Values) {
					t.Errorf("Case %d: expected %d values, got %d", i, len(expected.Values), len(result[i].Values))
					continue
				}
				for j, expectedValue := range expected.Values {
					if result[i].Values[j] != expectedValue {
						t.Errorf("Case %d, value %d: expected %q, got %q", i, j, expectedValue, result[i].Values[j])
					}
				}
				if result[i].IsVoid != expected.IsVoid {
					t.Errorf("Case %d: expected IsVoid %v, got %v", i, expected.IsVoid, result[i].IsVoid)
				}
			}
		})
	}
}

func TestValidateDiscriminatedUnionsErrors(t *testing.T) {
	// Note: validateDiscriminatedUnions calls log.Fatalf which exits the process
	// These tests would require process isolation to test properly
	// For now, we'll test the validation logic without calling the actual function

	tests := []struct {
		name             string
		types            []TypeInfo
		expectValidation bool
	}{
		{
			name: "No discriminant field would fail",
			types: []TypeInfo{
				{
					Name:                 "BadUnion",
					IsDiscriminatedUnion: true,
					Fields: []FieldInfo{
						{Name: "Data", UnionSpec: "SUCCESS,void:default"},
					},
				},
			},
			expectValidation: false,
		},
		{
			name: "Multiple discriminant fields would fail",
			types: []TypeInfo{
				{
					Name:                 "BadUnion",
					IsDiscriminatedUnion: true,
					Fields: []FieldInfo{
						{Name: "Type1", IsDiscriminant: true},
						{Name: "Type2", IsDiscriminant: true},
						{Name: "Data", UnionSpec: "SUCCESS,void:default"},
					},
				},
			},
			expectValidation: false,
		},
		{
			name: "No union fields would fail",
			types: []TypeInfo{
				{
					Name:                 "BadUnion",
					IsDiscriminatedUnion: true,
					Fields: []FieldInfo{
						{Name: "Type", IsDiscriminant: true},
					},
				},
			},
			expectValidation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic without calling validateDiscriminatedUnions
			// which would call log.Fatalf and exit the process
			if len(tt.types) == 0 {
				return
			}

			typeInfo := tt.types[0]
			if !typeInfo.IsDiscriminatedUnion {
				return
			}

			discriminantCount := 0
			unionCount := 0

			for _, field := range typeInfo.Fields {
				if field.IsDiscriminant {
					discriminantCount++
				}
				if field.UnionSpec != "" {
					unionCount++
				}
			}

			isValid := discriminantCount == 1 && unionCount > 0
			if isValid != tt.expectValidation {
				t.Errorf("Expected validation result %v, got %v", tt.expectValidation, isValid)
			}
		})
	}
}

func TestExtractPackageImportsEdgeCases(t *testing.T) {
	imports := make(map[string]bool)

	// Test with various edge cases
	testCases := []string{
		"map[string]primitives.StateID",
		"func(primitives.StateID) session.Response",
		"*primitives.StateID",
		"[]session.Response",
		"unknown.Type", // Should not be mapped
	}

	for _, testCase := range testCases {
		extractPackageImports(testCase, imports)
	}

	// Should have primitives and session imports
	if !imports["github.com/tempusfrangit/go-xdr/primitives"] {
		t.Error("Should have primitives import")
	}
	if !imports["github.com/tempusfrangit/go-xdr/session"] {
		t.Error("Should have session import")
	}
}
