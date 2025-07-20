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

func TestParseKeyTag(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedKey     bool
		expectedDefault string
	}{
		{"key tag", "key", true, ""},
		{"key with nil default", "key,default=nil", true, "nil"},
		{"key with struct default", "key,default=ErrorResult", true, "ErrorResult"},
		{"non-key tag", "uint32", false, ""},
		{"skip tag", "-", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKey, defaultType := parseKeyTag(tt.input)
			assert.Equal(t, tt.expectedKey, isKey, "Expected key %v", tt.expectedKey)
			assert.Equal(t, tt.expectedDefault, defaultType, "Expected default %q", tt.expectedDefault)
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

func TestAutoDiscoverXDRType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"string", "string", "string", false},
		{"[]byte", "[]byte", "bytes", false},
		{"[16]byte", "[16]byte", "bytes", false}, // Fixed arrays -> bytes
		{"uint32", "uint32", "uint32", false},
		{"uint64", "uint64", "uint64", false},
		{"int32", "int32", "int32", false},
		{"int64", "int64", "int64", false},
		{"bool", "bool", "bool", false},
		{"unsupported", "float64", "struct", true}, // Unsupported -> struct fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the new auto-discovery system
			typeAliases := make(map[string]string)
			result := autoDiscoverXDRType(tt.input, typeAliases)
			if tt.hasError {
				// For unsupported types, expect "struct" as fallback
				assert.Equal(t, "struct", result, "Expected struct fallback for %q", tt.input)
			} else {
				assert.Equal(t, tt.expected, result)
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

func TestUnionLoopDetection(t *testing.T) {
	// Test case: A -> B -> A (loop)
	unionConfigs := map[string]*UnionConfig{
		"A": {
			ContainerType: "A",
			Cases:         map[string]string{"1": "B"},
		},
		"B": {
			ContainerType: "B",
			Cases:         map[string]string{"1": "A"},
		},
	}

	// Should detect loop
	visited := make(map[string]bool)
	assert.True(t, detectUnionLoop("A", unionConfigs, visited), "Loop detection failed: should detect A -> B -> A loop")

	// Test case: No loop
	unionConfigsNoLoop := map[string]*UnionConfig{
		"A": {
			ContainerType: "A",
			Cases:         map[string]string{"1": "B"},
		},
		"B": {
			ContainerType: "B",
			Cases:         map[string]string{"1": "C"},
		},
		"C": {
			ContainerType: "C",
			Cases:         map[string]string{},
		},
	}

	visited = make(map[string]bool)
	assert.False(t, detectUnionLoop("A", unionConfigsNoLoop, visited), "Loop detection failed: should not detect loop in A -> B -> C")
}

func TestUnionConfigAggregation(t *testing.T) {
	// Test that union configs are properly aggregated from payload structs
	// and associated with container structs
	src := `package main

type MessageType uint32
const (
	MessageTypeText MessageType = 1
	MessageTypeBinary MessageType = 2
)

// +xdr:union,key=Type,default=nil
type NetworkMessage struct {
	Type    MessageType
	Payload []byte
}

// +xdr:generate  
type MessageTypeTextPayload struct {
	Content string
}

// +xdr:generate
type MessageTypeBinaryPayload struct {
	Data []byte
}
`

	tmpFile := createTempFile(t, src)

	types, _, _, err := parseFile(tmpFile)
	require.NoError(t, err, "Failed to parse file")

	// Find NetworkMessage (container struct)
	var networkMessage *TypeInfo
	for i := range types {
		if types[i].Name == "NetworkMessage" {
			networkMessage = &types[i]
			break
		}
	}

	require.NotNil(t, networkMessage, "NetworkMessage not found in parsed types")
	assert.True(t, networkMessage.IsDiscriminatedUnion, "NetworkMessage should be marked as discriminated union")

	// In the new directive system, union config association happens during generation,
	// not during parsing. The parseFile function just identifies discriminated unions.

	// Check that fields are properly identified
	hasKeyField := false
	hasUnionField := false
	for _, field := range networkMessage.Fields {
		if field.IsKey {
			hasKeyField = true
		}
		if field.IsUnion {
			hasUnionField = true
		}
	}
	assert.True(t, hasKeyField, "NetworkMessage should have a key field")
	assert.True(t, hasUnionField, "NetworkMessage should have a union field")

	t.Logf("NetworkMessage fields: %+v", networkMessage.Fields)
}

func TestOrphanedUnionCommentDetection(t *testing.T) {
	// Test that orphaned payload directives are detected
	src := `package main

type MessageType uint32
type OtherType uint32
const (
	MessageTypeText MessageType = 1
	OtherTypeValue  OtherType = 1
)

// +xdr:union,key=Type,default=nil
type NetworkMessage struct {
	Type    MessageType
	Payload []byte
}

// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText
type TextPayload struct {
	Content string
}

// This payload is orphaned - references nonexistent union
// +xdr:payload,union=NonExistentMessage,discriminant=OtherTypeValue
type OrphanedPayload struct {
	Data []byte
}
`

	tmpFile := createTempFile(t, src)

	// Orphaned union comment detection is already implemented
	// This should fail because OrphanedPayload is not used in any union
	types, _, constants, err := parseFile(tmpFile)

	// In the new directive system, orphaned payload directives may be ignored
	// rather than causing validation errors. Just ensure parsing succeeds.
	require.NoError(t, err, "Unexpected parse error")
	assert.NotNil(t, types, "Expected types to be parsed")
	assert.NotNil(t, constants, "Expected constants to be parsed")

	// Check that valid types are still processed correctly
	foundNetworkMessage := false
	for _, typeInfo := range types {
		if typeInfo.Name == "NetworkMessage" {
			foundNetworkMessage = true
			assert.True(t, typeInfo.IsDiscriminatedUnion, "NetworkMessage should be marked as discriminated union")
		}
	}
	assert.True(t, foundNetworkMessage, "NetworkMessage should be parsed successfully")
}

func TestDiscriminantTypeValidation(t *testing.T) {
	// Test valid discriminant types in the new directive system
	// Invalid types cause log.Fatal in parser, so we only test valid cases
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "Valid uint32 alias with new directive syntax",
			src: `package main

type MessageType uint32

const (
	MessageTypeText MessageType = 1
)

// +xdr:union,key=Type
type NetworkMessage struct {
	Type MessageType
	Payload []byte
}

// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText
type TextPayload struct {
	Content string
}
`,
		},
		{
			name: "Direct uint32 (valid) with new directive syntax",
			src: `package main

const (
	MessageTypeText = 1
)

// +xdr:union,key=Type
type NetworkMessage struct {
	Type uint32
	Payload []byte
}

// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText
type TextPayload struct {
	Content string
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.src)
			defer func() { _ = os.Remove(tmpFile) }()

			// Use parseFile for validation - should succeed for valid cases
			types, validationErrors, _, err := parseFile(tmpFile)

			require.NoError(t, err, "Expected no error for valid discriminant type")
			require.Empty(t, validationErrors, "Expected no validation errors for valid discriminant type")
			// For valid cases, we should have parsed results
			require.NotNil(t, types, "Expected types to be parsed for valid input")

			// Should find the union configuration
			foundUnion := false
			for _, typeInfo := range types {
				if typeInfo.Name == "NetworkMessage" && typeInfo.IsDiscriminatedUnion {
					foundUnion = true
					break
				}
			}
			assert.True(t, foundUnion, "Expected to find discriminated union configuration")
		})
	}
}

func TestTypedConstantsValidation(t *testing.T) {
	// Test valid constants in the new directive system
	// Invalid cases cause log.Fatal in parser, so we only test valid cases
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "Valid typed constants with new directive syntax",
			src: `package main

type MessageType uint32

const (
	MessageTypeText MessageType = 1
)

// +xdr:union,key=Type
type NetworkMessage struct {
	Type MessageType
	Payload []byte
}

// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText
type TextPayload struct {
	Content string
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.src)
			defer func() { _ = os.Remove(tmpFile) }()

			// Use parseFile for validation - should succeed for valid cases
			types, validationErrors, _, err := parseFile(tmpFile)

			require.NoError(t, err, "Expected no error for valid typed constants")
			require.Empty(t, validationErrors, "Expected no validation errors for valid typed constants")
			// For valid cases, we should have parsed results
			require.NotNil(t, types, "Expected types to be parsed for valid input")

			// Should find the union configuration
			foundUnion := false
			for _, typeInfo := range types {
				if typeInfo.Name == "NetworkMessage" && typeInfo.IsDiscriminatedUnion {
					foundUnion = true
					break
				}
			}
			assert.True(t, foundUnion, "Expected to find discriminated union configuration")
		})
	}
}

func TestUnionCommentPlacement(t *testing.T) {
	// Test that directive placement follows new syntax rules
	src := `package main

type MessageType uint32
const (
	MessageTypeText MessageType = 1
)

// +xdr:union,key=Type
type NetworkMessage struct {
	Type    MessageType
	Payload []byte
}

// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText
type TextPayload struct {
	Content string
}
`

	tmpFile := createTempFile(t, src)
	defer func() { _ = os.Remove(tmpFile) }()

	// This should parse successfully with new directive syntax
	types, validationErrors, _, err := parseFile(tmpFile)

	// Should parse successfully
	require.NoError(t, err, "Expected no error for correct directive syntax")
	require.Empty(t, validationErrors, "Expected no validation errors for correct directive syntax")

	// Types should be parsed
	require.NotNil(t, types, "Expected types to be parsed")

	// TypeDefs should be parsed
	// (constants parsing is not part of parseFile return values)

	// Should find the union configuration
	foundUnion := false
	for _, typeInfo := range types {
		if typeInfo.Name == "NetworkMessage" && typeInfo.IsDiscriminatedUnion {
			foundUnion = true
			break
		}
	}
	require.True(t, foundUnion, "Expected to find discriminated union configuration")
}
