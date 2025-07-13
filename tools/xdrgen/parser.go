package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

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

// parseFileWithPackageTypeDefs parses a Go file with package-level type definitions
func parseFileWithPackageTypeDefs(filename string, packageTypeDefs map[string]ast.Node, packageConstants map[string]string) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
	return parseFileInternal(filename, packageTypeDefs, packageConstants)
}

// parseFile parses a Go file and extracts structs that have go:generate xdrgen directives
func parseFile(filename string) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
	return parseFileInternal(filename, nil, nil)
}

// parseFileInternal is the internal implementation that can optionally use package-level type definitions
func parseFileInternal(filename string, packageTypeDefs map[string]ast.Node, packageConstants map[string]string) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
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

	// If package-level type definitions are provided, include them
	if packageTypeDefs != nil {
		for typeName, typeNode := range packageTypeDefs {
			if _, exists := typeAliases[typeName]; !exists {
				if typeExpr, ok := typeNode.(ast.Expr); ok {
					underlyingType := formatType(typeExpr)
					typeAliases[typeName] = underlyingType
				}
			}
		}
	}

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
								if !ok {
									log.Fatalf("Key field %s.%s references type %s which is not defined in this file. For cross-file dependencies, process the entire package directory instead of individual files", typeInfo.Name, fieldInfo.Name, fieldInfo.Type)
								} else {
									log.Fatalf("Key field %s.%s must be uint32 or an alias of uint32, got %s (resolves to %s)", typeInfo.Name, fieldInfo.Name, fieldInfo.Type, underlyingType)
								}
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
			var typedConstants map[string]string
			if packageConstants != nil {
				// Use package-level constants for cross-file processing
				typedConstants = packageConstants
			} else {
				// Use file-level constants for single-file processing
				typedConstants = collectTypedConstants(file, keyField.Type)
			}

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
