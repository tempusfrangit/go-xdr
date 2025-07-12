package main

import (
	"go/parser"
	"go/token"
	"os"
	"testing"
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
		{"key", "key", true},
		{"fixed array", "fixed:16", true},
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
			input:    `xdr:"union,default=nil"`,
			expected: "union,default=nil",
		},
		{
			name:     "Key tag",
			input:    `xdr:"key"`,
			expected: "key",
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

func TestParseKeyUnionTag(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedKey     bool
		expectedUnion   bool
		expectedDefault string
	}{
		{"key tag", "key", true, false, ""},
		{"union tag", "union", false, true, ""},
		{"union with default", "union,default=OpOpenResult", false, true, "OpOpenResult"},
		{"union with nil default", "union,default=nil", false, true, "nil"},
		{"regular tag", "uint32", false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isKey, isUnion, defaultType := parseKeyUnionTag(tt.input)
			if isKey != tt.expectedKey {
				t.Errorf("Expected key %v, got %v", tt.expectedKey, isKey)
			}
			if isUnion != tt.expectedUnion {
				t.Errorf("Expected union %v, got %v", tt.expectedUnion, isUnion)
			}
			if defaultType != tt.expectedDefault {
				t.Errorf("Expected default %q, got %q", tt.expectedDefault, defaultType)
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
		{"Pointer receiver", "*TestStruct", "TestStruct"},
		{"Value receiver", "TestStruct", "TestStruct"},
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
		{"Basic type", "uint32", "uint32"},
		{"Slice type", "[]byte", "[]byte"},
		{"Array type", "[16]byte", "[16]byte"},
		{"Package qualified", "pkg.Type", "pkg.Type"},
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
						{Name: "ID", Type: "uint32", XDRType: "uint32"},
						{Name: "Name", Type: "string", XDRType: "string"},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "With external imports",
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
			expected: []string{
				"github.com/tempusfrangit/go-xdr/primitives",
				"github.com/tempusfrangit/go-xdr/session",
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

func TestExtractPackageImportsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected []string
	}{
		{"Empty string", "", []string{}},
		{"Basic type", "uint32", []string{}},
		{"Package qualified", "primitives.StateID", []string{"github.com/tempusfrangit/go-xdr/primitives"}},
		{"Multiple packages", "primitives.StateID,session.Response", []string{"github.com/tempusfrangit/go-xdr/primitives", "github.com/tempusfrangit/go-xdr/session"}},
		{"Array with package", "[]primitives.StateID", []string{"github.com/tempusfrangit/go-xdr/primitives"}},
		{"Pointer with package", "*primitives.StateID", []string{"github.com/tempusfrangit/go-xdr/primitives"}},
		{"Unknown package", "unknown.Type", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageImports(tt.typeStr)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d packages, got %d", len(tt.expected), len(result))
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
					t.Errorf("Expected package %q not found in result %v", expected, result)
				}
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
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
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
			if tt.hasError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expected != nil && result == nil {
				t.Error("Expected result, got nil")
			}
			if tt.expected == nil && result != nil {
				t.Error("Expected nil, got result")
			}
		})
	}
}

func TestInferUnderlyingType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"string", "string", "string", false},
		{"[]byte", "[]byte", "bytes", false},
		{"[16]byte", "[16]byte", "fixed:16", false},
		{"uint32", "uint32", "uint32", false},
		{"uint64", "uint64", "uint64", false},
		{"int32", "int32", "int32", false},
		{"int64", "int64", "int64", false},
		{"bool", "bool", "bool", false},
		{"unsupported", "float64", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := inferUnderlyingType(tt.input)
			if tt.hasError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
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

func TestExtractBuildTags(t *testing.T) {
	content := `//go:build linux
// +build linux

package test

type TestStruct struct {
	ID uint32
}
`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	tags := extractBuildTags(tmpFile)
	if len(tags) == 0 {
		t.Error("Should extract build tags")
	}

	expectedTags := []string{"//go:build linux", "// +build linux"}
	for _, expected := range expectedTags {
		found := false
		for _, actual := range tags {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected build tag %q not found", expected)
		}
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
	if !detectUnionLoop("A", unionConfigs, visited) {
		t.Error("Loop detection failed: should detect A -> B -> A loop")
	}

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
	if detectUnionLoop("A", unionConfigsNoLoop, visited) {
		t.Error("Loop detection failed: should not detect loop in A -> B -> C")
	}
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

type NetworkMessage struct {
	Type    MessageType ` + "`xdr:\"key\"`" + `
	Payload []byte ` + "`xdr:\"union,default=nil\"`" + `
}

//xdr:union=NetworkMessage,case=MessageTypeText
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}

//xdr:union=NetworkMessage,case=MessageTypeBinary
type BinaryPayload struct {
	Data []byte ` + "`xdr:\"bytes\"`" + `
}
`

	tmpFile := createTempFile(t, src)

	types, _, _, err := parseFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Find NetworkMessage (container struct)
	var networkMessage *TypeInfo
	for i := range types {
		if types[i].Name == "NetworkMessage" {
			networkMessage = &types[i]
			break
		}
	}

	if networkMessage == nil {
		t.Fatal("NetworkMessage not found in parsed types")
	}

	if !networkMessage.IsDiscriminatedUnion {
		t.Error("NetworkMessage should be marked as discriminated union")
	}

	// Check that union config is properly associated
	if networkMessage.UnionConfig == nil {
		t.Error("NetworkMessage should have union config associated")
	} else if networkMessage.UnionConfig.ContainerType != "NetworkMessage" {
		t.Errorf("Expected container type NetworkMessage, got %s", networkMessage.UnionConfig.ContainerType)
	}

	t.Logf("NetworkMessage fields: %+v", networkMessage.Fields)
}

func TestOrphanedUnionCommentDetection(t *testing.T) {
	// Test that orphaned union comments are detected
	src := `package main

type MessageType uint32
type OtherType uint32
const (
	MessageTypeText MessageType = 1
	OtherTypeValue  OtherType = 1
)

type NetworkMessage struct {
	Type    MessageType ` + "`xdr:\"key\"`" + `
	Payload []byte      ` + "`xdr:\"union,default=nil\"`" + `
}

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}

// This comment is orphaned - no union uses this struct
//xdr:union=OtherType,case=OtherTypeValue=OrphanedPayload
type OrphanedPayload struct {
	Data []byte ` + "`xdr:\"bytes\"`" + `
}
`

	tmpFile := createTempFile(t, src)

	// Orphaned union comment detection is already implemented
	// This should fail because OrphanedPayload is not used in any union
	_, _, _, err := parseFile(tmpFile)
	if err == nil {
		t.Error("Expected error for orphaned union comment, but got none")
	}
}

func TestDiscriminantTypeValidation(t *testing.T) {
	// Test that discriminant types must be uint32 aliases
	tests := []struct {
		name        string
		src         string
		shouldError bool
	}{
		{
			name: "Valid uint32 alias",
			src: `package main

type MessageType uint32

const (
	MessageTypeText MessageType = 1
)

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: false,
		},
		{
			name: "Invalid string alias",
			src: `package main

type MessageType string  // Wrong type - should be uint32

const (
	MessageTypeText MessageType = "text"
)

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: true,
		},
		{
			name: "Invalid int64 alias",
			src: `package main

type MessageType int64  // Wrong type - should be uint32

const (
	MessageTypeText MessageType = 1
)

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: true,
		},
		{
			name: "Direct uint32 (valid)",
			src: `package main

//xdr:union=uint32,case=1=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.src)
			defer os.Remove(tmpFile)

			_, _, _, err := parseFile(tmpFile)

			if tt.shouldError && err == nil {
				t.Error("Expected error for invalid discriminant type, but got none")
			} else if !tt.shouldError && err != nil {
				t.Errorf("Expected no error for valid discriminant type, but got: %v", err)
			}
		})
	}
}

func TestTypedConstantsValidation(t *testing.T) {
	// Test that constants must be of the discriminant type
	tests := []struct {
		name        string
		src         string
		shouldError bool
	}{
		{
			name: "Valid typed constants",
			src: `package main

type MessageType uint32

const (
	MessageTypeText MessageType = 1
)

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: false,
		},
		{
			name: "Invalid untyped constants",
			src: `package main

type MessageType uint32

const (
	MessageTypeText = 1  // Wrong - should be MessageType = 1
)

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: true,
		},
		{
			name: "Missing constants",
			src: `package main

type MessageType uint32

// No constants defined

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.src)
			defer os.Remove(tmpFile)

			_, _, _, err := parseFile(tmpFile)

			if tt.shouldError && err == nil {
				t.Error("Expected error for invalid typed constants, but got none")
			} else if !tt.shouldError && err != nil {
				t.Errorf("Expected no error for valid typed constants, but got: %v", err)
			}
		})
	}
}

func TestUnionCommentPlacement(t *testing.T) {
	// Test that union comments are only allowed above payload structs
	src := `package main

type MessageType uint32
const (
	MessageTypeText MessageType = 1
)

//xdr:union=MessageType,case=MessageTypeText=TextPayload
type NetworkMessage struct {  // Wrong - comment above container, not payload
	Type    uint32 ` + "`xdr:\"key\"`" + `
	Payload []byte ` + "`xdr:\"union,default=nil\"`" + `
}

type TextPayload struct {
	Content string ` + "`xdr:\"string\"`" + `
}
`

	tmpFile := createTempFile(t, src)

	// Union comment placement validation is already implemented
	// This should fail because the comment is above a container struct, not a payload
	_, _, _, err := parseFile(tmpFile)
	if err == nil {
		t.Error("Expected error for misplaced union comment, but got none")
	}
}
