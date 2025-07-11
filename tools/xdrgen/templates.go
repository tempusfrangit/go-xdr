package main

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// Template data structures
type FileData struct {
	SourceFile      string
	Timestamp       string
	PackageName     string
	ExternalImports []string
	Types           []TypeData
	BuildTags       []string
}

type TypeData struct {
	TypeName string
	Fields   []FieldData
}

type FieldData struct {
	FieldName         string
	FieldType         string
	ElementType       string
	EncodeCode        string
	DecodeCode        string
	Method            string
	VarName           string
	DiscriminantField string
	Cases             []UnionCaseData
	HasDefaultCase    bool
	ElementEncodeCode string
	ElementDecodeCode string
	// Alias-specific fields
	UnderlyingType string
	AliasType      string
	EncodeMethod   string
	DecodeMethod   string
}

type UnionCaseData struct {
	CaseLabels string
	EncodeCode string
	DecodeCode string
	IsVoid     bool
}

// Template functions
var templateFuncs = template.FuncMap{
	"toLowerCase": func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToLower(s[:1]) + s[1:]
	},
	"trimPrefix": strings.TrimPrefix,
}

// Template manager
type TemplateManager struct {
	templates map[string]*template.Template
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() (*TemplateManager, error) {
	tm := &TemplateManager{
		templates: make(map[string]*template.Template),
	}

	// Load all template files
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
			name := strings.TrimSuffix(entry.Name(), ".tmpl")
			content, err := templateFS.ReadFile("templates/" + entry.Name())
			if err != nil {
				return nil, err
			}

			tmpl, err := template.New(name).Funcs(templateFuncs).Parse(string(content))
			if err != nil {
				return nil, err
			}

			tm.templates[name] = tmpl
		}
	}

	return tm, nil
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
	err := tmpl.Execute(&buf, data)
	return buf.String(), err
}

// Global template manager instance
var templateManager *TemplateManager

// Initialize templates
func init() {
	var err error
	templateManager, err = NewTemplateManager()
	if err != nil {
		panic(err)
	}
}

// Template-based code generation functions

// GenerateFileHeader generates the file header using templates
func GenerateFileHeader(sourceFile, packageName string, externalImports []string, buildTags []string) (string, error) {
	data := FileData{
		SourceFile:      sourceFile,
		Timestamp:       time.Now().Format(time.RFC3339),
		PackageName:     packageName,
		ExternalImports: externalImports,
		BuildTags:       buildTags,
	}
	return templateManager.ExecuteTemplate("file_header", data)
}

// GenerateEncodeMethod generates the encode method using templates
func GenerateEncodeMethod(typeInfo TypeInfo) (string, error) {
	// Convert fields to template data
	var fields []FieldData
	for _, field := range typeInfo.Fields {
		fieldData := FieldData{
			FieldName: field.Name,
			FieldType: field.Type,
		}

		// Generate field-specific encode code
		if field.UnionSpec != "" {
			// Union field
			encodeCode, err := generateUnionEncodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.EncodeCode = encodeCode
		} else if field.IsDiscriminant {
			// Discriminant field
			encodeCode, err := generateBasicEncodeCode(field)
			if err != nil {
				return "", err
			}
			fieldData.EncodeCode = encodeCode
		} else {
			// Regular field
			encodeCode, err := generateBasicEncodeCode(field)
			if err != nil {
				return "", err
			}
			fieldData.EncodeCode = encodeCode
		}

		fields = append(fields, fieldData)
	}

	data := TypeData{
		TypeName: typeInfo.Name,
		Fields:   fields,
	}
	return templateManager.ExecuteTemplate("encode_method", data)
}

// GenerateDecodeMethod generates the decode method using templates
func GenerateDecodeMethod(typeInfo TypeInfo) (string, error) {
	// Convert fields to template data
	var fields []FieldData
	for _, field := range typeInfo.Fields {
		fieldData := FieldData{
			FieldName: field.Name,
			FieldType: field.Type,
		}

		// Generate field-specific decode code
		if field.UnionSpec != "" {
			// Union field
			decodeCode, err := generateUnionDecodeCode(field, typeInfo)
			if err != nil {
				return "", err
			}
			fieldData.DecodeCode = decodeCode
		} else if field.IsDiscriminant {
			// Discriminant field
			decodeCode, err := generateBasicDecodeCode(field)
			if err != nil {
				return "", err
			}
			fieldData.DecodeCode = decodeCode
		} else {
			// Regular field
			decodeCode, err := generateBasicDecodeCode(field)
			if err != nil {
				return "", err
			}
			fieldData.DecodeCode = decodeCode
		}

		fields = append(fields, fieldData)
	}

	data := TypeData{
		TypeName: typeInfo.Name,
		Fields:   fields,
	}
	return templateManager.ExecuteTemplate("decode_method", data)
}

// GenerateAssertion generates the compile-time assertion using templates
func GenerateAssertion(typeName string) (string, error) {
	data := TypeData{
		TypeName: typeName,
	}
	return templateManager.ExecuteTemplate("assertion", data)
}

// Helper functions for generating field-specific code

// generateBasicEncodeCode generates basic encode code for a field
func generateBasicEncodeCode(field FieldInfo) (string, error) {
	// Handle struct types specially
	if field.XDRType == "struct" {
		data := FieldData{
			FieldName: field.Name,
		}
		return templateManager.ExecuteTemplate("field_encode_struct", data)
	}

	// Handle array types specially
	if field.XDRType == "array" {
		elementType := strings.TrimPrefix(field.Type, "[]")
		data := FieldData{
			FieldName:   field.Name,
			FieldType:   field.Type,
			ElementType: elementType,
		}
		return templateManager.ExecuteTemplate("array_encode", data)
	}

	// Handle alias types specially
	if strings.HasPrefix(field.XDRType, "alias:") {
		// Extract the underlying type from the alias specification
		underlyingType := strings.TrimPrefix(field.XDRType, "alias:")
		if underlyingType == "[]byte" {
			underlyingType = "bytes"
		}
		encodeMethod := getEncodeMethod(underlyingType)
		if encodeMethod == "" {
			return "", fmt.Errorf("unsupported underlying type for alias: %s", underlyingType)
		}
		data := FieldData{
			FieldName:      field.Name,
			UnderlyingType: goTypeForXDRType(underlyingType),
			EncodeMethod:   encodeMethod,
		}
		return templateManager.ExecuteTemplate("field_encode_alias", data)
	}

	method := getEncodeMethod(field.XDRType)
	if method == "" {
		return "", fmt.Errorf("unsupported XDR type for encoding: %s", field.XDRType)
	}

	data := FieldData{
		FieldName: field.Name,
		Method:    method,
	}
	return templateManager.ExecuteTemplate("field_encode_basic", data)
}

// generateBasicDecodeCode generates basic decode code for a field
func generateBasicDecodeCode(field FieldInfo) (string, error) {
	// Handle struct types specially
	if field.XDRType == "struct" {
		data := FieldData{
			FieldName: field.Name,
		}
		return templateManager.ExecuteTemplate("field_decode_struct", data)
	}

	// Handle array types specially
	if field.XDRType == "array" {
		elementType := strings.TrimPrefix(field.Type, "[]")
		data := FieldData{
			FieldName:   field.Name,
			FieldType:   field.Type,
			ElementType: elementType,
		}
		return templateManager.ExecuteTemplate("array_decode", data)
	}

	// Handle alias types specially
	if strings.HasPrefix(field.XDRType, "alias:") {
		// Extract the underlying type from the alias specification
		underlyingType := strings.TrimPrefix(field.XDRType, "alias:")
		if underlyingType == "[]byte" {
			underlyingType = "bytes"
		}
		decodeMethod := getDecodeMethod(underlyingType)
		if decodeMethod == "" {
			return "", fmt.Errorf("unsupported underlying type for alias: %s", underlyingType)
		}
		data := FieldData{
			FieldName:      field.Name,
			UnderlyingType: goTypeForXDRType(underlyingType),
			AliasType:      field.Type,
			DecodeMethod:   decodeMethod,
		}
		return templateManager.ExecuteTemplate("field_decode_alias", data)
	}

	method := getDecodeMethod(field.XDRType)
	if method == "" {
		return "", fmt.Errorf("unsupported XDR type for decoding: %s", field.XDRType)
	}

	data := FieldData{
		FieldName: field.Name,
		Method:    method,
		VarName:   field.Name,
	}
	return templateManager.ExecuteTemplate("field_decode_basic", data)
}

// generateUnionEncodeCode generates union encode code for a field
func generateUnionEncodeCode(field FieldInfo, structInfo TypeInfo) (string, error) {
	discriminantField := findDiscriminantField(structInfo)
	if discriminantField == "" {
		return "", fmt.Errorf("no discriminant field found for union field %s", field.Name)
	}

	cases := parseUnionSpec(field.UnionSpec)
	var templateCases []UnionCaseData

	for _, unionCase := range cases {
		var caseLabels []string
		for _, value := range unionCase.Values {
			if value == "default" {
				caseLabels = append(caseLabels, "default")
			} else {
				caseLabels = append(caseLabels, "case "+value)
			}
		}

		var encodeCode string
		var err error

		if unionCase.IsVoid {
			encodeCode, err = templateManager.ExecuteTemplate("union_case_void", nil)
		} else {
			// Generate appropriate encode code based on field type
			if field.Type == "[]byte" {
				encodeCode, err = templateManager.ExecuteTemplate("union_case_bytes_encode", FieldData{
					FieldName: field.Name,
				})
			} else {
				encodeCode, err = templateManager.ExecuteTemplate("union_case_struct_encode", FieldData{
					FieldName: field.Name,
				})
			}
		}

		if err != nil {
			return "", err
		}

		templateCases = append(templateCases, UnionCaseData{
			CaseLabels: strings.Join(caseLabels, ":\n\t"),
			EncodeCode: encodeCode,
			IsVoid:     unionCase.IsVoid,
		})
	}

	data := FieldData{
		FieldName:         field.Name,
		DiscriminantField: discriminantField,
		Cases:             templateCases,
		HasDefaultCase:    hasDefaultCase(cases),
	}
	return templateManager.ExecuteTemplate("union_encode", data)
}

// generateUnionDecodeCode generates union decode code for a field
func generateUnionDecodeCode(field FieldInfo, structInfo TypeInfo) (string, error) {
	discriminantField := findDiscriminantField(structInfo)
	if discriminantField == "" {
		return "", fmt.Errorf("no discriminant field found for union field %s", field.Name)
	}

	cases := parseUnionSpec(field.UnionSpec)
	var templateCases []UnionCaseData

	for _, unionCase := range cases {
		var caseLabels []string
		for _, value := range unionCase.Values {
			if value == "default" {
				caseLabels = append(caseLabels, "default")
			} else {
				caseLabels = append(caseLabels, "case "+value)
			}
		}

		var decodeCode string
		var err error

		if unionCase.IsVoid {
			decodeCode, err = templateManager.ExecuteTemplate("union_case_void", nil)
		} else {
			// Generate appropriate decode code based on field type
			if field.Type == "[]byte" {
				decodeCode, err = templateManager.ExecuteTemplate("union_case_bytes_decode", FieldData{
					FieldName: field.Name,
				})
			} else {
				decodeCode, err = templateManager.ExecuteTemplate("union_case_struct_decode", FieldData{
					FieldName: field.Name,
					FieldType: strings.TrimPrefix(field.Type, "*"),
				})
			}
		}

		if err != nil {
			return "", err
		}

		templateCases = append(templateCases, UnionCaseData{
			CaseLabels: strings.Join(caseLabels, ":\n\t"),
			DecodeCode: decodeCode,
			IsVoid:     unionCase.IsVoid,
		})
	}

	data := FieldData{
		FieldName:         field.Name,
		DiscriminantField: discriminantField,
		Cases:             templateCases,
		HasDefaultCase:    hasDefaultCase(cases),
	}
	return templateManager.ExecuteTemplate("union_decode", data)
}

// getEncodeMethod returns the appropriate encoder method for an XDR type
func getEncodeMethod(xdrType string) string {
	switch xdrType {
	case "uint32", "discriminant":
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
		return ""
	}
}

// goTypeForXDRType converts XDR type names to Go type names
func goTypeForXDRType(xdrType string) string {
	switch xdrType {
	case "string":
		return "string"
	case "bytes":
		return "[]byte"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "bool":
		return "bool"
	default:
		panic("unsupported alias type: " + xdrType)
	}
}

// getDecodeMethod returns the appropriate decoder method for an XDR type
func getDecodeMethod(xdrType string) string {
	switch xdrType {
	case "uint32", "discriminant":
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
		return ""
	}
}

// Array element generation is now handled directly in the complete array templates
