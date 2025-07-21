package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set up test environment
	os.Exit(m.Run())
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
		{"fixed array", "fixed:16", true},
		{"alias", "alias:stateid", true},
		{"unsupported", "float64", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSupportedXDRType(tt.xdrType)
			assert.Equal(t, tt.expected, result, "Expected %v for type %q", tt.expected, tt.xdrType)
		})
	}
}

func TestParseXDRTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Skip tag",
			input:    `xdr:"-"`,
			expected: "-",
		},
		{
			name:     "Skip tag with other attributes",
			input:    `json:"name" xdr:"-" validate:"required"`,
			expected: "-",
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
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractReceiverType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Pointer receiver", "*TestStruct", "*TestStruct"},
		{"Value receiver", "TestStruct", "TestStruct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a type expression
			expr, err := parser.ParseExpr(tt.input)
			require.NoError(t, err, "Failed to parse expression")

			result := extractReceiverType(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Basic type", "uint32", "uint32"},
		{"Slice type", "[]byte", "[]byte"},
		{"Array type", "[16]byte", "[16]byte"},
		{"Package qualified", "pkg.Type", "pkg.Type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a type expression
			expr, err := parser.ParseExpr(tt.input)
			require.NoError(t, err, "Failed to parse expression")

			result := formatType(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectExternalImports(t *testing.T) {
	// Create a mock AST file with imports
	mockFile := &ast.File{
		Imports: []*ast.ImportSpec{
			{
				Path: &ast.BasicLit{Value: `"github.com/tempusfrangit/go-xdr/primitives"`},
			},
			{
				Path: &ast.BasicLit{Value: `"github.com/tempusfrangit/go-xdr/session"`},
			},
		},
	}

	tests := []struct {
		name     string
		types    []TypeInfo
		file     *ast.File
		expected []string
	}{
		{
			name: "No external imports",
			types: []TypeInfo{
				{
					Name: "TestStruct",
					Fields: []FieldInfo{
						{Name: "ID", Type: "uint32", XDRType: "uint32"},
						{Name: "Name", Type: "string", XDRType: "string"},
					},
				},
			},
			file:     nil,
			expected: []string{},
		},
		{
			name: "With external imports but no file",
			types: []TypeInfo{
				{
					Name: "TestStruct",
					Fields: []FieldInfo{
						{Name: "ID", Type: "uint32", XDRType: "uint32"},
						{Name: "State", Type: "primitives.StateID", XDRType: "struct"},
						{Name: "Session", Type: "session.ID", XDRType: "struct"},
					},
				},
			},
			file:     nil,
			expected: []string{}, // Can't resolve imports without AST file
		},
		{
			name: "With external imports and file",
			types: []TypeInfo{
				{
					Name: "TestStruct",
					Fields: []FieldInfo{
						{Name: "ID", Type: "uint32", XDRType: "uint32"},
						{Name: "State", Type: "primitives.StateID", XDRType: "bytes"},
						{Name: "Session", Type: "session.ID", XDRType: "bytes"},
					},
				},
			},
			file: mockFile,
			expected: []string{
				"github.com/tempusfrangit/go-xdr/primitives",
				"github.com/tempusfrangit/go-xdr/session",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectExternalImports(tt.types, tt.file)
			assert.Len(t, result, len(tt.expected), "Expected %d imports", len(tt.expected))
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Expected import %q not found", expected)
			}
		})
	}
}

func TestExtractPackageImportsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected []string
	}{
		{"Empty string", "", []string{}},
		{"Basic type", "uint32", []string{}},
		{"Package qualified", "primitives.StateID", []string{"primitives"}},
		{"Multiple packages", "primitives.StateID,session.Response", []string{"primitives", "session"}},
		{"Array with package", "[]primitives.StateID", []string{"primitives"}},
		{"Pointer with package", "*primitives.StateID", []string{"primitives"}},
		{"Unknown package", "unknown.Type", []string{"unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageNamesFromType(tt.typeStr)

			assert.Len(t, result, len(tt.expected), "Expected %d packages", len(tt.expected))
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Expected package %q not found", expected)
			}
		})
	}
}

func TestFindKeyField(t *testing.T) {
	tests := []struct {
		name       string
		structInfo TypeInfo
		expected   string
	}{
		{
			name: "Has key field",
			structInfo: TypeInfo{
				Name: "TestStruct",
				Fields: []FieldInfo{
					{Name: "Type", IsKey: true},
					{Name: "Data", IsUnion: true},
				},
			},
			expected: "Type",
		},
		{
			name: "No key field",
			structInfo: TypeInfo{
				Name: "TestStruct",
				Fields: []FieldInfo{
					{Name: "Field1", IsKey: false},
					{Name: "Field2", IsKey: false},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findKeyField(tt.structInfo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseUnionComment(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected *UnionConfig
		hasError bool
	}{
		{
			name:    "Valid union comment",
			comment: "//xdr:union=OpCode,case=OpOpen=OpOpenResult",
			expected: &UnionConfig{
				ContainerType: "OpCode",
				Cases:         map[string]string{"OpOpen": "OpOpenResult"},
				VoidCases:     []string{},
			},
			hasError: false,
		},
		{
			name:    "Union comment with default",
			comment: "//xdr:union=OpCode,case=OpOpen=OpOpenResult",
			expected: &UnionConfig{
				ContainerType: "OpCode",
				Cases:         map[string]string{"OpOpen": "OpOpenResult"},
				VoidCases:     []string{},
			},
			hasError: false,
		},
		{
			name:     "Not a union comment",
			comment:  "// regular comment",
			expected: nil,
			hasError: false,
		},
		{
			name:     "Invalid union comment",
			comment:  "//xdr:union=",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseUnionComment(tt.comment)
			if tt.hasError {
				require.Error(t, err, "Expected error, got nil")
			} else {
				require.NoError(t, err, "Unexpected error")
			}
			if tt.expected != nil {
				require.NotNil(t, result, "Expected result, got nil")
			} else {
				require.Nil(t, result, "Expected nil, got result")
			}
		})
	}
}

func TestHasExistingMethods(t *testing.T) {
	content := `package test

import "github.com/tempusfrangit/go-xdr"

type TestStruct struct {
	ID uint32
}

func (t *TestStruct) Encode(enc *xdr.Encoder) error { return nil }
func (t *TestStruct) Decode(dec *xdr.Decoder) error { return nil }

type PartialStruct struct {
	ID uint32
}

func (p *PartialStruct) Encode(enc *xdr.Encoder) error { return nil }
`

	tmpFile := createTempFile(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	// Parse the file to get the AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, tmpFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Test struct with both methods
	hasEncode, hasDecode := hasExistingMethods(file, "TestStruct")
	assert.True(t, hasEncode, "TestStruct should have Encode method")
	assert.True(t, hasDecode, "TestStruct should have Decode method")

	// Test struct with only Encode method
	hasEncode, hasDecode = hasExistingMethods(file, "PartialStruct")
	assert.True(t, hasEncode, "PartialStruct should have Encode method")
	assert.False(t, hasDecode, "PartialStruct should not have Decode method")

	// Test struct with no methods
	hasEncode, hasDecode = hasExistingMethods(file, "NonExistentStruct")
	assert.False(t, hasEncode, "NonExistentStruct should not have Encode method")
	assert.False(t, hasDecode, "NonExistentStruct should not have Decode method")
}

func TestExtractBuildTags(t *testing.T) {
	content := `//go:build linux
// +build linux

package test

type TestStruct struct {
	ID uint32
}
`

	tmpFile := createTempFile(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	tags := extractBuildTags(tmpFile)
	assert.NotEmpty(t, tags, "Should extract build tags")

	expectedTags := []string{"//go:build linux", "// +build linux"}
	for _, expected := range expectedTags {
		found := false
		for _, actual := range tags {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected build tag %q not found", expected)
	}
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	return tmpFile.Name()
}
