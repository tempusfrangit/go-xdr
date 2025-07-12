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

	if tm == nil {
		t.Error("NewTemplateManager should return non-nil manager")
	}

	// Test that we can get templates
	templates := []string{"file_header", "encode_method", "decode_method", "assertion"}
	for _, templateName := range templates {
		tmpl, exists := tm.GetTemplate(templateName)
		if !exists {
			t.Errorf("Template %s should exist", templateName)
		}
		if tmpl == nil {
			t.Errorf("Template %s should not be nil", templateName)
		}
	}

	// Test non-existent template
	_, exists := tm.GetTemplate("non_existent")
	if exists {
		t.Error("Non-existent template should not exist")
	}
}

func TestTemplateManagerExecute(t *testing.T) {
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager failed: %v", err)
	}

	// Test executing a simple template
	result, err := tm.ExecuteTemplate("assertion", TypeData{TypeName: "TestStruct"})
	if err != nil {
		t.Fatalf("ExecuteTemplate failed: %v", err)
	}

	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain type name")
	}
	if !strings.Contains(result, "xdr.Codec") {
		t.Error("Result should contain interface name")
	}
}

func TestTemplateManagerExecuteNonExistent(t *testing.T) {
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager failed: %v", err)
	}

	// Test executing non-existent template
	_, err = tm.ExecuteTemplate("non_existent", nil)
	if err == nil {
		t.Error("Should return error for non-existent template")
	}
}

func TestGenerateFileHeader(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	result, err := cg.GenerateFileHeader("test.go", "testpkg", []string{"external/pkg"}, []string{"//go:build test"})
	if err != nil {
		t.Fatalf("GenerateFileHeader failed: %v", err)
	}

	if !strings.Contains(result, "test.go") {
		t.Error("Result should contain source file name")
	}
	if !strings.Contains(result, "testpkg") {
		t.Error("Result should contain package name")
	}
	if !strings.Contains(result, "external/pkg") {
		t.Error("Result should contain external imports")
	}
	if !strings.Contains(result, "//go:build test") {
		t.Error("Result should contain build tags")
	}
}

func TestGenerateFileHeaderNoImports(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	result, err := cg.GenerateFileHeader("test.go", "testpkg", nil, nil)
	if err != nil {
		t.Fatalf("GenerateFileHeader failed: %v", err)
	}

	if !strings.Contains(result, "testpkg") {
		t.Error("Result should contain package name")
	}
}

func TestGenerateFileHeaderWithBuildTags(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	result, err := cg.GenerateFileHeader("test.go", "testpkg", nil, []string{"//go:build linux", "//go:build amd64"})
	if err != nil {
		t.Fatalf("GenerateFileHeader failed: %v", err)
	}

	if !strings.Contains(result, "//go:build linux") {
		t.Error("Result should contain first build tag")
	}
	if !strings.Contains(result, "//go:build amd64") {
		t.Error("Result should contain second build tag")
	}
}

func TestGenerateEncodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	typeInfo := TypeInfo{
		Name: "TestStruct",
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint32", XDRType: "uint32"},
			{Name: "Name", Type: "string", XDRType: "string"},
		},
	}

	result, err := cg.GenerateEncodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateEncodeMethod failed: %v", err)
	}

	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain struct name")
	}
	if !strings.Contains(result, "Encode") {
		t.Error("Result should contain Encode method")
	}
}

func TestGenerateDecodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	typeInfo := TypeInfo{
		Name: "TestStruct",
		Fields: []FieldInfo{
			{Name: "ID", Type: "uint32", XDRType: "uint32"},
			{Name: "Name", Type: "string", XDRType: "string"},
		},
	}

	result, err := cg.GenerateDecodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateDecodeMethod failed: %v", err)
	}

	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain struct name")
	}
	if !strings.Contains(result, "Decode") {
		t.Error("Result should contain Decode method")
	}
}

func TestGenerateAssertion(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	result, err := cg.GenerateAssertion("TestStruct")
	if err != nil {
		t.Fatalf("GenerateAssertion failed: %v", err)
	}

	if !strings.Contains(result, "TestStruct") {
		t.Error("Result should contain type name")
	}
	if !strings.Contains(result, "xdr.Codec") {
		t.Error("Result should contain interface name")
	}
}

func TestGenerateBasicEncodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

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
			if err != nil {
				t.Fatalf("generateBasicEncodeCode failed: %v", err)
			}

			if !strings.Contains(result, tt.field.Name) {
				t.Error("Result should contain field name")
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Result should contain %s", tt.expected)
			}
		})
	}
}

func TestGenerateBasicDecodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

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
			if err != nil {
				t.Fatalf("generateBasicDecodeCode failed: %v", err)
			}

			if !strings.Contains(result, tt.field.Name) {
				t.Error("Result should contain field name")
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Result should contain %s", tt.expected)
			}
		})
	}
}

func TestGenerateArrayEncodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	field := FieldInfo{Name: "Items", Type: "[]string", XDRType: "array"}

	result, err := cg.generateBasicEncodeCode(field)
	if err != nil {
		t.Fatalf("generateBasicEncodeCode failed: %v", err)
	}

	if !strings.Contains(result, "Items") {
		t.Error("Result should contain field name")
	}
	if !strings.Contains(result, "EncodeUint32") {
		t.Error("Result should contain array length encoding")
	}
}

func TestGenerateArrayDecodeCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	field := FieldInfo{Name: "Items", Type: "[]string", XDRType: "array"}

	result, err := cg.generateBasicDecodeCode(field)
	if err != nil {
		t.Fatalf("generateBasicDecodeCode failed: %v", err)
	}

	if !strings.Contains(result, "Items") {
		t.Error("Result should contain field name")
	}
	if !strings.Contains(result, "DecodeUint32") {
		t.Error("Result should contain array length decoding")
	}
}

func TestGetEncodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

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
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetDecodeMethod(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

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
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateUnsupportedXDRType(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	field := FieldInfo{Name: "Bad", Type: "float64", XDRType: "unsupported"}

	_, err = cg.generateBasicEncodeCode(field)
	if err == nil {
		t.Error("Expected error for unsupported XDR type")
	}

	_, err = cg.generateBasicDecodeCode(field)
	if err == nil {
		t.Error("Expected error for unsupported XDR type")
	}
}

func TestGenerateFixedBytesCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	// Test that fixed bytes field generates appropriate code
	field := FieldInfo{Name: "Hash", Type: "[16]byte", XDRType: "fixed:16"}

	encodeResult, err := cg.generateBasicEncodeCode(field)
	if err != nil {
		t.Fatalf("generateBasicEncodeCode failed: %v", err)
	}

	decodeResult, err := cg.generateBasicDecodeCode(field)
	if err != nil {
		t.Fatalf("generateBasicDecodeCode failed: %v", err)
	}

	if !strings.Contains(encodeResult, "EncodeFixedBytes") {
		t.Error("Encode result should contain EncodeFixedBytes")
	}
	if !strings.Contains(decodeResult, "DecodeFixedBytes") {
		t.Error("Decode result should contain DecodeFixedBytes")
	}
}

func TestGenerateAliasCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

	tests := []struct {
		name     string
		field    FieldInfo
		expected string
	}{
		{
			name:     "string alias",
			field:    FieldInfo{Name: "ID", Type: "UserID", XDRType: "alias:string"},
			expected: "EncodeString",
		},
		{
			name:     "bytes alias",
			field:    FieldInfo{Name: "Data", Type: "SessionID", XDRType: "alias:[]byte"},
			expected: "EncodeBytes",
		},
		{
			name:     "fixed bytes alias",
			field:    FieldInfo{Name: "Hash", Type: "Hash16", XDRType: "alias:[16]byte"},
			expected: "EncodeFixedBytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encodeResult, err := cg.generateBasicEncodeCode(tt.field)
			if err != nil {
				t.Fatalf("generateBasicEncodeCode failed: %v", err)
			}

			if !strings.Contains(encodeResult, tt.expected) {
				t.Errorf("Expected %s in encode result", tt.expected)
			}

			decodeResult, err := cg.generateBasicDecodeCode(tt.field)
			if err != nil {
				t.Fatalf("generateBasicDecodeCode failed: %v", err)
			}

			// For fixed bytes aliases, the decode code uses a special template
			if strings.Contains(tt.field.XDRType, "fixed:") {
				if !strings.Contains(decodeResult, "DecodeFixedBytes") {
					t.Errorf("Expected DecodeFixedBytes in decode result for fixed bytes alias")
				}
			} else {
				decodeExpected := strings.Replace(tt.expected, "Encode", "Decode", 1)
				if !strings.Contains(decodeResult, decodeExpected) {
					t.Errorf("Expected %s in decode result", decodeExpected)
				}
			}
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
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNewTemplateManagerError(t *testing.T) {
	// This test would require mocking template parsing to fail
	// For now, just test that NewTemplateManager works with valid templates
	tm, err := NewTemplateManager()
	if err != nil {
		t.Fatalf("NewTemplateManager failed: %v", err)
	}
	if tm == nil {
		t.Error("NewTemplateManager should return non-nil manager")
	}
}

func TestTemplateManagerInitError(t *testing.T) {
	// This would require mocking template parsing to fail
	// For now, just test that NewCodeGenerator works with valid templates
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}
	if cg == nil {
		t.Error("NewCodeGenerator should return non-nil generator")
	}
}

func TestGenerateUnionCode(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

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
	if err != nil {
		t.Fatalf("GenerateEncodeMethod failed: %v", err)
	}

	if !strings.Contains(encodeResult, "switch") {
		t.Error("Encode result should contain switch statement for union")
	}

	// Test decode method generation
	decodeResult, err := cg.GenerateDecodeMethod(typeInfo)
	if err != nil {
		t.Fatalf("GenerateDecodeMethod failed: %v", err)
	}

	if !strings.Contains(decodeResult, "switch") {
		t.Error("Decode result should contain switch statement for union")
	}
}

func TestGenerateUnionEncodeDecodeCodeErrors(t *testing.T) {
	cg, err := NewCodeGenerator([]string{})
	if err != nil {
		t.Fatalf("NewCodeGenerator failed: %v", err)
	}

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
		if err == nil || err.Error() != errMsg {
			t.Errorf("expected error %q, got %v", errMsg, err)
		}
		_, err = cg.generateUnionDecodeCode(structInfo.Fields[0], structInfo)
		if err == nil || err.Error() != errMsg {
			t.Errorf("expected error %q, got %v", errMsg, err)
		}
	})

	// Missing union config
	t.Run("missing union config", func(t *testing.T) {
		structInfo := TypeInfo{
			Name: "NoUnionConfig",
			Fields: []FieldInfo{
				{Name: "Type", Type: "uint32", XDRType: "uint32", IsKey: true},
				{Name: "Data", Type: "[]byte", XDRType: "union", IsUnion: true},
			},
			IsDiscriminatedUnion: true,
			UnionConfig:          nil,
		}
		errMsg := "no union configuration found for struct NoUnionConfig"
		_, err := cg.generateUnionEncodeCode(structInfo.Fields[1], structInfo)
		if err == nil || err.Error() != errMsg {
			t.Errorf("expected error %q, got %v", errMsg, err)
		}
		_, err = cg.generateUnionDecodeCode(structInfo.Fields[1], structInfo)
		if err == nil || err.Error() != errMsg {
			t.Errorf("expected error %q, got %v", errMsg, err)
		}
	})
}
