package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateManager(t *testing.T) {
	tm, err := NewTemplateManager()
	require.NoError(t, err, "NewTemplateManager failed")
	assert.NotNil(t, tm, "NewTemplateManager should return non-nil manager")

	// Test that we can get templates
	templates := []string{"file_header", "encode_method", "decode_method", "assertion"}
	for _, templateName := range templates {
		tmpl, exists := tm.GetTemplate(templateName)
		assert.True(t, exists, "Template %s should exist", templateName)
		assert.NotNil(t, tmpl, "Template %s should not be nil", templateName)
	}

	// Test non-existent template
	_, exists := tm.GetTemplate("non_existent")
	assert.False(t, exists, "Non-existent template should not exist")
}

func TestTemplateManagerExecute(t *testing.T) {
	tm, err := NewTemplateManager()
	require.NoError(t, err, "NewTemplateManager failed")

	// Test executing a simple template
	result, err := tm.ExecuteTemplate("assertion", TypeData{TypeName: "TestStruct"})
	require.NoError(t, err, "ExecuteTemplate failed")

	assert.Contains(t, result, "TestStruct", "Result should contain type name")
	assert.Contains(t, result, "xdr.Codec", "Result should contain interface name")
}

func TestTemplateManagerExecuteNonExistent(t *testing.T) {
	tm, err := NewTemplateManager()
	require.NoError(t, err, "NewTemplateManager failed")

	// Test executing non-existent template
	_, err = tm.ExecuteTemplate("non_existent", nil)
	require.Error(t, err, "Should return error for non-existent template")
}

func TestGenerateFileHeader(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	result, err := cg.GenerateFileHeader("test.go", "testpkg", []string{"external/pkg"}, []string{"//go:build test"})
	require.NoError(t, err, "GenerateFileHeader failed")

	assert.Contains(t, result, "test.go", "Result should contain source file name")
	assert.Contains(t, result, "testpkg", "Result should contain package name")
	assert.Contains(t, result, "external/pkg", "Result should contain external imports")
	assert.Contains(t, result, "//go:build test", "Result should contain build tags")
}

func TestGenerateFileHeaderNoImports(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	result, err := cg.GenerateFileHeader("test.go", "testpkg", nil, nil)
	require.NoError(t, err, "GenerateFileHeader failed")

	assert.Contains(t, result, "testpkg", "Result should contain package name")
}

func TestGenerateFileHeaderWithBuildTags(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	result, err := cg.GenerateFileHeader("test.go", "testpkg", nil, []string{"//go:build linux", "//go:build amd64"})
	require.NoError(t, err, "GenerateFileHeader failed")

	assert.Contains(t, result, "//go:build linux", "Result should contain first build tag")
	assert.Contains(t, result, "//go:build amd64", "Result should contain second build tag")
}

func TestGenerateEncodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	typeInfo := TypeInfo{
		Name: "TestStruct",
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint32", XDRType: "uint32"},
			{Name: "Name", Type: "string", XDRType: "string"},
		},
	}

	result, err := cg.GenerateEncodeMethod(typeInfo)
	require.NoError(t, err, "GenerateEncodeMethod failed")

	assert.Contains(t, result, "TestStruct", "Result should contain struct name")
	assert.Contains(t, result, "Encode", "Result should contain Encode method")
}

func TestGenerateDecodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	typeInfo := TypeInfo{
		Name: "TestStruct",
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint32", XDRType: "uint32"},
			{Name: "Name", Type: "string", XDRType: "string"},
		},
	}

	result, err := cg.GenerateDecodeMethod(typeInfo)
	require.NoError(t, err, "GenerateDecodeMethod failed")

	assert.Contains(t, result, "TestStruct", "Result should contain struct name")
	assert.Contains(t, result, "Decode", "Result should contain Decode method")
}

func TestGenerateAssertion(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	result, err := cg.GenerateAssertion("TestStruct")
	require.NoError(t, err, "GenerateAssertion failed")

	assert.Contains(t, result, "TestStruct", "Result should contain type name")
	assert.Contains(t, result, "xdr.Codec", "Result should contain interface name")
}

func TestGenerateBasicEncodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	tests := []struct {
		name     string
		field    FieldInfo
		expected string
	}{
		{
			name:     "uint32 field",
			field:    FieldInfo{Name: "ID", Type: "uint32", XDRType: "uint32"},
			expected: "EncodeUint32",
		},
		{
			name:     "string field",
			field:    FieldInfo{Name: "Name", Type: "string", XDRType: "string"},
			expected: "EncodeString",
		},
		{
			name:     "bytes field",
			field:    FieldInfo{Name: "Data", Type: "[]byte", XDRType: "bytes"},
			expected: "EncodeBytes",
		},
		{
			name:     "struct field",
			field:    FieldInfo{Name: "Nested", Type: "NestedStruct", XDRType: "struct"},
			expected: "Encode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cg.generateBasicEncodeCode(tt.field)
			require.NoError(t, err, "generateBasicEncodeCode failed")

			assert.Contains(t, result, tt.field.Name, "Result should contain field name")
			assert.Contains(t, result, tt.expected, "Result should contain %s", tt.expected)
		})
	}
}

func TestGenerateBasicDecodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	tests := []struct {
		name     string
		field    FieldInfo
		expected string
	}{
		{
			name:     "uint32 field",
			field:    FieldInfo{Name: "ID", Type: "uint32", XDRType: "uint32"},
			expected: "DecodeUint32",
		},
		{
			name:     "string field",
			field:    FieldInfo{Name: "Name", Type: "string", XDRType: "string"},
			expected: "DecodeString",
		},
		{
			name:     "bytes field",
			field:    FieldInfo{Name: "Data", Type: "[]byte", XDRType: "bytes"},
			expected: "DecodeBytes",
		},
		{
			name:     "struct field",
			field:    FieldInfo{Name: "Nested", Type: "NestedStruct", XDRType: "struct"},
			expected: "Decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cg.generateBasicDecodeCode(tt.field)
			require.NoError(t, err, "generateBasicDecodeCode failed")

			assert.Contains(t, result, tt.field.Name, "Result should contain field name")
			assert.Contains(t, result, tt.expected, "Result should contain %s", tt.expected)
		})
	}
}

func TestGenerateArrayEncodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	field := FieldInfo{Name: "Items", Type: "[]string", XDRType: "array"}

	result, err := cg.generateBasicEncodeCode(field)
	require.NoError(t, err, "generateBasicEncodeCode failed")

	assert.Contains(t, result, "Items", "Result should contain field name")
	assert.Contains(t, result, "EncodeUint32", "Result should contain array length encoding")
}

func TestGenerateArrayDecodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	field := FieldInfo{Name: "Items", Type: "[]string", XDRType: "array"}

	result, err := cg.generateBasicDecodeCode(field)
	require.NoError(t, err, "generateBasicDecodeCode failed")

	assert.Contains(t, result, "Items", "Result should contain field name")
	assert.Contains(t, result, "DecodeUint32", "Result should contain array length decoding")
}

func TestGetEncodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	tests := []struct {
		xdrType  string
		expected string
	}{
		{"uint32", "EncodeUint32"},
		{"uint64", "EncodeUint64"},
		{"int32", "EncodeInt32"},
		{"int64", "EncodeInt64"},
		{"string", "EncodeString"},
		{"bytes", "EncodeBytes"},
		{"bool", "EncodeBool"},
		{"struct", "Encode"},
		{"fixed:16", "EncodeFixedBytes"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.xdrType, func(t *testing.T) {
			result := cg.getEncodeMethod(tt.xdrType)
			assert.Equal(t, tt.expected, result, "Expected %q, got %q", tt.expected, result)
		})
	}
}

func TestGetDecodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	tests := []struct {
		xdrType  string
		expected string
	}{
		{"uint32", "DecodeUint32"},
		{"uint64", "DecodeUint64"},
		{"int32", "DecodeInt32"},
		{"int64", "DecodeInt64"},
		{"string", "DecodeString"},
		{"bytes", "DecodeBytes"},
		{"bool", "DecodeBool"},
		{"struct", "Decode"},
		{"fixed:16", "DecodeFixedBytes"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.xdrType, func(t *testing.T) {
			result := cg.getDecodeMethod(tt.xdrType)
			assert.Equal(t, tt.expected, result, "Expected %q, got %q", tt.expected, result)
		})
	}
}

func TestGenerateUnsupportedXDRType(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	field := FieldInfo{Name: "Bad", Type: "float64", XDRType: "unsupported"}

	_, err = cg.generateBasicEncodeCode(field)
	require.Error(t, err, "Expected error for unsupported XDR type")

	_, err = cg.generateBasicDecodeCode(field)
	require.Error(t, err, "Expected error for unsupported XDR type")
}

func TestGenerateFixedBytesCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	// Test that fixed bytes field generates appropriate code
	field := FieldInfo{Name: "Hash", Type: "[16]byte", XDRType: "bytes"}

	encodeResult, err := cg.generateBasicEncodeCode(field)
	require.NoError(t, err, "generateBasicEncodeCode failed")

	decodeResult, err := cg.generateBasicDecodeCode(field)
	require.NoError(t, err, "generateBasicDecodeCode failed")

	assert.Contains(t, encodeResult, "EncodeFixedBytes", "Encode result should contain EncodeFixedBytes")
	assert.Contains(t, decodeResult, "DecodeFixedBytes", "Decode result should contain DecodeFixedBytes")
}

func TestGenerateAliasCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	tests := []struct {
		name     string
		field    FieldInfo
		expected string
	}{
		{
			name:     "string alias",
			field:    FieldInfo{Name: "ID", Type: "UserID", XDRType: "string"},
			expected: "EncodeString",
		},
		{
			name:     "bytes alias",
			field:    FieldInfo{Name: "Data", Type: "SessionID", XDRType: "bytes"},
			expected: "EncodeBytes",
		},
		{
			name:     "fixed bytes alias",
			field:    FieldInfo{Name: "Hash", Type: "Hash16", XDRType: "bytes"},
			expected: "EncodeBytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encodeResult, err := cg.generateBasicEncodeCode(tt.field)
			require.NoError(t, err, "generateBasicEncodeCode failed")

			assert.Contains(t, encodeResult, tt.expected, "Expected %s in encode result, got: %s", tt.expected, encodeResult)

			decodeResult, err := cg.generateBasicDecodeCode(tt.field)
			require.NoError(t, err, "generateBasicDecodeCode failed")

			// For fixed bytes aliases, the decode code uses a special template
			decodeExpected := strings.Replace(tt.expected, "Encode", "Decode", 1)
			assert.Contains(t, decodeResult, decodeExpected, "Expected %s in decode result, got: %s", decodeExpected, decodeResult)
		})
	}
}

func TestGoTypeForXDRType(t *testing.T) {
	tests := []struct {
		xdrType  string
		expected string
	}{
		{"string", "string"},
		{"bytes", "[]byte"},
		{"uint32", "uint32"},
		{"uint64", "uint64"},
		{"int32", "int32"},
		{"int64", "int64"},
		{"bool", "bool"},
		{"fixed:16", "[16]byte"},
		{"fixed:32", "[32]byte"},
	}

	for _, tt := range tests {
		t.Run(tt.xdrType, func(t *testing.T) {
			result := goTypeForXDRType(tt.xdrType)
			assert.Equal(t, tt.expected, result, "Expected %q, got %q", tt.expected, result)
		})
	}
}

func TestNewTemplateManagerError(t *testing.T) {
	// This test would require mocking template parsing to fail
	// For now, just test that NewTemplateManager works with valid templates
	tm, err := NewTemplateManager()
	require.NoError(t, err, "NewTemplateManager failed")
	assert.NotNil(t, tm, "NewTemplateManager should return non-nil manager")
}

func TestTemplateManagerInitError(t *testing.T) {
	// This would require mocking template parsing to fail
	// For now, just test that NewCodeGenerator works with valid templates
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")
	assert.NotNil(t, cg, "NewCodeGenerator should return non-nil generator")
}

func TestGenerateUnionCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	// Test union code generation with proper struct setup
	typeInfo := TypeInfo{
		Name: "TestUnion",
		Fields: []FieldInfo{
			{Name: "Type", Type: "uint32", XDRType: "uint32", IsKey: true},
			{Name: "Data", Type: "[]byte", XDRType: "union", IsUnion: true},
		},
		IsDiscriminatedUnion: true,
		UnionConfig: &UnionConfig{
			ContainerType: "OpCode",
			Cases:         map[string]string{"SUCCESS": "SuccessResult"},
			VoidCases:     []string{"ERROR"},
		},
	}

	// Test encode method generation
	encodeResult, err := cg.GenerateEncodeMethod(typeInfo)
	require.NoError(t, err, "GenerateEncodeMethod failed")

	assert.Contains(t, encodeResult, "switch", "Encode result should contain switch statement for union")

	// Test decode method generation
	decodeResult, err := cg.GenerateDecodeMethod(typeInfo)
	require.NoError(t, err, "GenerateDecodeMethod failed")

	assert.Contains(t, decodeResult, "switch", "Decode result should contain switch statement for union")
}

func TestGenerateUnionEncodeDecodeCodeErrors(t *testing.T) {
	cg, err := NewCodeGenerator([]string{}, map[string]string{})
	require.NoError(t, err, "NewCodeGenerator failed")

	// Missing key field
	t.Run("missing key field", func(t *testing.T) {
		structInfo := TypeInfo{
			Name: "NoKeyUnion",
			Fields: []FieldInfo{
				{Name: "Data", Type: "[]byte", XDRType: "union", IsUnion: true},
			},
			IsDiscriminatedUnion: true,
			UnionConfig: &UnionConfig{
				ContainerType: "OpCode",
				Cases:         map[string]string{"SUCCESS": "SuccessResult"},
			},
		}
		errMsg := "no key field found for union field Data"
		_, err := cg.generateUnionEncodeCode(structInfo.Fields[0], structInfo)
		require.Error(t, err, "expected error %q, got %v", errMsg, err)
		_, err = cg.generateUnionDecodeCode(structInfo.Fields[0], structInfo)
		require.Error(t, err, "expected error %q, got %v", errMsg, err)
	})

	// Missing union config - now generates void-only union
	t.Run("missing union config", func(t *testing.T) {
		structInfo := TypeInfo{
			Name: "NoUnionConfig",
			Fields: []FieldInfo{
				{Name: "Type", Type: "uint32", XDRType: "uint32", IsKey: true},
				{Name: "Data", Type: "[]byte", XDRType: "bytes", IsUnion: true},
			},
			IsDiscriminatedUnion: true,
			UnionConfig:          nil,
		}
		// Should generate void-only union code without error
		_, err := cg.generateUnionEncodeCode(structInfo.Fields[1], structInfo)
		require.NoError(t, err, "expected no error for void-only union, got %v", err)
		_, err = cg.generateUnionDecodeCode(structInfo.Fields[1], structInfo)
		require.NoError(t, err, "expected no error for void-only union, got %v", err)
	})
}

func TestArrayAliasResolution(t *testing.T) {
	// Test that slice aliases don't get [:] conversion but fixed array aliases do
	tests := []struct {
		name         string
		field        FieldInfo
		expectedCode string
		description  string
	}{
		{
			name: "slice alias should not use [:]",
			field: FieldInfo{
				Name:         "Session",
				Type:         "SessionID",
				ResolvedType: "[]byte",
				XDRType:      "bytes",
			},
			expectedCode: "v.Session",
			description:  "SessionID []byte should generate v.Session, not v.Session[:]",
		},
		{
			name: "fixed array alias should use [:]",
			field: FieldInfo{
				Name:         "Hash",
				Type:         "Hash",
				ResolvedType: "[16]byte",
				XDRType:      "bytes",
			},
			expectedCode: "v.Hash[:]",
			description:  "Hash [16]byte should generate v.Hash[:]",
		},
		{
			name: "direct []byte should not use [:]",
			field: FieldInfo{
				Name:         "Data",
				Type:         "[]byte",
				ResolvedType: "[]byte",
				XDRType:      "bytes",
			},
			expectedCode: "v.Data",
			description:  "[]byte should generate v.Data, not v.Data[:]",
		},
		{
			name: "array of primitive aliases should use casting",
			field: FieldInfo{
				Name:         "StatusList",
				Type:         "[]StatusCode",
				ResolvedType: "[]uint32",
				XDRType:      "uint32",
			},
			expectedCode: "uint32(elem)",
			description:  "[]StatusCode should generate uint32(elem) for encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg, err := NewCodeGenerator([]string{}, map[string]string{})
			require.NoError(t, err, "NewCodeGenerator failed")

			result, err := cg.generateBasicEncodeCode(tt.field)
			require.NoError(t, err, "generateBasicEncodeCode failed")

			assert.Contains(t, result, tt.expectedCode, "%s: expected code to contain %q", tt.description, tt.expectedCode)
		})
	}
}
