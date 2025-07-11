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
// Discriminated unions follow XDR standard format: discriminant + union data.
//
// LIMITATION: Only uint32 discriminants are supported. This design choice optimizes
// for binary protocol efficiency and matches common protocol conventions. String
// discriminants would require variable-length encoding and break zero-allocation
// design goals. Other integer types (uint64, int32) are not supported to maintain
// API simplicity.
//
//	`xdr:"discriminant"`              - marks uint32 discriminant field
//	`xdr:"union:pattern"`             - auto-discover based on discriminant constants
//	`xdr:"union:VALUE"`               - single discriminant value maps to this field
//	`xdr:"union:VALUE1,VALUE2"`       - multiple values map to this field
//	`xdr:"union:VALUE,void:default"`  - VALUE maps to field, others are void
//	`xdr:"union:void:VALUE"`          - VALUE is void, others use this field
//	`xdr:"union:default"`             - handles all unmatched discriminant values
//	`xdr:"union:pattern,void:OpX"`    - pattern discovery with specific voids
//
// Pattern Discovery (`union:pattern`):
//
//	Uses AST parsing to discover discriminant constants and matching structs:
//
//	1. Parse Go AST to find all constants of discriminant type (e.g., OpCode)
//	2. For each constant like "OpAccess", extract base name "Access"
//	3. Search AST for structs matching naming patterns:
//	   - "AccessArgs" for operation arguments
//	   - "AccessResult" for operation results
//	4. Generate switch cases for found structs, void cases for missing structs
//
//	Discovery Algorithm:
//	- OpAccess (3) → looks for AccessArgs/AccessResult structs
//	- OpGetFH (10) → looks for GetFHArgs/GetFHResult structs
//	- OpSaveFH (32) → no SaveFHArgs exists → void case (encodes nothing)
//
//	Example:
//	type OperationArg struct {
//	    OpCode OpCode `xdr:"discriminant"`
//	    Args   []byte `xdr:"union:pattern"`  // Auto-discovers all OpXxxArgs
//	}
//
//	Generated switch includes cases for every OpXxx constant that has XxxArgs struct.
//	Build-time validation ensures all referenced structs have proper XDR tags.
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

// logf logs a message unless in silent mode
func logf(format string, args ...any) {
	if !silent {
		log.Printf(format, args...)
	}
}

// FieldInfo represents a struct field with XDR encoding information
type FieldInfo struct {
	Name           string
	Type           string
	XDRType        string
	Tag            string
	IsDiscriminant bool   // true if this field is a discriminated union discriminant
	UnionSpec      string // union specification (e.g., "SUCCESS,void:default")
}

// TypeInfo represents a struct that needs XDR generation
type TypeInfo struct {
	Name                 string
	Fields               []FieldInfo
	IsDiscriminatedUnion bool // true if this struct represents a discriminated union
}

// UnionCase represents a single case in a discriminated union
type UnionCase struct {
	Values []string // Discriminant values for this case
	IsVoid bool     // True if this is a void case
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

// parseDiscriminatedUnionTag parses discriminated union tags
// Discriminants are uint32 by default for protocol efficiency
func parseDiscriminatedUnionTag(xdrTag string) (isDiscriminant bool, unionSpec string) {
	if xdrTag == "discriminant" {
		return true, ""
	}
	if strings.HasPrefix(xdrTag, "union:") {
		return false, strings.TrimPrefix(xdrTag, "union:")
	}
	return false, ""
}

// isSupportedXDRType checks if an XDR type is supported by the generator
func isSupportedXDRType(xdrType string) bool {
	switch xdrType {
	case "uint32", "uint64", "int64", "string", "bytes", "bool", "struct", "array", "discriminant":
		return true
	default:
		return strings.HasPrefix(xdrType, "fixed:") || strings.HasPrefix(xdrType, "union:") || strings.HasPrefix(xdrType, "alias:")
	}
}

// collectExternalImports scans TypeInfo for external package dependencies
func collectExternalImports(types []TypeInfo) []string {
	imports := make(map[string]bool)

	for _, typeInfo := range types {
		for _, field := range typeInfo.Fields {
			// Extract package qualifiers from field type (handles arrays/slices too)
			extractPackageImports(field.Type, imports)
		}
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}

	return result
}

// extractPackageImports extracts package imports from a type string
// Handles cases like: "primitives.NFSImplID", "[]primitives.NFSImplID", "map[string]primitives.Type"
func extractPackageImports(typeStr string, imports map[string]bool) {
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
					imports["github.com/tempusfrangit/go-xdr/primitives"] = true
				case "session":
					imports["github.com/tempusfrangit/go-xdr/session"] = true
					// Add other packages as needed
				}
			}
		}
	}
}

// parseUnionSpec parses union specification strings
// Examples:
//
//	"SUCCESS,void:default" -> case SUCCESS (data), default void
//	"VALUE1,VALUE2" -> cases VALUE1,VALUE2 (data)
//	"void:VALUE" -> case VALUE (void)
func parseUnionSpec(spec string) []UnionCase {
	var cases []UnionCase

	// Split by commas and parse each part
	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "void:") {
			// Void case specification
			value := strings.TrimPrefix(part, "void:")
			cases = append(cases, UnionCase{
				Values: []string{value},
				IsVoid: true,
			})
		} else {
			// Data case - check if it's a group or single value
			cases = append(cases, UnionCase{
				Values: []string{part},
				IsVoid: false,
			})
		}
	}

	return cases
}

// hasDefaultCase checks if the union cases include a default case
func hasDefaultCase(cases []UnionCase) bool {
	for _, unionCase := range cases {
		for _, value := range unionCase.Values {
			if value == "default" {
				return true
			}
		}
	}
	return false
}

// findDiscriminantField finds the discriminant field name in the given struct
func findDiscriminantField(structInfo TypeInfo) string {
	for _, field := range structInfo.Fields {
		if field.IsDiscriminant {
			return field.Name
		}
	}
	return ""
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
func extractBuildTags(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
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

	return buildTags, nil
}

// parseFile parses a Go file and extracts structs that have go:generate xdrgen directives
func parseFile(filename string) ([]TypeInfo, error) {
	fset := token.NewFileSet()

	// For files with build tags, we need to parse them with the appropriate build context
	// For now, we'll parse all files regardless of build tags to ensure we don't miss any
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

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
			if strings.HasPrefix(text, "go:generate") && strings.Contains(text, "xdrgen") && strings.Contains(text, "$GOFILE") {
				hasGOFILEDirective = true
				break
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

	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := node.Type.(*ast.StructType); ok {
				typeInfo := TypeInfo{
					Name: node.Name.Name,
				}

				// Check if this struct should be generated
				shouldGenerate := false
				if hasGOFILEDirective {
					// For $GOFILE directive, check if struct has XDR tags
					for _, field := range structType.Fields.List {
						if field.Tag != nil {
							tag := strings.Trim(field.Tag.Value, "`")
							if strings.Contains(tag, "xdr:") {
								shouldGenerate = true
								break
							}
						}
					}
				} else {
					// Check explicit struct name in go:generate directives
					shouldGenerate = structsToGenerate[typeInfo.Name]
				}

				if !shouldGenerate {
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
						isDiscriminant, unionSpec := parseDiscriminatedUnionTag(fieldInfo.XDRType)
						fieldInfo.IsDiscriminant = isDiscriminant
						fieldInfo.UnionSpec = unionSpec

						// Mark struct as discriminated union if it contains discriminant field
						if isDiscriminant {
							typeInfo.IsDiscriminatedUnion = true
						}

						// Validate that we can handle this XDR type
						if !isSupportedXDRType(fieldInfo.XDRType) {
							log.Fatalf("Unsupported XDR type '%s' for field %s.%s. Supported types: uint32, uint64, string, bytes, bool, stateid, sessionid, discriminant, union:..., fixed:N. Add support or implement custom Encode/Decode methods.",
								fieldInfo.XDRType, typeInfo.Name, fieldInfo.Name)
						}

						typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
					}
				}

				if len(typeInfo.Fields) > 0 {
					types = append(types, typeInfo)
				}
			}
		}
		return true
	})

	return types, nil
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
		return "", fmt.Errorf("unsupported type for alias inference: %s (supported: string, []byte, uint32, uint64, int32, int64, bool)", goType)
	}
}

func main() {
	// Configure log to remove timestamps for cleaner output
	log.SetFlags(0)

	flag.BoolVar(&silent, "silent", false, "suppress all output except errors")
	flag.BoolVar(&silent, "s", false, "suppress all output except errors (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "xdrgen - XDR Code Generator\n\n")
		fmt.Fprintf(os.Stderr, "Generates Encode and Decode methods for Go structs with XDR struct tags.\n")
		fmt.Fprintf(os.Stderr, "Supports the xdr.Codec interface for automatic XDR serialization/deserialization.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  xdrgen [flags] <input.go>\n\n")
		fmt.Fprintf(os.Stderr, "The generator processes structs that are explicitly opted in via go:generate directives:\n")
		fmt.Fprintf(os.Stderr, "  //go:generate xdrgen $GOFILE              (process all XDR-tagged structs in file)\n")
		fmt.Fprintf(os.Stderr, "  //go:generate xdrgen StructName           (process specific struct)\n")
		fmt.Fprintf(os.Stderr, "  //go:generate ../../tools/bin/xdrgen StructName  (with path)\n\n")
		fmt.Fprintf(os.Stderr, "Supported XDR struct tags:\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"uint32\"`     - 32-bit unsigned integer\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"uint64\"`     - 64-bit unsigned integer\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"string\"`     - variable-length string\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"bytes\"`      - variable-length byte array\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"bool\"`       - boolean (encoded as uint32)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"struct\"`     - nested struct (must implement xdr.Codec)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"array\"`      - variable-length array\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"fixed:N\"`    - fixed-size byte array (N bytes)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"alias\"`       - type alias with automatic type inference (Go type must be: string, []byte, uint32, uint64, int32, int64, bool)\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"-\"`          - exclude field from encoding/decoding\n\n")
		fmt.Fprintf(os.Stderr, "Output:\n")
		fmt.Fprintf(os.Stderr, "  Creates <input>_xdr.go with generated Encode/Decode methods\n")
		fmt.Fprintf(os.Stderr, "  Includes compile-time assertions that types implement xdr.Codec\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  xdrgen types.go         # Generate for all XDR-tagged structs\n")
		fmt.Fprintf(os.Stderr, "  xdrgen -s types.go      # Generate silently (no output except errors)\n")
		fmt.Fprintf(os.Stderr, "  go generate            # Use with go:generate directives\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// Convert to absolute path
	inputFile, err := filepath.Abs(inputFile)
	if err != nil {
		log.Fatal("Error getting absolute path:", err)
	}

	// Generate output file name, handling test files specially
	var outputFile string
	if strings.HasSuffix(inputFile, "_test.go") {
		outputFile = strings.TrimSuffix(inputFile, "_test.go") + "_xdr_test.go"
	} else {
		outputFile = strings.TrimSuffix(inputFile, ".go") + "_xdr.go"
	}

	types, err := parseFile(inputFile)
	if err != nil {
		log.Fatal("Error parsing file:", err)
	}

	if len(types) == 0 {
		logf("No types with XDR tags found in %s", inputFile)
		return
	}

	// Validate discriminated union structures
	validateDiscriminatedUnions(types)

	// Extract package name from input file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		log.Fatal("Error parsing package:", err)
	}

	// Collect external package dependencies
	externalImports := collectExternalImports(types)

	// Extract build tags from input file
	buildTags, err := extractBuildTags(inputFile)
	if err != nil {
		log.Fatal("Error extracting build tags:", err)
	}
	if len(buildTags) > 0 {
		logf("Found build tags: %v", buildTags)
	}

	// Generate the output file using templates
	var output strings.Builder

	// Generate file header
	header, err := GenerateFileHeader(filepath.Base(inputFile), file.Name.Name, externalImports, buildTags)
	if err != nil {
		log.Fatal("Error generating file header:", err)
	}
	output.WriteString(header)
	output.WriteString("\n")

	// Generate methods for each type
	for _, typeInfo := range types {
		// Generate encode method
		encodeMethod, err := GenerateEncodeMethod(typeInfo)
		if err != nil {
			log.Fatal("Error generating encode method:", err)
		}
		output.WriteString(encodeMethod)
		output.WriteString("\n")

		// Generate decode method
		decodeMethod, err := GenerateDecodeMethod(typeInfo)
		if err != nil {
			log.Fatal("Error generating decode method:", err)
		}
		output.WriteString(decodeMethod)
		output.WriteString("\n")

		// Generate compile-time assertion
		assertion, err := GenerateAssertion(typeInfo.Name)
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

	logf("Generated XDR methods for %d types in %s", len(types), outputFile)
}

// validateDiscriminatedUnions validates that discriminated union structures are correctly formed
func validateDiscriminatedUnions(types []TypeInfo) {
	for _, typeInfo := range types {
		if typeInfo.IsDiscriminatedUnion {
			discriminantCount := 0
			unionCount := 0

			// Count discriminant and union fields
			for _, field := range typeInfo.Fields {
				if field.IsDiscriminant {
					discriminantCount++
				}
				if field.UnionSpec != "" {
					unionCount++
				}
			}

			// Validate discriminated union structure
			if discriminantCount != 1 {
				log.Fatalf("Discriminated union struct %s must have exactly one discriminant field, found %d", typeInfo.Name, discriminantCount)
			}
			if unionCount == 0 {
				log.Fatalf("Discriminated union struct %s must have at least one union field", typeInfo.Name)
			}

			logf("Validated discriminated union %s: %d discriminant, %d union fields", typeInfo.Name, discriminantCount, unionCount)
		}
	}
}
