package main

import (
	"strings"
	"testing"
)

func TestTemplateManager(t *testing.T) {
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager failed: %v", err)
	}

	// Test that templates are loaded
	expectedTemplates := []string{
		"file_header",
		"encode_method",
		"decode_method",
		"assertion",
		"field_encode_basic",
		"field_decode_basic",
		"field_encode_struct",
		"field_decode_struct",
		"array_encode",
		"array_decode",
		"union_encode",
		"union_decode",
		"union_case_void",
		"union_case_bytes_encode",
		"union_case_bytes_decode",
		"union_case_struct_encode",
		"union_case_struct_decode",
	}

	for _, templateName := range expectedTemplates {
		t.Run("Template_"+templateName, func(t *testing.T) {
			tmpl, exists := tm.GetTemplate(templateName)
			if !exists {
				t.Errorf("Template %q not found", templateName)
			}
			if tmpl == nil {
				t.Errorf("Template %q is nil", templateName)
			}
		})
	}
}

func TestTemplateManagerExecute(t *testing.T) {
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager failed: %v", err)
	}

	// Test executing a simple template
	data := TypeData{
		TypeName: "TestStruct",
	}

	result, err := tm.ExecuteTemplate("assertion", data)
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}

	if !strings.Contains(result, "TestStruct") {
		t.Errorf("Template result should contain 'TestStruct', got: %s", result)
	}
}

func TestTemplateManagerExecuteNonExistent(t *testing.T) {
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager failed: %v", err)
	}

	_, err = tm.ExecuteTemplate("nonexistent", nil)
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestGenerateFileHeader(t *testing.T) {
	result, err := GenerateFileHeader("test.go", "testpkg", []string{"fmt", "strings"})
	if err != nil {
		t.Fatalf("GenerateFileHeader failed: %v", err)
	}

	// Check that the result contains expected elements
	if !strings.Contains(result, "testpkg") {
		t.Error("Result should contain package name")
	}
	if !strings.Contains(result, "test.go") {
		t.Error("Result should contain source file name")
	}
}

func TestGenerateFileHeaderNoImports(t *testing.T) {
	result, err := GenerateFileHeader("test.go", "testpkg", []string{})
	if err != nil {
		t.Fatalf("GenerateFileHeader failed: %v", err)
	}

	// Should work with no external imports
	if !strings.Contains(result, "testpkg") {
		t.Error("Result should contain package name")
	}
}

func TestGenerateEncodeMethod(t *testing.T) {
	typeInfo := TypeInfo{
		Name: "TestStruct",
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint32", XDRType: "uint32"},
			{Name: "Name", Type: "string", XDRType: "string"},
		},
	}

	result, err := GenerateEncodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateEncodeMethod failed: %v", err)
	}

	// Check that the result contains expected elements
	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain type name")
	}
	if !strings.Contains(result, "Encode") {
		t.Error("Result should contain Encode method")
	}
	if !strings.Contains(result, "ID") {
		t.Error("Result should contain field names")
	}
}

func TestGenerateDecodeMethod(t *testing.T) {
	typeInfo := TypeInfo{
		Name: "TestStruct",
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint32", XDRType: "uint32"},
			{Name: "Name", Type: "string", XDRType: "string"},
		},
	}

	result, err := GenerateDecodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateDecodeMethod failed: %v", err)
	}

	// Check that the result contains expected elements
	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain type name")
	}
	if !strings.Contains(result, "Decode") {
		t.Error("Result should contain Decode method")
	}
	if !strings.Contains(result, "ID") {
		t.Error("Result should contain field names")
	}
}

func TestGenerateAssertion(t *testing.T) {
	result, err := GenerateAssertion("TestStruct")
	if err != nil {
		t.Fatalf("GenerateAssertion failed: %v", err)
	}

	// Check that the result contains expected elements
	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain type name")
	}
	if !strings.Contains(result, "xdr.Codec") {
		t.Error("Result should contain Codec interface")
	}
}

func TestGenerateBasicEncodeCode(t *testing.T) {
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
			result, err := generateBasicEncodeCode(tt.field)
			if err != nil {
				t.Fatalf("generateBasicEncodeCode failed: %v", err)
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain %q, got: %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateBasicDecodeCode(t *testing.T) {
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
			result, err := generateBasicDecodeCode(tt.field)
			if err != nil {
				t.Fatalf("generateBasicDecodeCode failed: %v", err)
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to contain %q, got: %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateArrayEncodeCode(t *testing.T) {
	field := FieldInfo{Name: "Items", Type: "[]uint32", XDRType: "array"}
	result, err := generateBasicEncodeCode(field)
	if err != nil {
		t.Fatalf("generateBasicEncodeCode failed: %v", err)
	}

	// Should contain array-specific logic
	if !strings.Contains(result, "Items") {
		t.Error("Result should contain field name")
	}
}

func TestGenerateArrayDecodeCode(t *testing.T) {
	field := FieldInfo{Name: "Items", Type: "[]uint32", XDRType: "array"}
	result, err := generateBasicDecodeCode(field)
	if err != nil {
		t.Fatalf("generateBasicDecodeCode failed: %v", err)
	}

	// Should contain array-specific logic
	if !strings.Contains(result, "Items") {
		t.Error("Result should contain field name")
	}
}

func TestGenerateUnionEncodeCode(t *testing.T) {
	field := FieldInfo{
		Name:      "Data",
		Type:      "[]byte",
		XDRType:   "union",
		UnionSpec: "SUCCESS,void:default",
	}

	structInfo := TypeInfo{
		Name: "TestUnion",
		Fields: []FieldInfo{
			{Name: "Type", IsDiscriminant: true},
			field,
		},
	}

	result, err := generateUnionEncodeCode(field, structInfo)
	if err != nil {
		t.Fatalf("generateUnionEncodeCode failed: %v", err)
	}

	// Should contain union-specific logic
	if !strings.Contains(result, "switch") {
		t.Error("Result should contain switch statement")
	}
	if !strings.Contains(result, "Type") {
		t.Error("Result should contain discriminant field")
	}
}

func TestGenerateUnionDecodeCode(t *testing.T) {
	field := FieldInfo{
		Name:      "Data",
		Type:      "[]byte",
		XDRType:   "union",
		UnionSpec: "SUCCESS,void:default",
	}

	structInfo := TypeInfo{
		Name: "TestUnion",
		Fields: []FieldInfo{
			{Name: "Type", IsDiscriminant: true},
			field,
		},
	}

	result, err := generateUnionDecodeCode(field, structInfo)
	if err != nil {
		t.Fatalf("generateUnionDecodeCode failed: %v", err)
	}

	// Should contain union-specific logic
	if !strings.Contains(result, "switch") {
		t.Error("Result should contain switch statement")
	}
	if !strings.Contains(result, "Type") {
		t.Error("Result should contain discriminant field")
	}
}

func TestGenerateUnionCodeError(t *testing.T) {
	field := FieldInfo{
		Name:      "Data",
		Type:      "[]byte",
		XDRType:   "union",
		UnionSpec: "SUCCESS,void:default",
	}

	// Struct without discriminant field should cause error
	structInfo := TypeInfo{
		Name: "TestUnion",
		Fields: []FieldInfo{
			{Name: "NotDiscriminant", IsDiscriminant: false},
			field,
		},
	}

	_, err := generateUnionEncodeCode(field, structInfo)
	if err == nil {
		t.Error("Expected error for union without discriminant field")
	}

	_, err = generateUnionDecodeCode(field, structInfo)
	if err == nil {
		t.Error("Expected error for union without discriminant field")
	}
}

func TestGetEncodeMethod(t *testing.T) {
	tests := []struct {
		xdrType  string
		expected string
	}{
		{"uint32", "EncodeUint32"},
		{"uint64", "EncodeUint64"},
		{"int64", "EncodeInt64"},
		{"string", "EncodeString"},
		{"bytes", "EncodeBytes"},
		{"bool", "EncodeBool"},
		{"discriminant", "EncodeUint32"},
		{"struct", "Encode"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.xdrType, func(t *testing.T) {
			result := getEncodeMethod(tt.xdrType)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetDecodeMethod(t *testing.T) {
	tests := []struct {
		xdrType  string
		expected string
	}{
		{"uint32", "DecodeUint32"},
		{"uint64", "DecodeUint64"},
		{"int64", "DecodeInt64"},
		{"string", "DecodeString"},
		{"bytes", "DecodeBytes"},
		{"bool", "DecodeBool"},
		{"discriminant", "DecodeUint32"},
		{"struct", "Decode"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.xdrType, func(t *testing.T) {
			result := getDecodeMethod(tt.xdrType)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateUnsupportedXDRType(t *testing.T) {
	field := FieldInfo{Name: "Bad", Type: "float64", XDRType: "unsupported"}

	_, err := generateBasicEncodeCode(field)
	if err == nil {
		t.Error("Expected error for unsupported XDR type")
	}

	_, err = generateBasicDecodeCode(field)
	if err == nil {
		t.Error("Expected error for unsupported XDR type")
	}
}

func TestTemplateFunctions(t *testing.T) {
	// Test template functions
	t.Run("toLowerCase", func(t *testing.T) {
		fn := templateFuncs["toLowerCase"].(func(string) string)

		tests := []struct {
			input    string
			expected string
		}{
			{"TestStruct", "testStruct"},
			{"ID", "iD"},
			{"", ""},
			{"a", "a"},
			{"ABC", "aBC"},
		}

		for _, tt := range tests {
			result := fn(tt.input)
			if result != tt.expected {
				t.Errorf("toLowerCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("trimPrefix", func(t *testing.T) {
		fn := templateFuncs["trimPrefix"].(func(string, string) string)

		result := fn("prefixValue", "prefix")
		if result != "Value" {
			t.Errorf("trimPrefix failed: got %q, want %q", result, "Value")
		}
	})
}

func TestComplexUnionGeneration(t *testing.T) {
	// Test complex union generation with multiple cases
	field := FieldInfo{
		Name:      "Data",
		Type:      "*OperationResult",
		XDRType:   "union",
		UnionSpec: "SUCCESS,PARTIAL,void:ERROR,void:default",
	}

	structInfo := TypeInfo{
		Name: "OperationUnion",
		Fields: []FieldInfo{
			{Name: "OpCode", IsDiscriminant: true},
			field,
		},
	}

	encodeResult, err := generateUnionEncodeCode(field, structInfo)
	if err != nil {
		t.Fatalf("generateUnionEncodeCode failed: %v", err)
	}

	decodeResult, err := generateUnionDecodeCode(field, structInfo)
	if err != nil {
		t.Fatalf("generateUnionDecodeCode failed: %v", err)
	}

	// Should contain all case values
	expectedCases := []string{"SUCCESS", "PARTIAL", "ERROR", "default"}
	for _, expectedCase := range expectedCases {
		if !strings.Contains(encodeResult, expectedCase) {
			t.Errorf("Encode result should contain case %q", expectedCase)
		}
		if !strings.Contains(decodeResult, expectedCase) {
			t.Errorf("Decode result should contain case %q", expectedCase)
		}
	}
}

func TestGlobalTemplateManager(t *testing.T) {
	// Test that the global template manager is initialized
	if templateManager == nil {
		t.Error("Global template manager should be initialized")
	}

	// Test that we can use it
	result, err := templateManager.ExecuteTemplate("assertion", TypeData{TypeName: "Test"})
	if err != nil {
		t.Errorf("Global template manager should work: %v", err)
	}

	if !strings.Contains(result, "Test") {
		t.Error("Global template manager should produce expected output")
	}
}

func TestGenerateDiscriminatedUnionMethods(t *testing.T) {
	// Test generating methods for discriminated union structs
	typeInfo := TypeInfo{
		Name: "TestUnion",
		Fields: []FieldInfo{
			{Name: "Type", Type: "uint32", XDRType: "discriminant", IsDiscriminant: true},
			{Name: "Data", Type: "[]byte", XDRType: "union", UnionSpec: "SUCCESS,void:default"},
		},
		IsDiscriminatedUnion: true,
	}

	encodeResult, err := GenerateEncodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateEncodeMethod failed: %v", err)
	}

	decodeResult, err := GenerateDecodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateDecodeMethod failed: %v", err)
	}

	// Should contain union-specific logic
	if !strings.Contains(encodeResult, "switch") {
		t.Error("Encode result should contain switch statement for union")
	}
	if !strings.Contains(decodeResult, "switch") {
		t.Error("Decode result should contain switch statement for union")
	}
}

func TestGenerateFixedBytesCode(t *testing.T) {
	// Test that fixed bytes field generates an error (not supported in basic generation)
	field := FieldInfo{Name: "Hash", Type: "[16]byte", XDRType: "fixed:16"}

	_, err := generateBasicEncodeCode(field)
	if err == nil {
		t.Error("generateBasicEncodeCode should fail for fixed bytes (not supported)")
	}

	_, err = generateBasicDecodeCode(field)
	if err == nil {
		t.Error("generateBasicDecodeCode should fail for fixed bytes (not supported)")
	}
}

func TestNewTemplateManagerError(t *testing.T) {
	// This test is difficult to trigger without breaking the embedded filesystem
	// We'll just verify the current manager works
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager should work: %v", err)
	}

	if tm == nil {
		t.Error("NewTemplateManager should return valid manager")
	}
}

func TestTemplateManagerInitError(t *testing.T) {
	// Test that the init function handles errors correctly
	// This is difficult to test without modifying the embedded filesystem
	// We'll just verify that templateManager is properly initialized
	if templateManager == nil {
		t.Error("Template manager should be initialized by init function")
	}
}
