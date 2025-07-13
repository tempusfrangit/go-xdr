// XDR Code Generator
//
// This tool generates Encode and Decode methods for Go structs with XDR struct tags.
// It supports the xdr.Codec interface for automatic XDR serialization/deserialization.
//
// Usage:
//
//	go run cmd/xdrgen/main.go <input.go>
//
// The generator processes structs that are explicitly opted in via go:generate directives:
//
//	//go:generate xdrgen $GOFILE              (process all XDR-tagged structs in file)
//	//go:generate xdrgen StructName           (process specific struct)
//	//go:generate ../../tools/bin/xdrgen StructName  (with path)
//
// Supported XDR struct tags:
//
//	`xdr:"uint32"`    - 32-bit unsigned integer
//	`xdr:"uint64"`    - 64-bit unsigned integer
//	`xdr:"int64"`     - 64-bit signed integer
//	`xdr:"string"`    - variable-length string
//	`xdr:"bytes"`     - variable-length byte array
//	`xdr:"bool"`      - boolean (encoded as uint32)
//	`xdr:"alias"`     - type alias with automatic type inference
//	`xdr:"struct"`    - nested struct (must implement xdr.Codec)
//	`xdr:"array"`     - variable-length array (delegates to element type)
//	`xdr:"fixed:N"`   - fixed-size byte array (N bytes)
//	`xdr:"-"`         - exclude field from encoding/decoding
//
// Discriminated Union Support:
//
// Discriminated unions use key/union field pairs with comment-based configuration:
//
//	`xdr:"key"`       - marks the discriminant field (must be uint32 type)
//	`xdr:"union"`     - marks the union payload field (must immediately follow key)
//
// Union configuration is specified via comment directives:
//
//	//xdr:union=DiscriminantType,case=ConstantValue
//
// Example:
//
//	type Operation struct {
//	    OpCode OpCode `xdr:"key"`    // Discriminant field
//	    Result []byte `xdr:"union"`  // Union payload (must follow key)
//	}
//
//	//xdr:union=OpCode,case=OpOpen
//	type OpOpenResult struct {
//	    FileHandle []byte `xdr:"bytes"`
//	}
//
// Validation Rules:
// - Voids are implicit (constants without mapping comments)
// - Key field must be immediately followed by union field
// - Union case must match existing constant/type
// - All errors are aggregated (no fail-fast)
//
// Design principles:
// - Explicit opt-in: structs must have go:generate directive
// - Explicit tagging: all fields in opted-in structs must have XDR tags
// - No reflection in generated code: all encoding/decoding is static
// - Build-time validation: generates compile-time interface assertions
// - Clean imports: proper separation of standard library and module imports
//
// Example:
//
//	//go:generate xdrgen $GOFILE
//	type MyStruct struct {
//	    ID    uint32 `xdr:"uint32"`
//	    Name  string `xdr:"string"`
//	    Data  []byte `xdr:"bytes"`
//	    Skip  int    `xdr:"-"`       // excluded from XDR
//	}
//
// Generates:
//
//	func (v *MyStruct) Encode(enc *xdr.Encoder) error { ... }
//	func (v *MyStruct) Decode(dec *xdr.Decoder) error { ... }
//	var _ xdr.Codec = (*MyStruct)(nil)  // compile-time assertion
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var silent bool
var debug = false

// Set debug from environment variable if present
func init() {
	if os.Getenv("XDRGEN_DEBUG") != "" {
		debug = true
	}
}

// logf logs a message unless in silent mode
func logf(format string, args ...any) {
	if !silent {
		log.Printf(format, args...)
	}
}

// debugf logs a message only in debug mode
func debugf(format string, args ...any) {
	if debug {
		log.Printf("DEBUG: "+format, args...)
	}
}

// FieldInfo represents a struct field with XDR encoding information
type FieldInfo struct {
	Name        string
	Type        string
	XDRType     string
	Tag         string
	IsKey       bool   // true if this field is a discriminated union key
	IsUnion     bool   // true if this field is a discriminated union payload
	DefaultType string // default type from union tag (empty, "nil", or struct name)
}

// TypeInfo represents a struct that needs XDR generation
type TypeInfo struct {
	Name                 string
	Fields               []FieldInfo
	IsDiscriminatedUnion bool // true if this struct represents a discriminated union
	UnionConfig          *UnionConfig
}

// UnionConfig represents discriminated union configuration
type UnionConfig struct {
	ContainerType string            // e.g., "OperationResult"
	DefaultCase   string            // struct name for default case (if any)
	Cases         map[string]string // constant name -> struct name
	VoidCases     []string          // constant names that are void
}

// ValidationError represents a validation error
type ValidationError struct {
	Location string
	Message  string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Location, e.Message)
}

// parseXDRTag extracts XDR encoding information from struct tag
func parseXDRTag(tag string) string {
	if tag == "" {
		return ""
	}

	// Extract xdr:"..." from full tag
	parts := strings.Split(tag, `xdr:"`)
	if len(parts) < 2 {
		return ""
	}

	// Find closing quote
	xdrPart := strings.Split(parts[1], `"`)[0]
	return xdrPart
}

// parseKeyUnionTag parses key/union tags
func parseKeyUnionTag(xdrTag string) (isKey bool, isUnion bool, defaultType string) {
	if xdrTag == "key" {
		return true, false, ""
	}
	if xdrTag == "union" {
		return false, true, ""
	}
	if strings.HasPrefix(xdrTag, "union,") {
		// Parse union with options like "union,default=OpOpenResult"
		parts := strings.Split(xdrTag, ",")
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "default=") {
				defaultType = strings.TrimPrefix(part, "default=")
			}
		}
		return false, true, defaultType
	}
	return false, false, ""
}

// isSupportedXDRType checks if an XDR type is supported by the generator
func isSupportedXDRType(xdrType string) bool {
	switch xdrType {
	case "uint32", "uint64", "int64", "string", "bytes", "bool", "struct", "array", "key":
		return true
	default:
		return strings.HasPrefix(xdrType, "fixed:") || strings.HasPrefix(xdrType, "alias:")
	}
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// collectExternalImports collects external package imports from types
func collectExternalImports(types []TypeInfo) []string {
	var imports []string
	for _, typeInfo := range types {
		for _, field := range typeInfo.Fields {
			fieldImports := extractPackageImports(field.Type)
			imports = append(imports, fieldImports...)
		}
	}
	return imports
}

// extractPackageImports extracts package imports from a type string
// Handles cases like: "primitives.StateID", "[]primitives.StateID", "map[string]primitives.Type"
func extractPackageImports(typeStr string) []string {
	var imports []string
	// Find all package.Type patterns in the type string
	parts := strings.FieldsFunc(typeStr, func(r rune) bool {
		return r == '[' || r == ']' || r == '{' || r == '}' || r == '(' || r == ')' || r == '*' || r == ' ' || r == ','
	})

	for _, part := range parts {
		if strings.Contains(part, ".") {
			packageParts := strings.Split(part, ".")
			if len(packageParts) == 2 {
				packageName := packageParts[0]
				// Map known package names to their import paths
				switch packageName {
				case "primitives":
					imports = append(imports, "github.com/tempusfrangit/go-xdr/primitives")
				case "session":
					imports = append(imports, "github.com/tempusfrangit/go-xdr/session")
					// Add other packages as needed
				}
			}
		}
	}
	return imports
}

// findKeyField finds the key field name in the given struct
func findKeyField(structInfo TypeInfo) string {
	for _, field := range structInfo.Fields {
		if field.IsKey {
			return field.Name
		}
	}
	return ""
}

// parseUnionComment parses a union configuration comment
// Format: //xdr:union=DiscriminantType,case=ConstantValue
func parseUnionComment(comment string) (*UnionConfig, error) {
	if !strings.HasPrefix(comment, "//xdr:union=") {
		return nil, nil
	}

	// Extract the union specification
	spec := strings.TrimPrefix(comment, "//xdr:union=")
	parts := strings.Split(spec, ",")

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid union comment format: %s", comment)
	}

	config := &UnionConfig{
		Cases:     make(map[string]string),
		VoidCases: []string{},
	}

	// Parse container type
	containerPart := strings.TrimSpace(parts[0])
	if containerPart == "" {
		return nil, fmt.Errorf("missing container type in union comment: %s", comment)
	}
	config.ContainerType = containerPart

	// Parse cases
	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "case=") {
			// Parse case - just the constant name, struct name is implicit
			constantName := strings.TrimSpace(strings.TrimPrefix(part, "case="))
			if constantName == "" {
				return nil, fmt.Errorf("empty case name in union comment: %s", comment)
			}
			// The struct name will be filled in when we associate the comment with the struct
			config.Cases[constantName] = "" // Will be filled in later
		} else {
			return nil, fmt.Errorf("unknown union comment part: %s", comment)
		}
	}

	return config, nil
}

// validateUnionConfiguration validates discriminated union configuration
func validateUnionConfiguration(types []TypeInfo, constants map[string]string, typeDefs map[string]ast.Node, file *ast.File) []ValidationError {
	var errors []ValidationError

	// Helper to resolve type aliases recursively
	var resolveToStruct func(typeName string) (*ast.StructType, string)
	resolveToStruct = func(typeName string) (*ast.StructType, string) {
		node, ok := typeDefs[typeName]
		if !ok {
			return nil, "notype"
		}
		switch t := node.(type) {
		case *ast.StructType:
			return t, "struct"
		case *ast.Ident:
			// Alias to another type
			return resolveToStruct(t.Name)
		case *ast.SelectorExpr:
			// External type, treat as not a struct
			return nil, "external"
		default:
			return nil, fmt.Sprintf("%T", t)
		}
	}

	// Collect all union configurations by container type
	unionConfigs := make(map[string]*UnionConfig)
	defaultCount := 0

	for _, typeInfo := range types {
		if typeInfo.UnionConfig != nil {
			unionConfigs[typeInfo.UnionConfig.ContainerType] = typeInfo.UnionConfig

			// Check for default case
			if typeInfo.UnionConfig.DefaultCase != "" {
				defaultCount++
			}
		}
	}

	// Validate default count (0 or 1)
	if defaultCount > 1 {
		errors = append(errors, ValidationError{
			Location: "union configuration",
			Message:  fmt.Sprintf("found %d default cases, only 0 or 1 allowed", defaultCount),
		})
	}

	// Validate each union configuration
	for containerType, config := range unionConfigs {
		// Find the container struct to get its key field
		var containerStruct TypeInfo
		found := false
		for _, typeInfo := range types {
			if typeInfo.Name == containerType {
				containerStruct = typeInfo
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s", containerType),
				Message:  fmt.Sprintf("container type %s not found", containerType),
			})
			continue
		}

		// Find the key field to get the discriminant type
		var keyField FieldInfo
		for _, field := range containerStruct.Fields {
			if field.IsKey {
				keyField = field
				break
			}
		}
		if keyField.Name == "" {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s", containerType),
				Message:  fmt.Sprintf("container type %s must have a field with xdr:\"key\" tag", containerType),
			})
			continue
		}

		// Validate that the key field type is a uint32 alias
		keyFieldNode, exists := typeDefs[keyField.Type]
		if !exists {
			var keys []string
			for k := range typeDefs {
				keys = append(keys, k)
			}
			fmt.Printf("validateUnionConfiguration: keyField.Type=%q, typeDefs keys: %v\n", keyField.Type, keys)
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("key field type %s must be a Go type, not found", keyField.Type),
			})
			continue
		}

		// Check that the key field type is a uint32 alias
		if ident, ok := keyFieldNode.(*ast.Ident); ok {
			// This is a type alias, check if it's uint32
			if ident.Name != "uint32" {
				errors = append(errors, ValidationError{
					Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
					Message:  fmt.Sprintf("key field type %s must be a uint32 alias, got %s", keyField.Type, ident.Name),
				})
				continue
			}
		} else {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("key field type %s must be a uint32 alias", keyField.Type),
			})
			continue
		}

		// Collect constants of this discriminant type
		typedConstants := collectTypedConstants(file, keyField.Type)
		if len(typedConstants) == 0 {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("no constants found for discriminant type %s", keyField.Type),
			})
			continue
		}

		// Validate that each case maps to a valid constant and struct
		for constantName, structName := range config.Cases {
			// Check if the constant exists for this discriminant type
			if _, exists := typedConstants[constantName]; !exists {
				errors = append(errors, ValidationError{
					Location: fmt.Sprintf("union=%s,case=%s", containerType, constantName),
					Message:  fmt.Sprintf("constant %s not found for discriminant type %s", constantName, keyField.Type),
				})
			}

			// Handle void cases (nil)
			if structName == "nil" {
				continue // Void case - no validation needed
			}

			// Validate struct or alias-to-struct
			resolved, kind := resolveToStruct(structName)
			switch kind {
			case "notype":
				errors = append(errors, ValidationError{
					Location: fmt.Sprintf("union=%s,case=%s", containerType, constantName),
					Message:  fmt.Sprintf("union case payload type %s not found", structName),
				})
			case "struct":
				// Check all fields are tagged
				for _, field := range resolved.Fields.List {
					if field.Tag == nil {
						errors = append(errors, ValidationError{
							Location: fmt.Sprintf("union=%s,case=%s", containerType, constantName),
							Message:  fmt.Sprintf("field %s in struct %s is missing xdr tag", field.Names[0].Name, structName),
						})
					}
				}
			default:
				errors = append(errors, ValidationError{
					Location: fmt.Sprintf("union=%s,case=%s", containerType, constantName),
					Message:  fmt.Sprintf("union case payload type %s must be a struct or alias to struct, got %s", structName, kind),
				})
			}
		}

		// Find void cases (constants without mappings)
		// For now, we'll skip this as it requires more complex logic to determine
		// which constants belong to which discriminant type
	}

	// In validateUnionConfiguration, add validation for default types
	for i := range types {
		if types[i].IsDiscriminatedUnion {
			// Find the union field to validate default type
			var unionField FieldInfo
			for _, field := range types[i].Fields {
				if field.IsUnion {
					unionField = field
					break
				}
			}
			if unionField.DefaultType != "" && unionField.DefaultType != "nil" {
				// Validate that the default type exists
				resolved, kind := resolveToStruct(unionField.DefaultType)
				switch kind {
				case "notype":
					errors = append(errors, ValidationError{
						Location: fmt.Sprintf("union field %s", unionField.Name),
						Message:  fmt.Sprintf("default type %s not found", unionField.DefaultType),
					})
				case "struct":
					// Check all fields are tagged
					for _, field := range resolved.Fields.List {
						if field.Tag == nil {
							errors = append(errors, ValidationError{
								Location: fmt.Sprintf("union field %s default type %s", unionField.Name, unionField.DefaultType),
								Message:  fmt.Sprintf("field %s in default struct %s is missing xdr tag", field.Names[0].Name, unionField.DefaultType),
							})
						}
					}
				default:
					errors = append(errors, ValidationError{
						Location: fmt.Sprintf("union field %s", unionField.Name),
						Message:  fmt.Sprintf("default type %s must be a struct or alias to struct, got %s", unionField.DefaultType, kind),
					})
				}
			}
		}
	}

	return errors
}

// collectConstants collects all constant definitions from the AST
func collectConstants(file *ast.File) map[string]string {
	constants := make(map[string]string)

	ast.Inspect(file, func(n ast.Node) bool {
		if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range valueSpec.Names {
						if i < len(valueSpec.Values) {
							if basicLit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
								constants[name.Name] = basicLit.Value
							}
						}
					}
				}
			}
		}
		return true
	})

	return constants
}

// collectTypedConstants collects constants that are of a specific type
func collectTypedConstants(file *ast.File, typeName string) map[string]string {
	constants := make(map[string]string)

	ast.Inspect(file, func(n ast.Node) bool {
		if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					// Check if this constant group has the target type
					if valueSpec.Type != nil {
						if ident, ok := valueSpec.Type.(*ast.Ident); ok && ident.Name == typeName {
							for i, name := range valueSpec.Names {
								if i < len(valueSpec.Values) {
									if basicLit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
										constants[name.Name] = basicLit.Value
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return constants
}

// hasExistingMethods checks if a type already has Encode/Decode methods
func hasExistingMethods(file *ast.File, typeName string) (hasEncode, hasDecode bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				receiverType := extractReceiverType(fn.Recv.List[0].Type)
				if receiverType == typeName {
					switch fn.Name.Name {
					case "Encode":
						hasEncode = true
					case "Decode":
						hasDecode = true
					}
				}
			}
		}
		return true
	})
	return
}

// extractReceiverType extracts the type name from a receiver type expression
func extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return extractReceiverType(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}

// extractBuildTags extracts build tags from a Go source file
func extractBuildTags(filename string) []string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	var buildTags []string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for //go:build directive
		if strings.HasPrefix(line, "//go:build ") {
			buildTags = append(buildTags, line)
			continue
		}

		// Check for // +build directive
		if strings.HasPrefix(line, "// +build ") {
			buildTags = append(buildTags, line)
			continue
		}

		// If we hit a non-comment/non-blank line, stop looking
		if !strings.HasPrefix(line, "//") {
			break
		}
	}

	return buildTags
}

// parseFile parses a Go file and extracts structs that have go:generate xdrgen directives
func parseFile(filename string) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
	fset := token.NewFileSet()

	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, nil, err
	}
	lines := strings.Split(string(fileBytes), "\n")
	var misplacedUnionComments []ValidationError

	// Map of all type definitions (structs and aliases)
	typeDefs := make(map[string]ast.Node)

	// Scan for misplaced //xdr:union=... comments
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//xdr:union=") {
			// Look ahead for next non-comment, non-blank line
			seen := false
			for j := i + 1; j < len(lines); j++ {
				next := strings.TrimSpace(lines[j])
				if next == "" || strings.HasPrefix(next, "//") {
					continue
				}
				seen = true
				if !strings.HasPrefix(next, "type ") || !strings.Contains(next, " struct {") {
					misplacedUnionComments = append(misplacedUnionComments, ValidationError{
						Location: fmt.Sprintf("line %d", i+1),
						Message:  fmt.Sprintf("union comment not followed by struct: %s", trimmed),
					})
				}
				break
			}
			if !seen {
				misplacedUnionComments = append(misplacedUnionComments, ValidationError{
					Location: fmt.Sprintf("line %d", i+1),
					Message:  fmt.Sprintf("union comment not followed by struct: %s", trimmed),
				})
			}
		}
		if strings.HasPrefix(trimmed, "// xdr:") && debug {
			debugf("Warning: ignoring malformed union comment (has space): %s", trimmed)
		}
	}

	// For files with build tags, we need to parse them with the appropriate build context
	// For now, we'll parse all files regardless of build tags to ensure we don't miss any
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build typeDefs map
	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.TypeSpec); ok {
			typeDefs[node.Name.Name] = node.Type
		}
		return true
	})

	var types []TypeInfo

	// Build a map of type aliases for lookup
	typeAliases := make(map[string]string)

	// First pass: collect all type definitions (type Alias Underlying)
	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.TypeSpec); ok {
			if node.Assign == 0 { // This is a type definition (type Alias Underlying)
				underlyingType := formatType(node.Type)
				typeAliases[node.Name.Name] = underlyingType
			}
		}
		return true
	})

	// Build a map of struct names that have go:generate directives
	structsToGenerate := make(map[string]bool)

	// Look for go:generate comments that reference xdrgen
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(text, "go:generate") && strings.Contains(text, "xdrgen") {
				// Extract struct name from go:generate comment
				// Flexible format: "go:generate [path/]xdrgen StructName" or "go:generate [path/]xdrgen $GOFILE"
				parts := strings.Fields(text)
				if len(parts) >= 2 {
					structName := parts[len(parts)-1] // Last argument should be struct name
					// Handle $GOFILE case - means generate for all structs in this file
					if structName == "$GOFILE" {
						// For $GOFILE, we'll mark all structs with XDR tags for generation
						// This will be handled in the main inspection loop
						continue
					}
					structsToGenerate[structName] = true
				}
			}
		}
	}

	// Check if we have a $GOFILE directive (generate all structs with XDR tags)
	hasGOFILEDirective := false
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(text, "go:generate") && strings.Contains(text, "xdrgen") {
				// Check for $GOFILE or just xdrgen (both mean process all structs with XDR tags)
				if strings.Contains(text, "$GOFILE") || strings.TrimSpace(strings.TrimPrefix(text, "go:generate")) == "xdrgen" {
					hasGOFILEDirective = true
					break
				}
			}
		}
		if hasGOFILEDirective {
			break
		}
	}

	// If no go:generate directive found, process all structs with XDR tags
	// This allows processing test files and other files without explicit directives
	if !hasGOFILEDirective && len(structsToGenerate) == 0 {
		hasGOFILEDirective = true
	}

	// Collect all union comments from the file
	unionComments := make(map[string]*UnionConfig)
	unionCommentPositions := make(map[token.Pos]*UnionConfig) // Track comment positions

	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			unionConfig, err := parseUnionComment(comment.Text)
			if err != nil {
				log.Fatalf("Error parsing union comment: %v", err)
			}
			if unionConfig != nil {
				debugf("Found union comment: %+v", unionConfig)
				// Store by container type, but we'll fill in the struct names later
				unionComments[unionConfig.ContainerType] = unionConfig
				// Track the position of this comment for better association
				unionCommentPositions[comment.Pos()] = unionConfig
			}
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := node.Type.(*ast.StructType); ok {
				typeInfo := TypeInfo{
					Name: node.Name.Name,
				}

				debugf("Found struct: %s", typeInfo.Name)

				// Check if this struct should be generated
				shouldGenerate := false
				if hasGOFILEDirective {
					// For $GOFILE directive, check if struct has XDR tags
					for _, field := range structType.Fields.List {
						if field.Tag != nil {
							tag := strings.Trim(field.Tag.Value, "`")
							if strings.Contains(tag, "xdr:") {
								shouldGenerate = true
								debugf("Struct %s has XDR tag: %s", typeInfo.Name, tag)
								break
							}
						}
					}
				} else {
					// Check explicit struct name in go:generate directives
					shouldGenerate = structsToGenerate[typeInfo.Name]
					debugf("Struct %s explicit generation: %v", typeInfo.Name, shouldGenerate)
				}

				if !shouldGenerate {
					debugf("Skipping struct %s - no XDR tags or not in generate list", typeInfo.Name)
					return true
				}

				// Check if this type already has Encode/Decode methods
				hasEncode, hasDecode := hasExistingMethods(file, typeInfo.Name)
				if hasEncode && hasDecode {
					logf("Skipping %s - already has Encode/Decode methods", typeInfo.Name)
					return true
				}
				if hasEncode || hasDecode {
					log.Fatalf("Type %s has partial XDR implementation (only %s method found). Either implement both Encode and Decode manually, or remove existing method to use code generation.",
						typeInfo.Name,
						map[bool]string{true: "Encode", false: "Decode"}[hasEncode])
				}

				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						fieldInfo := FieldInfo{
							Name: name.Name,
							Type: formatType(field.Type),
						}

						// Require explicit XDR tags for all fields when struct is opted in
						if field.Tag == nil {
							log.Fatalf("Field %s.%s missing XDR tag. All fields in XDR-enabled structs must have explicit tags like `xdr:\"uint32\"` or `xdr:\"-\"` to exclude",
								typeInfo.Name, name.Name)
						}

						tag := strings.Trim(field.Tag.Value, "`")
						fieldInfo.Tag = tag
						xdrTag := parseXDRTag(tag)

						// Require XDR tag when struct is opted in
						if xdrTag == "" {
							log.Fatalf("Field %s.%s missing XDR tag. Use `xdr:\"type\"` or `xdr:\"-\"` to exclude",
								typeInfo.Name, name.Name)
						}

						// Check for explicit exclusion (like json:"-")
						if xdrTag == "-" {
							continue // Skip this field
						}

						// Only allow xdr:"alias" now
						if xdrTag == "alias" {
							// Look up underlying type from type definitions
							underlyingType, exists := typeAliases[fieldInfo.Type]
							if !exists {
								// Fall back to direct type inference for built-in types
								var err error
								underlyingType, err = inferUnderlyingType(fieldInfo.Type)
								if err != nil {
									log.Fatalf("Cannot infer underlying type for alias field %s.%s of type %s: %v. Use xdr:\"alias\" only on supported types.",
										typeInfo.Name, fieldInfo.Name, fieldInfo.Type, err)
								}
							}
							logf("Alias field %s.%s: %s -> %s", typeInfo.Name, fieldInfo.Name, fieldInfo.Type, underlyingType)
							fieldInfo.XDRType = "alias:" + underlyingType
						} else {
							fieldInfo.XDRType = xdrTag
						}

						// Check for discriminated union tags
						isKey, isUnion, defaultType := parseKeyUnionTag(fieldInfo.XDRType)
						fieldInfo.IsKey = isKey
						fieldInfo.IsUnion = isUnion
						fieldInfo.DefaultType = defaultType // Store the default type from the tag

						// Mark struct as discriminated union if it contains key field
						if isKey {
							typeInfo.IsDiscriminatedUnion = true
							// Key fields must be uint32 type or an alias of uint32
							underlyingType, ok := typeAliases[fieldInfo.Type]
							if fieldInfo.Type != "uint32" && (!ok || underlyingType != "uint32") {
								log.Fatalf("Key field %s.%s must be uint32 or an alias of uint32, got %s", typeInfo.Name, fieldInfo.Name, fieldInfo.Type)
							}
							// Key fields should be treated as uint32 for encoding/decoding
							fieldInfo.XDRType = "uint32"
						}
						if isUnion {
							// Union fields must be []byte
							if fieldInfo.Type != "[]byte" {
								log.Fatalf("Union field %s.%s must be of type []byte (got %s)", typeInfo.Name, fieldInfo.Name, fieldInfo.Type)
							}
							// Union fields should be treated as bytes for encoding/decoding
							fieldInfo.XDRType = "bytes"
						}

						// Validate that we can handle this XDR type (after key/union processing)
						if !isSupportedXDRType(fieldInfo.XDRType) {
							log.Fatalf("Unsupported XDR type '%s' for field %s.%s. Supported types: uint32, uint64, string, bytes, bool, struct, array, fixed:N, key, union. Add support or implement custom Encode/Decode methods.",
								fieldInfo.XDRType, typeInfo.Name, fieldInfo.Name)
						}

						typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
					}
				}

				// Associate union comments with this struct
				if typeInfo.IsDiscriminatedUnion {
					// Container struct - look for union config by container type name
					if unionConfig, exists := unionComments[typeInfo.Name]; exists {
						typeInfo.UnionConfig = unionConfig
						if debug {
							debugf("Associated union config with container struct %s", typeInfo.Name)
						}
					}
				} else {
					// Check if this struct has a union comment (payload struct)
					// For cross-file processing, we need to associate union comments with payload structs
					// Since we don't have position information, we'll use a simpler approach
					for containerType, unionConfig := range unionComments {
						// Fill in the struct name for any empty case mappings
						for constantName, structName := range unionConfig.Cases {
							if structName == "" {
								unionConfig.Cases[constantName] = typeInfo.Name
								debugf("Filled in cross-file struct name %s for case %s in container %s", typeInfo.Name, constantName, containerType)
								break
							}
						}
					}
				}

				if len(typeInfo.Fields) > 0 {
					types = append(types, typeInfo)
				}
			}
		}
		return true
	})

	// Compute void cases for each container struct
	for _, typeInfo := range types {
		if typeInfo.IsDiscriminatedUnion && typeInfo.UnionConfig != nil {
			// Find the key field to get the discriminant type
			var keyField FieldInfo
			for _, field := range typeInfo.Fields {
				if field.IsKey {
					keyField = field
					break
				}
			}
			if keyField.Name == "" {
				log.Fatalf("Container struct %s must have a field with xdr:\"key\" tag", typeInfo.Name)
			}

			// Collect all constants of this discriminant type
			typedConstants := collectTypedConstants(file, keyField.Type)

			// Compute void cases: all constants not mapped to payload structs
			for constantName := range typedConstants {
				if _, exists := typeInfo.UnionConfig.Cases[constantName]; !exists {
					typeInfo.UnionConfig.VoidCases = append(typeInfo.UnionConfig.VoidCases, constantName)
					if debug {
						debugf("Added void case %s for container %s", constantName, typeInfo.Name)
					}
				}
			}
		}
	}

	return types, misplacedUnionComments, typeDefs, nil
}

// parsePackage parses all files in a package to build complete union configurations
func parsePackage(packageDir string) (map[string][]TypeInfo, map[string]map[string]ast.Node, error) {
	// Discover all Go files in the package
	files, err := discoverGoFiles(packageDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error discovering Go files: %w", err)
	}

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no files with //go:generate xdrgen found in %s", packageDir)
	}

	// Parse each file
	fileTypes := make(map[string][]TypeInfo)
	fileTypeDefs := make(map[string]map[string]ast.Node)

	for _, file := range files {
		types, _, typeDefs, err := parseFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing %s: %w", file, err)
		}

		fileTypes[file] = types
		fileTypeDefs[file] = typeDefs
	}

	// Build complete union configurations across all files
	// TODO: Implement cross-file union configuration gathering

	return fileTypes, fileTypeDefs, nil
}

// formatType converts an ast.Expr to a string representation
func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return formatType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + formatType(t.Elt)
		}
		return "[" + formatType(t.Len) + "]" + formatType(t.Elt)
	case *ast.InterfaceType:
		return "any"
	case *ast.BasicLit:
		return t.Value
	default:
		return fmt.Sprintf("%T", t)
	}
}

// inferUnderlyingType infers the XDR underlying type from a Go type alias
func inferUnderlyingType(goType string) (string, error) {
	switch goType {
	case "string":
		return "string", nil
	case "[]byte":
		return "bytes", nil
	case "uint32":
		return "uint32", nil
	case "uint64":
		return "uint64", nil
	case "int32":
		return "int32", nil
	case "int64":
		return "int64", nil
	case "bool":
		return "bool", nil
	default:
		// Check for fixed-size byte arrays like [16]byte
		if strings.HasPrefix(goType, "[") && strings.HasSuffix(goType, "]byte") {
			return "fixed:" + strings.TrimSuffix(strings.TrimPrefix(goType, "["), "]byte"), nil
		}
		return "", fmt.Errorf("unsupported type for alias inference: %s (supported: string, []byte, [N]byte, uint32, uint64, int32, int64, bool)", goType)
	}
}

// Helper: detect loops in union payload chains
func detectUnionLoop(start string, unionConfigs map[string]*UnionConfig, visited map[string]bool) bool {
	if visited[start] {
		return true
	}
	visited[start] = true
	for _, payload := range unionConfigs[start].Cases {
		if unionConfigs[payload] == nil {
			continue // not a union container, skip
		}
		if detectUnionLoop(payload, unionConfigs, visited) {
			return true
		}
	}
	visited[start] = false
	return false
}

// discoverGoFiles finds all .go files in a directory that have //go:generate xdrgen directives
func discoverGoFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())

		// Check if file has //go:generate xdrgen directive
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		hasGenerate := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(strings.TrimPrefix(line, "//"))
			if strings.HasPrefix(trimmed, "go:generate") && strings.Contains(trimmed, "xdrgen") {
				hasGenerate = true
				break
			}
		}

		if hasGenerate {
			files = append(files, filePath)
		}
	}

	return files, nil
}

// isDirectory checks if the given path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func main() {
	// Configure log to remove timestamps for cleaner output
	log.SetFlags(0)

	flag.BoolVar(&silent, "silent", false, "suppress all output except errors")
	flag.BoolVar(&silent, "s", false, "suppress all output except errors (shorthand)")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "xdrgen - XDR Code Generator\n\n")
		fmt.Fprintf(os.Stderr, "Generates Encode and Decode methods for Go structs with XDR struct tags.\n")
		fmt.Fprintf(os.Stderr, "Supports the xdr.Codec interface for automatic XDR serialization/deserialization.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  xdrgen [flags] <package-dir>     # Process package (files with //go:generate)\n")
		fmt.Fprintf(os.Stderr, "  xdrgen [flags] <file.go>         # Process single file\n\n")
		fmt.Fprintf(os.Stderr, "The generator processes files that are explicitly opted in via go:generate directives:\n")
		fmt.Fprintf(os.Stderr, "  //go:generate xdrgen $GOPACKAGE              (process all XDR-tagged structs in package)\n")
		fmt.Fprintf(os.Stderr, "  //go:generate xdrgen StructName              (process specific struct in package)\n")
		fmt.Fprintf(os.Stderr, "  //go:generate ../../tools/bin/xdrgen StructName  (with path)\n\n")
		fmt.Fprintf(os.Stderr, "Supported XDR struct tags:\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"uint32\"`     - 32-bit unsigned integer\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"uint64\"`     - 64-bit unsigned integer\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"int32\"`      - 32-bit signed integer\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"int64\"`      - 64-bit signed integer\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"string\"`     - variable-length string\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"bytes\"`      - variable-length byte array\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"bool\"`       - boolean (encoded as uint32)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"struct\"`     - nested struct (must implement xdr.Codec)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"array\"`      - variable-length array\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"fixed:N\"`    - fixed-size byte array (N bytes)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"alias\"`       - type alias with automatic type inference\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"key\"`         - discriminated union discriminant field\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"union\"`       - discriminated union payload field\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"union,default=nil\"` - discriminated union payload with void default\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"-\"`          - exclude field from encoding/decoding\n\n")
		fmt.Fprintf(os.Stderr, "Discriminated Unions:\n")
		fmt.Fprintf(os.Stderr, "  Use //xdr:union=<container>,case=<constant> comments on payload structs\n")
		fmt.Fprintf(os.Stderr, "  Void cases are automatically inferred from unmapped constants\n")
		fmt.Fprintf(os.Stderr, "  Example: //xdr:union=NetworkMessage,case=MessageTypeText\n\n")
		fmt.Fprintf(os.Stderr, "Type Aliases:\n")
		fmt.Fprintf(os.Stderr, "  Supported Go types: string, []byte, [N]byte, uint32, uint64, int32, int64, bool\n")
		fmt.Fprintf(os.Stderr, "  Example: type UserID string `xdr:\"alias\"`\n\n")
		fmt.Fprintf(os.Stderr, "Output:\n")
		fmt.Fprintf(os.Stderr, "  Creates <input>_xdr.go with generated Encode/Decode methods\n")
		fmt.Fprintf(os.Stderr, "  Includes compile-time assertions that types implement xdr.Codec\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  xdrgen ./types/         # Generate for files with //go:generate in package\n")
		fmt.Fprintf(os.Stderr, "  xdrgen types.go         # Generate for single file (any file)\n")
		fmt.Fprintf(os.Stderr, "  xdrgen -s ./types/      # Generate silently (no output except errors)\n")
		fmt.Fprintf(os.Stderr, "  go generate            # Use with go:generate directives\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := flag.Arg(0)

	// Convert to absolute path
	inputPath, err := filepath.Abs(inputPath)
	if err != nil {
		log.Fatal("Error getting absolute path:", err)
	}

	var filesToProcess []string

	if isDirectory(inputPath) {
		// Package mode: discover files with //go:generate xdrgen
		files, err := discoverGoFiles(inputPath)
		if err != nil {
			log.Fatal("Error discovering Go files:", err)
		}
		logf("processFileWithPackageContext: files to process: %v", files)

		if len(files) == 0 {
			logf("No files with //go:generate xdrgen found in %s", inputPath)
			return
		}

		filesToProcess = files
		if !silent {
			logf("Found %d files with //go:generate xdrgen in %s", len(files), inputPath)
		}
	} else {
		// Single file mode: process the specified file
		filesToProcess = []string{inputPath}
	}

	// Process each file
	for _, inputFile := range filesToProcess {
		processFile(inputFile)
	}
}

// processFile handles the processing of a single file
func processFile(inputFile string) {
	// Check if we're in package mode (inputFile is in a directory with other files)
	dir := filepath.Dir(inputFile)
	if isDirectory(dir) {
		// Package mode: gather union configurations across all files
		processFileWithPackageContext(inputFile, dir)
	} else {
		// Single file mode: process independently
		processFileIndependent(inputFile)
	}
}

// processFileWithPackageContext processes a file with package-level union configuration gathering
func processFileWithPackageContext(inputFile, packageDir string) {
	// Discover all Go files in the package
	files, err := discoverGoFiles(packageDir)
	if err != nil {
		log.Fatal("Error discovering Go files:", err)
	}

	// Parse all files to gather union configurations and type information
	allUnionComments := make(map[string]*UnionConfig)
	allTypeDefs := make(map[string]ast.Node)
	allConstants := make(map[string]string)
	allStructTypes := make(map[string]bool)           // Track all struct types across package
	allContainerStructs := make(map[string]*TypeInfo) // Track discriminated union containers across package

	for _, file := range files {
		// Parse each file to get union comments and type definitions
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			log.Fatal("Error parsing file:", err)
		}

		// Collect union comments from this file and associate them with following struct definitions
		ast.Inspect(astFile, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GenDecl:
				if node.Tok == token.TYPE {
					// Look for union comments preceding type declarations
					for _, spec := range node.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if _, ok := typeSpec.Type.(*ast.StructType); ok {
								// Check if there's a union comment before this struct
								if node.Doc != nil {
									for _, comment := range node.Doc.List {
										unionConfig, err := parseUnionComment(comment.Text)
										if err != nil {
											log.Fatalf("Error parsing union comment: %v", err)
										}
										if unionConfig != nil {
											debugf("Found cross-file union comment: %+v for struct %s", unionConfig, typeSpec.Name.Name)

											// Fill in the struct name for the case
											for constantName := range unionConfig.Cases {
												debugf("Filling case %s with struct name %s", constantName, typeSpec.Name.Name)
												unionConfig.Cases[constantName] = typeSpec.Name.Name
											}

											// Merge with existing config for this container type
											if existingConfig, exists := allUnionComments[unionConfig.ContainerType]; exists {
												// Merge cases
												for constantName, structName := range unionConfig.Cases {
													existingConfig.Cases[constantName] = structName
												}
												// Merge void cases
												existingConfig.VoidCases = append(existingConfig.VoidCases, unionConfig.VoidCases...)
												// Handle default case
												if unionConfig.DefaultCase != "" {
													existingConfig.DefaultCase = unionConfig.DefaultCase
												}
											} else {
												// Store new config
												allUnionComments[unionConfig.ContainerType] = unionConfig
											}
											debugf("After processing, config for %s: %+v", unionConfig.ContainerType, allUnionComments[unionConfig.ContainerType])
										}
									}
								}
							}
						}
					}
				}
			}
			return true
		})

		// Build typeDefs map for this file
		ast.Inspect(astFile, func(n ast.Node) bool {
			if node, ok := n.(*ast.TypeSpec); ok {
				allTypeDefs[node.Name.Name] = node.Type
				// Track struct types
				if _, ok := node.Type.(*ast.StructType); ok {
					allStructTypes[node.Name.Name] = true
				}
			}
			return true
		})

		// Print typeDefs keys for this file
		var fileKeys []string
		for k := range allTypeDefs {
			fileKeys = append(fileKeys, k)
		}
		debugf("processFileWithPackageContext: after %s, allTypeDefs keys: %v", file, fileKeys)

		// Collect constants from this file
		fileConstants := collectConstants(astFile)
		for name, value := range fileConstants {
			allConstants[name] = value
		}

		// Collect container structs from AST inspection (without full parsing/validation)
		ast.Inspect(astFile, func(n ast.Node) bool {
			if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							// Check if this struct has key/union fields (discriminated union)
							hasKey := false
							hasUnion := false
							for _, field := range structType.Fields.List {
								if field.Tag != nil {
									tag := strings.Trim(field.Tag.Value, "`")
									xdrTag := parseXDRTag(tag)
									isKey, isUnion, _ := parseKeyUnionTag(xdrTag)
									if isKey {
										hasKey = true
									}
									if isUnion {
										hasUnion = true
									}
								}
							}
							if hasKey && hasUnion {
								// Create a TypeInfo with field information for the container
								var fields []FieldInfo
								for _, field := range structType.Fields.List {
									if field.Tag != nil {
										tag := strings.Trim(field.Tag.Value, "`")
										xdrTag := parseXDRTag(tag)
										isKey, isUnion, defaultType := parseKeyUnionTag(xdrTag)

										if field.Names != nil && len(field.Names) > 0 {
											fieldInfo := FieldInfo{
												Name:        field.Names[0].Name,
												IsKey:       isKey,
												IsUnion:     isUnion,
												DefaultType: defaultType,
											}
											fields = append(fields, fieldInfo)
										}
									}
								}

								containerInfo := &TypeInfo{
									Name:                 typeSpec.Name.Name,
									IsDiscriminatedUnion: true,
									Fields:               fields,
									UnionConfig:          nil, // Will be set during association
								}
								allContainerStructs[typeSpec.Name.Name] = containerInfo
								debugf("Found container struct %s in file %s", typeSpec.Name.Name, file)
							}
						}
					}
				}
			}
			return true
		})
	}

	// Package-level union configuration association
	// Associate union configurations with container structs across all files
	debugf("Available union configs: %+v", allUnionComments)
	debugf("Available container structs: %v", func() []string {
		var names []string
		for name := range allContainerStructs {
			names = append(names, name)
		}
		return names
	}())

	for containerName, unionConfig := range allUnionComments {
		if containerStruct, exists := allContainerStructs[containerName]; exists {
			containerStruct.UnionConfig = unionConfig
			debugf("Associated union config with container struct %s at package level", containerName)
		} else {
			log.Fatalf("Cross-package union association error: Container struct %s referenced in union comments but not found in package", containerName)
		}
	}

	if debug {
		var keys []string
		for k := range allTypeDefs {
			keys = append(keys, k)
		}
		fmt.Printf("processFileWithPackageContext: allTypeDefs keys: %v\n", keys)
	}

	// Validate the complete package state before generating any code
	if err := validateCompletePackageState(allContainerStructs, allUnionComments, allConstants, allTypeDefs); err != nil {
		log.Fatal("Package validation error:", err)
	}

	// Now process the specific file with complete package context
	processFileWithPackageUnionContext(inputFile, allUnionComments, allTypeDefs, allConstants, allStructTypes, allContainerStructs)
}

// validateCompletePackageState validates the complete package state after all aggregation and mapping
func validateCompletePackageState(allContainerStructs map[string]*TypeInfo, allUnionComments map[string]*UnionConfig, allConstants map[string]string, allTypeDefs map[string]ast.Node) error {
	var errors []string

	// Validate that all union comments reference existing containers
	for containerType := range allUnionComments {
		if _, exists := allContainerStructs[containerType]; !exists {
			errors = append(errors, fmt.Sprintf("Union comment references unknown container struct %s", containerType))
		}
	}

	// Validate that all containers have union configs
	for containerName, containerStruct := range allContainerStructs {
		if containerStruct.UnionConfig == nil {
			// Check if the container has default=nil on its union field
			hasDefaultNil := false
			if len(containerStruct.Fields) > 0 {
				for _, field := range containerStruct.Fields {
					if field.IsUnion && field.DefaultType == "nil" {
						hasDefaultNil = true
						break
					}
				}
			}

			if !hasDefaultNil {
				errors = append(errors, fmt.Sprintf("Container struct %s has key/union fields but no payload structs found. Add union comments (//xdr:union=%s,case=<constant>) to payload structs or use xdr:\"union,default=nil\" for void-only unions", containerName, containerName))
			}
		}
	}

	// Validate that all constants referenced in union cases exist
	for containerType, unionConfig := range allUnionComments {
		for constantName := range unionConfig.Cases {
			if _, exists := allConstants[constantName]; !exists {
				errors = append(errors, fmt.Sprintf("Union case %s in container %s references unknown constant %s", constantName, containerType, constantName))
			}
		}
	}

	// Report all errors at once
	if len(errors) > 0 {
		return fmt.Errorf("Package validation errors:\n%s", strings.Join(errors, "\n"))
	}

	debugf("Package validation passed: %d containers, %d union configs", len(allContainerStructs), len(allUnionComments))
	return nil
}

// processFileWithPackageUnionContext processes a file with complete package-level union configuration context
func processFileWithPackageUnionContext(inputFile string, allUnionComments map[string]*UnionConfig, allTypeDefs map[string]ast.Node, allConstants map[string]string, allStructTypes map[string]bool, allContainerStructs map[string]*TypeInfo) {
	// Generate output file name, handling test files specially
	var outputFile string
	if strings.HasSuffix(inputFile, "_test.go") {
		outputFile = strings.TrimSuffix(inputFile, "_test.go") + "_xdr_test.go"
	} else {
		outputFile = strings.TrimSuffix(inputFile, ".go") + "_xdr.go"
	}

	debugf("Processing input file: %s", inputFile)
	types, _, _, err := parseFile(inputFile)
	if err != nil {
		log.Fatal("Error parsing file:", err)
	}

	debugf("Found %d types in input file", len(types))
	for i := range types {
		debugf("Type %s: IsDiscriminatedUnion=%v", types[i].Name, types[i].IsDiscriminatedUnion)
	}

	// Update container structs with package-level union configurations
	for i := range types {
		if types[i].IsDiscriminatedUnion {
			if packageContainer, exists := allContainerStructs[types[i].Name]; exists {
				types[i].UnionConfig = packageContainer.UnionConfig
				debugf("Updated container struct %s with package-level union config", types[i].Name)
			} else {
				debugf("No package container found for %s", types[i].Name)
			}
		}
	}

	// Package-level validation is done before individual file processing

	if len(types) == 0 {
		logf("No types with XDR tags found in %s", inputFile)
		return
	}

	// Show counts in standard mode
	if !silent {
		logf("Found %d types with XDR tags", len(types))
	}

	// Extract package name from input file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		log.Fatal("Error parsing package:", err)
	}

	packageName := file.Name.Name // Use the package name from the source file exactly, do not modify

	// Use package-level constants for union validation
	constants := allConstants

	// Package-level union configuration aggregation already done
	// Union configs are now associated with container structs at package level

	// Check if this file contains any structs that need code generation
	// Only generate code for files that actually contain XDR-tagged structs
	if len(types) == 0 {
		debugf("No XDR-tagged structs found in %s, skipping code generation", inputFile)
		return
	}

	// Validate discriminated unions with package-level constants
	if err := validateDiscriminatedUnions(types, constants); err != nil {
		log.Fatal("Validation error:", err)
	}

	// Validate union configurations with package-level context
	validationErrors := validateUnionConfiguration(types, constants, allTypeDefs, file)
	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			logf("Validation error: %s: %s", err.Location, err.Message)
		}
		log.Fatal("Union configuration validation failed")
	}

	// Collect external imports
	var externalImports []string
	for _, typeInfo := range types {
		imports := collectExternalImports([]TypeInfo{typeInfo})
		externalImports = append(externalImports, imports...)
	}

	// Remove duplicates
	externalImports = removeDuplicates(externalImports)

	// Extract build tags from input file
	buildTags := extractBuildTags(inputFile)

	// Extract struct type names for dynamic type detection
	// Include ALL struct types from the file, not just those being processed
	var structTypeNames []string
	for _, typeInfo := range types {
		structTypeNames = append(structTypeNames, typeInfo.Name)
	}

	// Also collect all struct types from allTypeDefs (including those not being processed)
	debugf("TypeDefs map contains %d types", len(allTypeDefs))
	for typeName, typeNode := range allTypeDefs {
		if _, ok := typeNode.(*ast.StructType); ok {
			debugf("Found struct type in typeDefs: %s", typeName)
			// Check if this struct type is not already in the list
			found := false
			for _, existing := range structTypeNames {
				if existing == typeName {
					found = true
					break
				}
			}
			if !found {
				structTypeNames = append(structTypeNames, typeName)
				debugf("Added struct type from typeDefs: %s", typeName)
			}
		}
	}

	// Also collect imported struct types from field types
	for _, typeInfo := range types {
		for _, field := range typeInfo.Fields {
			debugf("Processing field %s.%s: Type=%s, XDRType=%s", typeInfo.Name, field.Name, field.Type, field.XDRType)

			// Check if this field type is an imported struct
			if strings.Contains(field.Type, ".") {
				// Extract the type name from package.Type format
				parts := strings.Split(field.Type, ".")
				if len(parts) == 2 {
					importedTypeName := parts[1]
					debugf("Found imported type: %s (from %s)", importedTypeName, field.Type)
					// Check if this imported type is not already in the list
					found := false
					for _, existing := range structTypeNames {
						if existing == importedTypeName {
							found = true
							break
						}
					}
					if !found {
						// For imported types, we assume they are structs if they're used with xdr:"struct"
						// or if they appear in array contexts
						if field.XDRType == "struct" || field.XDRType == "array" {
							structTypeNames = append(structTypeNames, importedTypeName)
							debugf("Added imported struct type: %s", importedTypeName)
						}
					}
				}
			}

			// Also check for imported types in array element types
			if field.XDRType == "array" && strings.Contains(field.Type, "[]") {
				elementType := strings.TrimPrefix(field.Type, "[]")
				debugf("Array field %s.%s has element type: %s", typeInfo.Name, field.Name, elementType)
				if strings.Contains(elementType, ".") {
					// Extract the type name from package.Type format
					parts := strings.Split(elementType, ".")
					if len(parts) == 2 {
						importedElementTypeName := parts[1]
						debugf("Found imported array element type: %s (from %s)", importedElementTypeName, elementType)
						// Check if this imported element type is not already in the list
						found := false
						for _, existing := range structTypeNames {
							if existing == importedElementTypeName {
								found = true
								break
							}
						}
						if !found {
							// For imported types used as array elements, we assume they are structs
							structTypeNames = append(structTypeNames, importedElementTypeName)
							debugf("Added imported array element struct type: %s", importedElementTypeName)
						}
					}
				} else {
					// This is a type from the same package (no dot), check if it's not in allTypeDefs
					if _, exists := allTypeDefs[elementType]; !exists {
						// Type not found in current file, assume it's a struct from another file in the package
						found := false
						for _, existing := range structTypeNames {
							if existing == elementType {
								found = true
								break
							}
						}
						if !found {
							structTypeNames = append(structTypeNames, elementType)
							debugf("Added cross-file struct type from same package: %s", elementType)
						}
					}
				}
			}
		}
	}

	// Debug output for package-level processing
	debugf("Package-level processing: parsed %d types", len(types))
	for _, typeInfo := range types {
		debugf("Type %s: IsDiscriminatedUnion=%v, UnionConfig=%+v",
			typeInfo.Name, typeInfo.IsDiscriminatedUnion, typeInfo.UnionConfig)
	}

	debugf("Final structTypeNames list (%d types): %v", len(structTypeNames), structTypeNames)

	// Create code generator
	codeGen, err := NewCodeGenerator(structTypeNames)
	if err != nil {
		log.Fatal("Error creating code generator:", err)
	}

	// Generate output
	var output strings.Builder

	// Generate file header
	header, err := codeGen.GenerateFileHeader(inputFile, packageName, externalImports, buildTags)
	if err != nil {
		log.Fatal("Error generating file header:", err)
	}
	output.WriteString(header)
	output.WriteString("\n")

	// Generate code for each type
	for _, typeInfo := range types {
		// Generate encode method
		encodeMethod, err := codeGen.GenerateEncodeMethod(typeInfo)
		if err != nil {
			log.Fatal("Error generating encode method:", err)
		}
		output.WriteString(encodeMethod)
		output.WriteString("\n")

		// Generate decode method
		decodeMethod, err := codeGen.GenerateDecodeMethod(typeInfo)
		if err != nil {
			log.Fatal("Error generating decode method:", err)
		}
		output.WriteString(decodeMethod)
		output.WriteString("\n")

		// Generate compile-time assertion
		assertion, err := codeGen.GenerateAssertion(typeInfo.Name)
		if err != nil {
			log.Fatal("Error generating assertion:", err)
		}
		output.WriteString(assertion)
		output.WriteString("\n")
	}

	// Write to output file
	if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
		log.Fatal("Error writing output file:", err)
	}

	if !silent {
		logf("Generated XDR methods for %d types in %s", len(types), outputFile)
	}
}

// processFileIndependent processes a single file without package-level context
func processFileIndependent(inputFile string) {
	// Generate output file name, handling test files specially
	var outputFile string
	if strings.HasSuffix(inputFile, "_test.go") {
		outputFile = strings.TrimSuffix(inputFile, "_test.go") + "_xdr_test.go"
	} else {
		outputFile = strings.TrimSuffix(inputFile, ".go") + "_xdr.go"
	}

	types, _, typeDefs, err := parseFile(inputFile)
	if err != nil {
		log.Fatal("Error parsing file:", err)
	}

	if len(types) == 0 {
		logf("No types with XDR tags found in %s", inputFile)
		return
	}

	// Show counts in standard mode
	if !silent {
		logf("Found %d types with XDR tags", len(types))
	}

	// Extract package name from input file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		log.Fatal("Error parsing package:", err)
	}

	packageName := file.Name.Name // Use the package name from the source file exactly, do not modify

	// Collect constants for union validation
	constants := collectConstants(file)

	// Validate discriminated unions
	if err := validateDiscriminatedUnions(types, constants); err != nil {
		log.Fatal("Validation error:", err)
	}

	// Validate union configurations
	validationErrors := validateUnionConfiguration(types, constants, typeDefs, file)
	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			logf("Validation error: %s: %s", err.Location, err.Message)
		}
		log.Fatal("Union configuration validation failed")
	}

	// Collect external imports
	var externalImports []string
	for _, typeInfo := range types {
		imports := collectExternalImports([]TypeInfo{typeInfo})
		externalImports = append(externalImports, imports...)
	}

	// Remove duplicates
	externalImports = removeDuplicates(externalImports)

	// Extract build tags from input file
	buildTags := extractBuildTags(inputFile)

	// Extract struct type names for dynamic type detection
	// Include ALL struct types from the file, not just those being processed
	var structTypeNames []string
	for _, typeInfo := range types {
		structTypeNames = append(structTypeNames, typeInfo.Name)
	}

	// Also collect all struct types from typeDefs (including those not being processed)
	debugf("TypeDefs map contains %d types", len(typeDefs))
	for typeName, typeNode := range typeDefs {
		if _, ok := typeNode.(*ast.StructType); ok {
			debugf("Found struct type in typeDefs: %s", typeName)
			// Check if this struct type is not already in the list
			found := false
			for _, existing := range structTypeNames {
				if existing == typeName {
					found = true
					break
				}
			}
			if !found {
				structTypeNames = append(structTypeNames, typeName)
				debugf("Added struct type from typeDefs: %s", typeName)
			}
		}
	}

	// Also collect imported struct types from field types
	for _, typeInfo := range types {
		for _, field := range typeInfo.Fields {
			debugf("Processing field %s.%s: Type=%s, XDRType=%s", typeInfo.Name, field.Name, field.Type, field.XDRType)

			// Check if this field type is an imported struct
			if strings.Contains(field.Type, ".") {
				// Extract the type name from package.Type format
				parts := strings.Split(field.Type, ".")
				if len(parts) == 2 {
					importedTypeName := parts[1]
					debugf("Found imported type: %s (from %s)", importedTypeName, field.Type)
					// Check if this imported type is not already in the list
					found := false
					for _, existing := range structTypeNames {
						if existing == importedTypeName {
							found = true
							break
						}
					}
					if !found {
						// For imported types, we assume they are structs if they're used with xdr:"struct"
						// or if they appear in array contexts
						if field.XDRType == "struct" || field.XDRType == "array" {
							structTypeNames = append(structTypeNames, importedTypeName)
							debugf("Added imported struct type: %s", importedTypeName)
						}
					}
				}
			}

			// Also check for imported types in array element types
			if field.XDRType == "array" && strings.Contains(field.Type, "[]") {
				elementType := strings.TrimPrefix(field.Type, "[]")
				debugf("Array field %s.%s has element type: %s", typeInfo.Name, field.Name, elementType)
				if strings.Contains(elementType, ".") {
					// Extract the type name from package.Type format
					parts := strings.Split(elementType, ".")
					if len(parts) == 2 {
						importedElementTypeName := parts[1]
						debugf("Found imported array element type: %s (from %s)", importedElementTypeName, elementType)
						// Check if this imported element type is not already in the list
						found := false
						for _, existing := range structTypeNames {
							if existing == importedElementTypeName {
								found = true
								break
							}
						}
						if !found {
							// For imported types used as array elements, we assume they are structs
							structTypeNames = append(structTypeNames, importedElementTypeName)
							debugf("Added imported array element struct type: %s", importedElementTypeName)
						}
					}
				} else {
					// This is a type from the same package (no dot), check if it's not in typeDefs
					if _, exists := typeDefs[elementType]; !exists {
						// Type not found in current file, assume it's a struct from another file in the package
						found := false
						for _, existing := range structTypeNames {
							if existing == elementType {
								found = true
								break
							}
						}
						if !found {
							structTypeNames = append(structTypeNames, elementType)
							debugf("Added cross-file struct type from same package: %s", elementType)
						}
					}
				}
			}
		}
	}

	// For now, just print debug info and exit
	debugf("Parsed %d types", len(types))
	for _, typeInfo := range types {
		debugf("Type %s: IsDiscriminatedUnion=%v, UnionConfig=%+v",
			typeInfo.Name, typeInfo.IsDiscriminatedUnion, typeInfo.UnionConfig)
	}

	debugf("Final structTypeNames list (%d types): %v", len(structTypeNames), structTypeNames)

	// Create code generator
	codeGen, err := NewCodeGenerator(structTypeNames)
	if err != nil {
		log.Fatal("Error creating code generator:", err)
	}

	// Generate output
	var output strings.Builder

	// Generate file header
	header, err := codeGen.GenerateFileHeader(inputFile, packageName, externalImports, buildTags)
	if err != nil {
		log.Fatal("Error generating file header:", err)
	}
	output.WriteString(header)
	output.WriteString("\n")

	// Generate code for each type
	for _, typeInfo := range types {
		// Generate encode method
		encodeMethod, err := codeGen.GenerateEncodeMethod(typeInfo)
		if err != nil {
			log.Fatal("Error generating encode method:", err)
		}
		output.WriteString(encodeMethod)
		output.WriteString("\n")

		// Generate decode method
		decodeMethod, err := codeGen.GenerateDecodeMethod(typeInfo)
		if err != nil {
			log.Fatal("Error generating decode method:", err)
		}
		output.WriteString(decodeMethod)
		output.WriteString("\n")

		// Generate compile-time assertion
		assertion, err := codeGen.GenerateAssertion(typeInfo.Name)
		if err != nil {
			log.Fatal("Error generating assertion:", err)
		}
		output.WriteString(assertion)
		output.WriteString("\n")
	}

	// Write to output file
	if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
		log.Fatal("Error writing output file:", err)
	}

	if !silent {
		logf("Generated XDR methods for %d types in %s", len(types), outputFile)
	}
}

// validateDiscriminatedUnions validates that discriminated union structures are correctly formed
func validateDiscriminatedUnions(types []TypeInfo, constants map[string]string) error {
	for _, typeInfo := range types {
		if typeInfo.IsDiscriminatedUnion {
			keyCount := 0
			unionCount := 0
			keyIndex := -1
			unionIndex := -1

			// Count key and union fields and find their positions
			for i, field := range typeInfo.Fields {
				if field.IsKey {
					keyCount++
					keyIndex = i
				}
				if field.IsUnion {
					unionCount++
					unionIndex = i
				}
			}

			// Validate discriminated union structure
			if keyCount != 1 {
				return fmt.Errorf("Discriminated union struct %s must have exactly one key field, found %d", typeInfo.Name, keyCount)
			}
			if unionCount == 0 {
				return fmt.Errorf("Discriminated union struct %s must have at least one union field", typeInfo.Name)
			}

			// Validate ordered pair requirement: key must be immediately followed by union
			if unionIndex != keyIndex+1 {
				return fmt.Errorf("Discriminated union struct %s: key field must be immediately followed by union field", typeInfo.Name)
			}

			logf("Validated discriminated union %s: %d key, %d union fields", typeInfo.Name, keyCount, unionCount)
		}
	}
	return nil
}
