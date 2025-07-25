package main

import (
	"embed"
	"fmt"
	"go/ast"
	"sort"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// FileData represents data for file header templates
type FileData struct {
	SourceFile      string
	PackageName     string
	ExternalImports []string
	Types           []TypeData
	BuildTags       []string
	TypeCount       int
}

// TypeData represents data for type templates
type TypeData struct {
	TypeName     string
	Fields       []FieldData
	CanHaveLoops bool // from static analysis
}

// FieldData represents data for field templates
type FieldData struct {
	FieldName                 string
	FieldType                 string
	ElementType               string
	ResolvedElementType       string // Resolved type for array elements (e.g., NFSStatus -> uint32)
	ElementIsStruct           bool
	ElementIsPointer          bool   // true if ElementType is a pointer (starts with *)
	ElementTypeWithoutPointer string // ElementType with * stripped off
	IsPointer                 bool   // true if FieldType is a pointer (for field_decode_struct)
	TypeWithoutPointer        string // FieldType with * stripped off (for field_decode_struct)
	EncodeCode                string
	DecodeCode                string
	Method                    string
	VarName                   string
	DiscriminantField         string
	Cases                     []UnionCaseData
	HasDefaultCase            bool
	ParentHasLoops            bool // true if the containing struct needs loop detection
	// Alias-specific fields
	UnderlyingType string
	AliasType      string
	EncodeMethod   string
	DecodeMethod   string
	// Type conversion fields
	TypeConversion    string
	TypeConversionEnd string
}

// UnionCaseData represents data for union case templates
type UnionCaseData struct {
	CaseLabels string
	EncodeCode string
	DecodeCode string
	IsVoid     bool
}

// TemplateManager manages Go templates for code generation
type TemplateManager struct {
	templates map[string]*template.Template
}

// CodeGenerator handles XDR code generation with embedded template manager
// Add structTypes field to hold set of struct type names
type CodeGenerator struct {
	tm           *TemplateManager
	structTypes  map[string]bool
	typeAliases  map[string]string // Maps type names to their underlying types
	usedPackages map[string]bool   // Track packages actually used in generated code
}

// NewCodeGenerator creates a new code generator with initialized templates
// Accepts a list of struct type names and type aliases
func NewCodeGenerator(structTypeNames []string, typeAliases map[string]string) (*CodeGenerator, error) {
	tm, err := NewTemplateManager()
	if err != nil {
		return nil, err
	}
	structTypes := make(map[string]bool)
	for _, name := range structTypeNames {
		structTypes[name] = true
	}
	if typeAliases == nil {
		typeAliases = make(map[string]string)
	}
	return &CodeGenerator{tm: tm, structTypes: structTypes, typeAliases: typeAliases, usedPackages: make(map[string]bool)}, nil
}

// GetUsedPackages returns the list of packages actually used in generated code
func (cg *CodeGenerator) GetUsedPackages() []string {
	var packages []string
	for pkg := range cg.usedPackages {
		packages = append(packages, pkg)
	}
	debugf("GetUsedPackages: tracked packages: %v", packages)
	return packages
}

// trackPackageUsage extracts and tracks packages used in type references
func (cg *CodeGenerator) trackPackageUsage(typeRef string) {
	if strings.Contains(typeRef, ".") {
		parts := strings.Split(typeRef, ".")
		if len(parts) == 2 {
			packageName := parts[0]
			cg.usedPackages[packageName] = true
		}
	}
}

// NewTemplateManager creates a new template manager with all required templates
func NewTemplateManager() (*TemplateManager, error) {
	templates := make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"toLowerCase": strings.ToLower,
		"trimPrefix":  strings.TrimPrefix,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,
		"ne":          func(a, b string) bool { return a != b },
	}

	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		path := "templates/" + entry.Name()
		content, err := templateFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}
		tmpl, err := template.New(name).Funcs(funcMap).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", path, err)
		}
		templates[name] = tmpl
	}

	// Validate all templates by executing with empty data
	var broken []string
	for name, tmpl := range templates {
		var dummy any
		// Create appropriate dummy data based on template name
		switch name {
		case "file_header":
			dummy = FileData{
				SourceFile:      "test.go",
				PackageName:     "test",
				ExternalImports: []string{},
				BuildTags:       []string{},
				TypeCount:       5,
			}
		case "encode_method", "decode_method":
			dummy = TypeData{
				TypeName:     "TestType",
				Fields:       []FieldData{},
				CanHaveLoops: false,
			}
		case "assertion":
			dummy = TypeData{
				TypeName:     "TestType",
				CanHaveLoops: false,
			}
		case "field_encode_basic", "field_decode_basic", "field_encode_struct", "field_decode_struct":
			dummy = FieldData{
				FieldName:          "TestField",
				FieldType:          "string",
				Method:             "EncodeString",
				VarName:            "tempTestField",
				TypeConversion:     "",
				TypeConversionEnd:  "",
				IsPointer:          false,
				TypeWithoutPointer: "string",
				ParentHasLoops:     false,
			}
		case "array_encode", "array_decode":
			dummy = FieldData{
				FieldName:                 "TestArray",
				FieldType:                 "[]string",
				ElementType:               "string",
				ElementIsStruct:           false,
				ElementIsPointer:          false,
				ElementTypeWithoutPointer: "string",
				ParentHasLoops:            false,
			}
		case "field_encode_alias", "field_decode_alias":
			dummy = FieldData{
				FieldName:      "TestAlias",
				UnderlyingType: "string",
				AliasType:      "TestAliasType",
				EncodeMethod:   "EncodeString",
				DecodeMethod:   "DecodeString",
			}
		case "union_encode", "union_decode":
			dummy = FieldData{
				FieldName:         "TestUnion",
				DiscriminantField: "TestKey",
				Cases:             []UnionCaseData{},
			}
		case "union_case_bytes_encode", "union_case_bytes_decode":
			dummy = FieldData{
				FieldName: "TestField",
				FieldType: "TestStruct",
			}
		case "union_case_struct_encode", "union_case_struct_decode":
			dummy = struct {
				PayloadTypeName string
				FieldName       string
			}{
				PayloadTypeName: "TestPayload",
				FieldName:       "TestField",
			}
		case "union_case_void":
			dummy = nil
		case "fixed_array_encode", "fixed_array_decode":
			dummy = FieldData{
				FieldName:       "TestArray",
				FieldType:       "[16]byte",
				ElementType:     "byte",
				ElementIsStruct: false,
				ParentHasLoops:  false,
			}
		case "fixed_bytes_encode", "fixed_bytes_decode":
			dummy = FieldData{
				FieldName:      "TestBytes",
				ParentHasLoops: false,
			}
		case "payload_to_union":
			dummy = struct {
				PayloadTypeName string
				UnionTypeName   string
				KeyField        string
				PayloadField    string
				Discriminant    string
			}{
				PayloadTypeName: "TestPayload",
				UnionTypeName:   "TestUnion",
				KeyField:        "Type",
				PayloadField:    "Payload",
				Discriminant:    "TestConstant",
			}
		case "payload_encode_to_union":
			dummy = struct {
				PayloadTypeName string
				Discriminant    string
				CanHaveLoops    bool
			}{
				PayloadTypeName: "TestPayload",
				Discriminant:    "TestConstant",
				CanHaveLoops:    false,
			}
		default:
			dummy = struct{}{}
		}

		if err := tmpl.Execute(&strings.Builder{}, dummy); err != nil {
			broken = append(broken, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(broken) > 0 {
		return nil, fmt.Errorf("template validation failed:\n%s", strings.Join(broken, "\n"))
	}

	return &TemplateManager{templates: templates}, nil
}

// GetTemplate returns a template by name
func (tm *TemplateManager) GetTemplate(name string) (*template.Template, bool) {
	tmpl, exists := tm.templates[name]
	return tmpl, exists
}

// ExecuteTemplate executes a template with the given data
func (tm *TemplateManager) ExecuteTemplate(name string, data any) (string, error) {
	tmpl, exists := tm.GetTemplate(name)
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// Template-based code generation functions

// GenerateFileHeader generates the file header using templates
func (cg *CodeGenerator) GenerateFileHeader(sourceFile, packageName string, externalImports []string, buildTags []string, typeCount int) (string, error) {
	data := FileData{
		SourceFile:      sourceFile,
		PackageName:     packageName,
		ExternalImports: externalImports,
		BuildTags:       buildTags,
		TypeCount:       typeCount,
	}
	return cg.tm.ExecuteTemplate("file_header", data)
}

// GenerateEncodeMethod generates the encode method using templates
func (cg *CodeGenerator) GenerateEncodeMethod(typeInfo TypeInfo) (string, error) {
	debugf("GenerateEncodeMethod: %s, CanHaveLoops=%t", typeInfo.Name, typeInfo.CanHaveLoops)
	// Convert fields to template data
	var fields []FieldData
	for _, field := range typeInfo.Fields {
		fieldData := FieldData{
			FieldName: field.Name,
			FieldType: field.Type,
		}

		// Generate field-specific encode code
		switch {
		case field.IsUnion:
			// Union field
			encodeCode, err := cg.generateUnionEncodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.EncodeCode = encodeCode
		case field.IsKey:
			// Key field
			encodeCode, err := cg.generateBasicEncodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.EncodeCode = encodeCode
		default:
			// Regular field
			encodeCode, err := cg.generateBasicEncodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.EncodeCode = encodeCode
		}

		fields = append(fields, fieldData)
	}

	data := TypeData{
		TypeName:     typeInfo.Name,
		Fields:       fields,
		CanHaveLoops: typeInfo.CanHaveLoops,
	}
	return cg.tm.ExecuteTemplate("encode_method", data)
}

// GenerateDecodeMethod generates the decode method using templates
func (cg *CodeGenerator) GenerateDecodeMethod(typeInfo TypeInfo) (string, error) {
	// Convert fields to template data
	var fields []FieldData
	for _, field := range typeInfo.Fields {
		fieldData := FieldData{
			FieldName: field.Name,
			FieldType: field.Type,
		}

		// Generate field-specific decode code
		switch {
		case field.IsUnion:
			// Union field
			decodeCode, err := cg.generateUnionDecodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.DecodeCode = decodeCode
		case field.IsKey:
			// Key field
			decodeCode, err := cg.generateBasicDecodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.DecodeCode = decodeCode
		default:
			// Regular field
			decodeCode, err := cg.generateBasicDecodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.DecodeCode = decodeCode
		}

		fields = append(fields, fieldData)
	}

	data := TypeData{
		TypeName:     typeInfo.Name,
		Fields:       fields,
		CanHaveLoops: typeInfo.CanHaveLoops,
	}
	return cg.tm.ExecuteTemplate("decode_method", data)
}

// GenerateAssertion generates the compile-time assertion using templates
func (cg *CodeGenerator) GenerateAssertion(typeName string) (string, error) {
	data := TypeData{
		TypeName: typeName,
	}
	return cg.tm.ExecuteTemplate("assertion", data)
}

// GeneratePayloadToUnion generates ToUnion method for payload types
func (cg *CodeGenerator) GeneratePayloadToUnion(typeInfo TypeInfo, typeDefs map[string]ast.Node) (string, error) {
	if typeInfo.PayloadConfig == nil {
		return "", fmt.Errorf("payload config is nil for type %s", typeInfo.Name)
	}

	// Find the actual union type's key and payload fields from the AST
	keyField, payloadField := getUnionFieldNames(typeInfo.PayloadConfig.UnionType, typeDefs)
	if keyField == "" || payloadField == "" {
		return "", fmt.Errorf("failed to find key field (%s) or payload field (%s) for union type %s", keyField, payloadField, typeInfo.PayloadConfig.UnionType)
	}

	data := struct {
		PayloadTypeName string
		UnionTypeName   string
		KeyField        string
		PayloadField    string
		Discriminant    string
	}{
		PayloadTypeName: typeInfo.Name,
		UnionTypeName:   typeInfo.PayloadConfig.UnionType,
		KeyField:        keyField,
		PayloadField:    payloadField,
		Discriminant:    typeInfo.PayloadConfig.Discriminant,
	}

	return cg.tm.ExecuteTemplate("payload_to_union", data)
}

// GeneratePayloadEncodeToUnion generates EncodeToUnion method for payload types
func (cg *CodeGenerator) GeneratePayloadEncodeToUnion(typeInfo TypeInfo) (string, error) {
	if typeInfo.PayloadConfig == nil {
		return "", fmt.Errorf("payload config is nil for type %s", typeInfo.Name)
	}

	data := struct {
		PayloadTypeName string
		Discriminant    string
		CanHaveLoops    bool
	}{
		PayloadTypeName: typeInfo.Name,
		Discriminant:    typeInfo.PayloadConfig.Discriminant,
		CanHaveLoops:    typeInfo.CanHaveLoops,
	}

	return cg.tm.ExecuteTemplate("payload_encode_to_union", data)
}

// Helper functions for generating field-specific code

// generateBasicEncodeCode generates basic encode code for a field
func (cg *CodeGenerator) generateBasicEncodeCode(field FieldInfo, typeInfo TypeInfo) (string, error) {
	// Special case: []byte with xdr:"bytes" should use bytes encoding, not array encoding
	// Check both Type and ResolvedType to handle aliases like SessionID which is []byte
	if (field.Type == "[]byte" || field.ResolvedType == "[]byte") && field.XDRType == "bytes" {
		method := cg.getEncodeMethod(field.XDRType)
		if method == "" {
			return "", fmt.Errorf("unsupported XDR type for encoding: %s", field.XDRType)
		}

		// Handle type conversions for alias types
		typeConversion := ""
		typeConversionEnd := ""
		expectedType := cg.getExpectedGoType(field.XDRType)
		if field.Type != expectedType {
			typeConversion = expectedType + "("
			typeConversionEnd = ")"
		}

		data := FieldData{
			FieldName:         field.Name,
			Method:            method,
			TypeConversion:    typeConversion,
			TypeConversionEnd: typeConversionEnd,
		}
		return cg.tm.ExecuteTemplate("field_encode_basic", data)
	}

	// Special case: [N]byte alias with xdr:"bytes" should use EncodeFixedBytes optimization
	// Check if ResolvedType is a fixed byte array, or if Type is a direct fixed byte array
	if field.XDRType == "bytes" {
		isFixedByteArray := false
		// Direct fixed byte array
		if strings.HasPrefix(field.Type, "[") && strings.Contains(field.Type, "]byte") {
			isFixedByteArray = true
		}
		// Alias to fixed byte array
		if field.ResolvedType != "" && strings.HasPrefix(field.ResolvedType, "[") && strings.Contains(field.ResolvedType, "]byte") {
			isFixedByteArray = true
		}
		if isFixedByteArray {
			return cg.tm.ExecuteTemplate("fixed_bytes_encode", FieldData{
				FieldName:      field.Name,
				ParentHasLoops: typeInfo.CanHaveLoops,
			})
		}
	}

	// Auto-detect array types from Go type syntax
	// Check both Type and ResolvedType to handle aliases like SessionID which is []byte
	if strings.HasPrefix(field.Type, "[]") || strings.HasPrefix(field.ResolvedType, "[]") {
		// Variable-length array: []Type with xdr:"elementXDRType"
		return cg.generateVariableArrayEncodeCode(field, typeInfo)
	}

	if (strings.HasPrefix(field.Type, "[") && strings.Contains(field.Type, "]")) ||
		(strings.HasPrefix(field.ResolvedType, "[") && strings.Contains(field.ResolvedType, "]")) {
		// Fixed-length array: [N]Type with xdr:"elementXDRType"
		return cg.generateFixedArrayEncodeCode(field, typeInfo)
	}

	// Handle struct types specially
	if field.XDRType == "struct" {
		isPointer := strings.HasPrefix(field.Type, "*")
		typeWithoutPointer := strings.TrimPrefix(field.Type, "*")

		data := FieldData{
			FieldName:          field.Name,
			FieldType:          field.Type,
			IsPointer:          isPointer,
			TypeWithoutPointer: typeWithoutPointer,
			ParentHasLoops:     typeInfo.CanHaveLoops,
		}
		return cg.tm.ExecuteTemplate("field_encode_struct", data)
	}

	method := cg.getEncodeMethod(field.XDRType)
	if method == "" {
		return "", fmt.Errorf("unsupported XDR type for encoding: %s", field.XDRType)
	}

	// Handle type conversions for alias types
	typeConversion := ""
	typeConversionEnd := ""

	// Get the expected Go type for this XDR type
	expectedType := cg.getExpectedGoType(field.XDRType)
	if field.Type != expectedType {
		// Field type is an alias, need to convert to underlying type
		// Special case: alias to fixed array that maps to []byte needs slicing
		if expectedType == "[]byte" {
			// Check if the alias resolves to a fixed byte array vs slice
			// This handles cases like: type Hash [16]byte -> needs v.Hash[:]
			// But NOT: type SessionID []byte -> should use v.Session directly
			if strings.HasPrefix(field.ResolvedType, "[") && strings.Contains(field.ResolvedType, "]byte") {
				// Fixed array alias: type Hash [16]byte
				typeConversion = ""
				typeConversionEnd = "[:]"
			} else {
				// Slice alias: type SessionID []byte
				typeConversion = expectedType + "("
				typeConversionEnd = ")"
			}
		} else {
			typeConversion = expectedType + "("
			typeConversionEnd = ")"
		}
	}

	data := FieldData{
		FieldName:         field.Name,
		Method:            method,
		TypeConversion:    typeConversion,
		TypeConversionEnd: typeConversionEnd,
	}
	return cg.tm.ExecuteTemplate("field_encode_basic", data)
}

// generateVariableArrayEncodeCode generates encode code for []Type fields
func (cg *CodeGenerator) generateVariableArrayEncodeCode(field FieldInfo, typeInfo TypeInfo) (string, error) {
	elementType := strings.TrimPrefix(field.Type, "[]")

	// For arrays, we need to resolve the element type to determine the encoding method
	// First, try to resolve the original element type using package-level type aliases
	resolvedElementType := cg.resolveTypeAlias(elementType)

	// If that didn't resolve to a primitive, fall back to the ResolvedType from parser
	if !isPrimitiveType(resolvedElementType) && strings.HasPrefix(field.ResolvedType, "[]") {
		resolvedElementType = strings.TrimPrefix(field.ResolvedType, "[]")
	}

	// Determine if element is a struct based on resolved type
	// For pointer types like *Node, we need to strip the * to check the underlying type
	typeToCheck := strings.TrimPrefix(resolvedElementType, "*")
	elementIsStruct := !cg.isPrimitiveTypeWithAliases(typeToCheck)

	data := FieldData{
		FieldName:           field.Name,
		FieldType:           field.Type,
		ElementType:         elementType,
		ResolvedElementType: resolvedElementType,
		ElementIsStruct:     elementIsStruct,
		ParentHasLoops:      typeInfo.CanHaveLoops,
		// Use the XDR tag as the element encoding type
	}
	return cg.tm.ExecuteTemplate("array_encode", data)
}

// generateFixedArrayEncodeCode generates encode code for [N]Type fields
func (cg *CodeGenerator) generateFixedArrayEncodeCode(field FieldInfo, typeInfo TypeInfo) (string, error) {
	// Use ResolvedType if available, otherwise fall back to Type
	arrayType := field.ResolvedType
	if arrayType == "" {
		arrayType = field.Type
	}

	// Extract [N] and Type from [N]Type
	if !strings.Contains(arrayType, "]") {
		return "", fmt.Errorf("invalid fixed array type: %s", arrayType)
	}

	closeBracket := strings.Index(arrayType, "]")
	elementType := arrayType[closeBracket+1:] // Extract Type from [N]Type

	// Special case: [N]byte with xdr:"bytes" -> use EncodeFixedBytes optimization
	if elementType == "byte" && field.XDRType == "bytes" {
		return cg.tm.ExecuteTemplate("fixed_bytes_encode", FieldData{
			FieldName:      field.Name,
			ParentHasLoops: typeInfo.CanHaveLoops,
		})
	}

	// General case: [N]Type -> encode N elements individually
	elementIsStruct := cg.resolveAliasToStruct(elementType)
	data := FieldData{
		FieldName:       field.Name,
		FieldType:       field.Type,
		ElementType:     elementType,
		ElementIsStruct: elementIsStruct,
		ParentHasLoops:  typeInfo.CanHaveLoops,
	}
	return cg.tm.ExecuteTemplate("fixed_array_encode", data)
}

// generateBasicDecodeCode generates basic decode code for a field
func (cg *CodeGenerator) generateBasicDecodeCode(field FieldInfo, typeInfo TypeInfo) (string, error) {
	// Special case: []byte with xdr:"bytes" should use bytes decoding, not array decoding
	if (field.Type == "[]byte" || field.ResolvedType == "[]byte") && field.XDRType == "bytes" {
		method := cg.getDecodeMethod(field.XDRType)
		if method == "" {
			return "", fmt.Errorf("unsupported XDR type for decoding: %s", field.XDRType)
		}

		// Handle type conversions for alias types
		typeConversion := ""
		typeConversionEnd := ""
		expectedType := cg.getExpectedGoType(field.XDRType)
		if field.Type != expectedType {
			typeConversion = field.Type + "("
			typeConversionEnd = ")"
		}

		data := FieldData{
			FieldName:         field.Name,
			Method:            method,
			VarName:           "temp" + field.Name,
			TypeConversion:    typeConversion,
			TypeConversionEnd: typeConversionEnd,
		}
		return cg.tm.ExecuteTemplate("field_decode_basic", data)
	}

	// Special case: [N]byte alias with xdr:"bytes" should use DecodeFixedBytesInto optimization
	// Check if ResolvedType is a fixed byte array ([N]byte) but NOT variable array ([]byte)
	if field.XDRType == "bytes" && field.ResolvedType != "" &&
		strings.HasPrefix(field.ResolvedType, "[") && !strings.HasPrefix(field.ResolvedType, "[]") && strings.Contains(field.ResolvedType, "]byte") {
		return cg.tm.ExecuteTemplate("fixed_bytes_decode", FieldData{
			FieldName:      field.Name,
			ParentHasLoops: typeInfo.CanHaveLoops,
		})
	}

	// Auto-detect array types from Go type syntax
	// Check both Type and ResolvedType to handle aliases like SessionID which is []byte
	if strings.HasPrefix(field.Type, "[]") || strings.HasPrefix(field.ResolvedType, "[]") {
		// Variable-length array: []Type with xdr:"elementXDRType"
		return cg.generateVariableArrayDecodeCode(field, typeInfo)
	}

	if (strings.HasPrefix(field.Type, "[") && strings.Contains(field.Type, "]")) ||
		(strings.HasPrefix(field.ResolvedType, "[") && strings.Contains(field.ResolvedType, "]")) {
		// Fixed-length array: [N]Type with xdr:"elementXDRType"
		return cg.generateFixedArrayDecodeCode(field, typeInfo)
	}

	// Handle struct types specially
	if field.XDRType == "struct" {
		// Note: We don't track package usage for struct types because they use method calls
		// like v.Field.Decode(dec) which don't require package prefixes
		isPointer := strings.HasPrefix(field.Type, "*")
		typeWithoutPointer := strings.TrimPrefix(field.Type, "*")

		data := FieldData{
			FieldName:          field.Name,
			FieldType:          field.Type,
			IsPointer:          isPointer,
			TypeWithoutPointer: typeWithoutPointer,
		}
		return cg.tm.ExecuteTemplate("field_decode_struct", data)
	}

	method := cg.getDecodeMethod(field.XDRType)
	if method == "" {
		return "", fmt.Errorf("unsupported XDR type for decoding: %s", field.XDRType)
	}

	// Handle type conversions for alias types
	typeConversion := ""
	typeConversionEnd := ""

	// Get the expected Go type for this XDR type
	expectedType := cg.getExpectedGoType(field.XDRType)
	if field.Type != expectedType {
		// Field type is an alias, need to convert from underlying type
		typeConversion = field.Type + "("
		typeConversionEnd = ")"
		// Track package usage for cross-package types
		cg.trackPackageUsage(field.Type)
	}

	data := FieldData{
		FieldName:         field.Name,
		FieldType:         field.Type,
		Method:            method,
		VarName:           "temp" + field.Name,
		TypeConversion:    typeConversion,
		TypeConversionEnd: typeConversionEnd,
	}
	return cg.tm.ExecuteTemplate("field_decode_basic", data)
}

// generateVariableArrayDecodeCode generates decode code for []Type fields
func (cg *CodeGenerator) generateVariableArrayDecodeCode(field FieldInfo, typeInfo TypeInfo) (string, error) {
	elementType := strings.TrimPrefix(field.Type, "[]")

	// For arrays, we need to resolve the element type to determine the decoding method
	// First, try to resolve the original element type using package-level type aliases
	resolvedElementType := cg.resolveTypeAlias(elementType)

	// If that didn't resolve to a primitive, fall back to the ResolvedType from parser
	if !isPrimitiveType(resolvedElementType) && strings.HasPrefix(field.ResolvedType, "[]") {
		resolvedElementType = strings.TrimPrefix(field.ResolvedType, "[]")
	}

	// Determine if element is a struct based on resolved type
	// For pointer types like *Node, we need to strip the * to check the underlying type
	typeToCheck := strings.TrimPrefix(resolvedElementType, "*")
	elementIsStruct := !cg.isPrimitiveTypeWithAliases(typeToCheck)

	elementIsPointer := strings.HasPrefix(elementType, "*")
	elementTypeWithoutPointer := strings.TrimPrefix(elementType, "*")

	data := FieldData{
		FieldName:                 field.Name,
		FieldType:                 field.Type,
		ElementType:               elementType,
		ResolvedElementType:       resolvedElementType,
		ElementIsStruct:           elementIsStruct,
		ElementIsPointer:          elementIsPointer,
		ElementTypeWithoutPointer: elementTypeWithoutPointer,
		ParentHasLoops:            typeInfo.CanHaveLoops,
		// Use the XDR tag as the element encoding type
	}
	return cg.tm.ExecuteTemplate("array_decode", data)
}

// generateFixedArrayDecodeCode generates decode code for [N]Type fields
func (cg *CodeGenerator) generateFixedArrayDecodeCode(field FieldInfo, typeInfo TypeInfo) (string, error) {
	// Use ResolvedType if available, otherwise fall back to Type
	arrayType := field.ResolvedType
	if arrayType == "" {
		arrayType = field.Type
	}

	// Extract [N] and Type from [N]Type
	if !strings.Contains(arrayType, "]") {
		return "", fmt.Errorf("invalid fixed array type: %s", arrayType)
	}

	closeBracket := strings.Index(arrayType, "]")
	elementType := arrayType[closeBracket+1:] // Extract Type from [N]Type

	// Special case: [N]byte with xdr:"bytes" -> use DecodeFixedBytesInto optimization
	if elementType == "byte" && field.XDRType == "bytes" {
		return cg.tm.ExecuteTemplate("fixed_bytes_decode", FieldData{
			FieldName:      field.Name,
			ParentHasLoops: typeInfo.CanHaveLoops,
		})
	}

	// General case: [N]Type -> decode N elements individually
	elementIsStruct := cg.resolveAliasToStruct(elementType)
	data := FieldData{
		FieldName:       field.Name,
		FieldType:       field.Type,
		ElementType:     elementType,
		ElementIsStruct: elementIsStruct,
		ParentHasLoops:  typeInfo.CanHaveLoops,
	}
	return cg.tm.ExecuteTemplate("fixed_array_decode", data)
}

// generateUnionEncodeCode generates union encode code for a field
func (cg *CodeGenerator) generateUnionEncodeCode(field FieldInfo, structInfo TypeInfo) (string, error) {
	keyField := findKeyField(structInfo)
	if keyField == "" {
		return "", fmt.Errorf("no key field found for union field %s", field.Name)
	}

	if structInfo.UnionConfig == nil {
		// Check if this is a void-only union (either explicit default=nil or no union config)
		if field.DefaultType == "nil" || structInfo.UnionConfig == nil {
			// Generate minimal void-only union encode code
			data := FieldData{
				FieldName:         field.Name,
				DiscriminantField: keyField,
				Cases:             []UnionCaseData{},
			}
			return cg.tm.ExecuteTemplate("union_encode", data)
		}
		return "", fmt.Errorf("no union configuration found for struct %s", structInfo.Name)
	}

	var templateCases []UnionCaseData

	// Generate cases for mapped constants (sorted for determinism)
	var constantValues []string
	for constantValue := range structInfo.UnionConfig.Cases {
		constantValues = append(constantValues, constantValue)
	}
	sort.Strings(constantValues)

	for _, constantValue := range constantValues {
		var encodeCode string
		var err error

		// Generate appropriate encode code based on field type
		if field.Type == "[]byte" {
			encodeCode, err = cg.tm.ExecuteTemplate("union_case_bytes_encode", FieldData{
				FieldName: field.Name,
			})
		} else {
			encodeCode, err = cg.tm.ExecuteTemplate("union_case_struct_encode", FieldData{
				FieldName: field.Name,
			})
		}

		if err != nil {
			return "", err
		}

		templateCases = append(templateCases, UnionCaseData{
			CaseLabels: "case " + constantValue,
			EncodeCode: encodeCode,
			IsVoid:     false,
		})
	}

	// Generate void cases
	for _, voidCase := range structInfo.UnionConfig.VoidCases {
		encodeCode, err := cg.tm.ExecuteTemplate("union_case_void", nil)
		if err != nil {
			return "", err
		}

		templateCases = append(templateCases, UnionCaseData{
			CaseLabels: "case " + voidCase,
			EncodeCode: encodeCode,
			IsVoid:     true,
		})
	}

	data := FieldData{
		FieldName:         field.Name,
		DiscriminantField: keyField,
		Cases:             templateCases,
	}
	return cg.tm.ExecuteTemplate("union_encode", data)
}

// generateUnionDecodeCode generates union decode code for a field
func (cg *CodeGenerator) generateUnionDecodeCode(field FieldInfo, structInfo TypeInfo) (string, error) {
	keyField := findKeyField(structInfo)
	if keyField == "" {
		return "", fmt.Errorf("no key field found for union field %s", field.Name)
	}

	if structInfo.UnionConfig == nil {
		// Check if this is a void-only union (either explicit default=nil or no union config)
		if field.DefaultType == "nil" || structInfo.UnionConfig == nil {
			// Generate minimal void-only union decode code
			data := FieldData{
				FieldName:         field.Name,
				DiscriminantField: keyField,
				Cases:             []UnionCaseData{},
			}
			return cg.tm.ExecuteTemplate("union_decode", data)
		}
		return "", fmt.Errorf("no union configuration found for struct %s", structInfo.Name)
	}

	var templateCases []UnionCaseData

	// Generate cases for mapped constants (sorted for determinism)
	var constantValues []string
	for constantValue := range structInfo.UnionConfig.Cases {
		constantValues = append(constantValues, constantValue)
	}
	sort.Strings(constantValues)

	for _, constantValue := range constantValues {
		var decodeCode string
		var err error

		// Generate appropriate decode code based on field type
		if field.Type == "[]byte" {
			decodeCode, err = cg.tm.ExecuteTemplate("union_case_bytes_decode", FieldData{
				FieldName: field.Name,
			})
		} else {
			decodeCode, err = cg.tm.ExecuteTemplate("union_case_struct_decode", FieldData{
				FieldName: field.Name,
				FieldType: strings.TrimPrefix(field.Type, "*"),
			})
		}

		if err != nil {
			return "", err
		}

		templateCases = append(templateCases, UnionCaseData{
			CaseLabels: "case " + constantValue,
			DecodeCode: decodeCode,
			IsVoid:     false,
		})
	}

	// Generate void cases
	for _, voidCase := range structInfo.UnionConfig.VoidCases {
		decodeCode, err := cg.tm.ExecuteTemplate("union_case_void", nil)
		if err != nil {
			return "", err
		}

		templateCases = append(templateCases, UnionCaseData{
			CaseLabels: "case " + voidCase,
			DecodeCode: decodeCode,
			IsVoid:     true,
		})
	}

	data := FieldData{
		FieldName:         field.Name,
		DiscriminantField: keyField,
		Cases:             templateCases,
	}
	return cg.tm.ExecuteTemplate("union_decode", data)
}

// getEncodeMethod returns the appropriate encoder method for an XDR type
func (cg *CodeGenerator) getEncodeMethod(xdrType string) string {
	switch xdrType {
	case "uint32":
		return "EncodeUint32"
	case "uint64":
		return "EncodeUint64"
	case "int32":
		return "EncodeInt32"
	case "int64":
		return "EncodeInt64"
	case "string":
		return "EncodeString"
	case "bytes":
		return "EncodeBytes"
	case "bool":
		return "EncodeBool"
	case "struct":
		return "Encode" // Delegate to struct's Encode method
	default:
		// Handle fixed-size byte arrays
		if strings.HasPrefix(xdrType, "fixed:") {
			return "EncodeFixedBytes"
		}
		// For unknown types, check if they resolve to a known primitive
		// This handles cross-package type aliases that should resolve to primitives
		if strings.Contains(xdrType, ".") {
			// Cross-package type - don't assume it implements Codec
			// Instead, this should be an error or the caller should provide more context
			// For now, return empty to indicate unsupported
			return ""
		}

		// Check if this is a verified struct type that we know should have Encode method
		if cg.structTypes[xdrType] {
			return "Encode"
		}
		return ""
	}
}

// getDecodeMethod returns the appropriate decoder method for an XDR type
func (cg *CodeGenerator) getDecodeMethod(xdrType string) string {
	switch xdrType {
	case "uint32":
		return "DecodeUint32"
	case "uint64":
		return "DecodeUint64"
	case "int32":
		return "DecodeInt32"
	case "int64":
		return "DecodeInt64"
	case "string":
		return "DecodeString"
	case "bytes":
		return "DecodeBytes"
	case "bool":
		return "DecodeBool"
	case "struct":
		return "Decode" // Delegate to struct's Decode method
	default:
		// Handle fixed-size byte arrays
		if strings.HasPrefix(xdrType, "fixed:") {
			return "DecodeFixedBytes"
		}
		// For unknown types, check if they resolve to a known primitive
		// This handles cross-package type aliases that should resolve to primitives
		if strings.Contains(xdrType, ".") {
			// Cross-package type - don't assume it implements Codec
			// Instead, this should be an error or the caller should provide more context
			// For now, return empty to indicate unsupported
			return ""
		}

		// Check if this is a verified struct type that we know should have Decode method
		if cg.structTypes[xdrType] {
			return "Decode"
		}
		return ""
	}
}

// isPrimitiveType checks if a type is a Go primitive that should use primitive XDR encoding
func isPrimitiveType(typeName string) bool {
	primitives := map[string]bool{
		"string": true, "[]byte": true, "uint32": true, "uint64": true,
		"int32": true, "int64": true, "bool": true, "byte": true,
	}
	return primitives[typeName]
}

// isPrimitiveTypeWithAliases checks if a type resolves to a primitive, following type aliases
func (cg *CodeGenerator) isPrimitiveTypeWithAliases(typeName string) bool {
	// Check direct primitive first
	if isPrimitiveType(typeName) {
		return true
	}

	// Follow type aliases to see if they resolve to primitives
	resolved := cg.resolveTypeAlias(typeName)
	return isPrimitiveType(resolved)
}

// resolveTypeAlias follows type aliases to their underlying type
func (cg *CodeGenerator) resolveTypeAlias(typeName string) string {
	seen := make(map[string]bool)
	current := typeName

	for !seen[current] {
		seen[current] = true

		alias, exists := cg.typeAliases[current]
		if !exists {
			break
		}
		current = alias
	}

	return current
}

// Helper to resolve aliases recursively for struct detection
func (cg *CodeGenerator) resolveAliasToStruct(typeName string) bool {
	// List of Go built-in types that are never structs
	builtins := map[string]bool{
		"string": true, "[]byte": true, "uint32": true, "uint64": true, "int32": true, "int64": true, "bool": true,
	}
	seen := map[string]bool{}
	for {
		if builtins[typeName] {
			return false
		}
		if cg.structTypes[typeName] {
			return true
		}
		if seen[typeName] {
			break // prevent cycles
		}
		seen[typeName] = true
		// If it's an alias, try to resolve
		if after, found := strings.CutPrefix(typeName, "alias:"); found {
			typeName = after
			continue
		}
		break
	}
	return false
}

// Array element generation is now handled directly in the complete array templates

// getExpectedGoType returns the expected Go type for a given XDR type
func (cg *CodeGenerator) getExpectedGoType(xdrType string) string {
	switch xdrType {
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "string":
		return "string"
	case "bytes":
		return "[]byte"
	case "bool":
		return "bool"
	case "auto":
		// For auto, we can't determine the expected type without more context
		// This should be resolved during parsing
		return ""
	default:
		// Handle fixed-size byte arrays
		if strings.HasPrefix(xdrType, "fixed:") {
			return "[]byte" // Fixed arrays are treated as []byte for encoding
		}
		// For struct types and other custom types, return the type as-is
		return xdrType
	}
}
