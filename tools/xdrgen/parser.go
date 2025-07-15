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

// isSupportedXDRType checks if an auto-detected XDR type is supported by the generator
func isSupportedXDRType(xdrType string) bool {
	switch xdrType {
	case "uint32", "uint64", "int32", "int64", "string", "bytes", "bool", "struct", "array":
		return true
	default:
		// Handle complex types with prefixes
		if strings.HasPrefix(xdrType, "fixed:") || strings.HasPrefix(xdrType, "alias:") {
			return true
		}
		return false
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

// parseKeyTag parses key tags with optional default parameter
// Supports: "key" and "key,default=nil" and "key,default=StructName"
func parseKeyTag(xdrTag string) (isKey bool, defaultType string) {
	if xdrTag == "key" {
		return true, ""
	}
	if strings.HasPrefix(xdrTag, "key,") {
		// Parse key with options like "key,default=nil" or "key,default=ErrorResult"
		parts := strings.Split(xdrTag, ",")
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "default=") {
				defaultType = strings.TrimPrefix(part, "default=")
				// Validate default value
				if defaultType != "nil" && defaultType == "" {
					return false, "" // Invalid empty default
				}
			}
		}
		return true, defaultType
	}
	return false, ""
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

		if !strings.HasPrefix(part, "case=") {
			return nil, fmt.Errorf("unknown union comment part: %s", comment)
		}
		// Parse case - just the constant name, struct name is implicit
		constantName := strings.TrimSpace(strings.TrimPrefix(part, "case="))
		if constantName == "" {
			return nil, fmt.Errorf("empty case name in union comment: %s", comment)
		}
		// The struct name will be filled in when we associate the comment with the struct
		config.Cases[constantName] = "" // Will be filled in later
	}

	return config, nil
}

// ConstantInfo holds information about a constant
type ConstantInfo struct {
	Value string
	Type  string
}

// collectConstants collects all constant definitions from the AST
func collectConstants(file *ast.File) map[string]ConstantInfo {
	constants := make(map[string]ConstantInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					// Get the type if specified
					var typeName string
					if valueSpec.Type != nil {
						if ident, ok := valueSpec.Type.(*ast.Ident); ok {
							typeName = ident.Name
						}
					}

					for i, name := range valueSpec.Names {
						if i < len(valueSpec.Values) {
							if basicLit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
								constants[name.Name] = ConstantInfo{
									Value: basicLit.Value,
									Type:  typeName,
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
func parseFileWithPackageTypeDefs(filename string, packageTypeDefs map[string]ast.Node, packageConstants map[string]ConstantInfo) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
	return parseFileInternal(filename, packageTypeDefs, packageConstants)
}

// parseFile parses a Go file and extracts structs that have go:generate xdrgen directives
func parseFile(filename string) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
	return parseFileInternal(filename, nil, nil)
}

// resolveAliasType recursively resolves alias chains to their underlying primitive type
func resolveAliasType(goType string, typeAliases map[string]string) string {
	seen := make(map[string]bool)
	current := goType

	// Handle slice types: []TypeName -> []ResolvedType
	if strings.HasPrefix(current, "[]") {
		elementType := strings.TrimPrefix(current, "[]")
		resolvedElement := resolveAliasType(elementType, typeAliases)
		return "[]" + resolvedElement
	}

	// Handle fixed array types: [N]TypeName -> [N]ResolvedType
	if strings.HasPrefix(current, "[") && strings.Contains(current, "]") {
		closeBracket := strings.Index(current, "]")
		if closeBracket > 0 {
			prefix := current[:closeBracket+1]      // [N]
			elementType := current[closeBracket+1:] // TypeName
			resolvedElement := resolveAliasType(elementType, typeAliases)
			return prefix + resolvedElement
		}
	}

	for {
		if seen[current] {
			// Cycle detected, return struct
			return "struct"
		}
		seen[current] = true

		// Check if this is a known alias
		if underlying, exists := typeAliases[current]; exists {
			current = underlying
			continue
		}

		// No more aliases to resolve
		break
	}

	return current
}

// autoDiscoverXDRType maps Go types to their natural XDR equivalents
func autoDiscoverXDRType(goType string, typeAliases map[string]string) string {
	// First resolve any alias chains
	resolvedType := resolveAliasType(goType, typeAliases)

	switch resolvedType {
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
	case "[]byte":
		return "bytes"
	case "bool":
		return "bool"
	default:
		// Handle array types - return element type for array encoding
		if strings.HasPrefix(resolvedType, "[]") {
			elementType := strings.TrimPrefix(resolvedType, "[]")
			return autoDiscoverXDRType(elementType, typeAliases) // Recurse for element type
		}

		// Handle fixed arrays - return element type for array encoding
		if strings.HasPrefix(resolvedType, "[") && strings.Contains(resolvedType, "]") {
			closeBracket := strings.Index(resolvedType, "]")
			elementType := resolvedType[closeBracket+1:]
			// Special case: [N]byte should be "bytes" not "struct"
			if elementType == "byte" {
				return "bytes"
			}
			return autoDiscoverXDRType(elementType, typeAliases) // Recurse for element type
		}

		// For custom types (structs, aliases), assume struct
		return "struct"
	}
}

// parseXDRDirective parses kubebuilder-style // +xdr: comments
func parseXDRDirective(comment string) (directive string, args map[string]string, isList bool) {
	comment = strings.TrimSpace(comment)
	if !strings.HasPrefix(comment, "// +xdr:") {
		return "", nil, false
	}

	// Remove "// +xdr:" prefix
	content := strings.TrimPrefix(comment, "// +xdr:")

	// Handle both old and new format
	args = make(map[string]string)

	// Check if this is new comma-separated format or old equals format
	if strings.Contains(content, ",") {
		// New format: +xdr:union,key=Type,default=MessageTypeVoid
		parts := strings.Split(content, ",")
		directive = strings.TrimSpace(parts[0])

		// Parse key=value pairs
		for _, part := range parts[1:] {
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				args[key] = value
			}
		}
	} else {
		// Old format: +xdr:union=OpType
		parts := strings.SplitN(content, "=", 2)
		directive = strings.TrimSpace(parts[0])

		if len(parts) > 1 {
			args["value"] = strings.TrimSpace(parts[1])
		}
	}

	return directive, args, true
}

// XDRDirectives holds all XDR directives for a struct
type XDRDirectives struct {
	Generate bool
	Union    *UnionDirective
	Payload  *PayloadDirective
}

// UnionDirective represents +xdr:union directive
type UnionDirective struct {
	Key     string // key field name
	Default string // default case (optional)
}

// PayloadDirective represents +xdr:payload directive
type PayloadDirective struct {
	Union        string // union type name
	Discriminant string // discriminant value
}

// parseUnionDirective parses +xdr:union directive
// Format: key=FieldName,default=DefaultValue
func parseUnionDirective(args map[string]string) *UnionDirective {
	directive := &UnionDirective{}

	if key, ok := args["key"]; ok {
		directive.Key = key
	}
	if defaultVal, ok := args["default"]; ok {
		directive.Default = defaultVal
	}

	return directive
}

// parsePayloadDirective parses +xdr:payload directive
// Format: union=UnionType,discriminant=DiscriminantValue
func parsePayloadDirective(args map[string]string) *PayloadDirective {
	directive := &PayloadDirective{}

	if union, ok := args["union"]; ok {
		directive.Union = union
	}
	if discriminant, ok := args["discriminant"]; ok {
		directive.Discriminant = discriminant
	}

	return directive
}

// collectXDRDirectives collects all // +xdr: directives immediately before a struct
func collectXDRDirectives(file *ast.File, structPos token.Pos) *XDRDirectives {
	directives := &XDRDirectives{}

	// Find the comment group that immediately precedes this struct
	var targetCommentGroup *ast.CommentGroup
	for _, commentGroup := range file.Comments {
		// Check if this comment group ends just before our struct (with possible whitespace)
		if commentGroup.End() < structPos {
			targetCommentGroup = commentGroup
		} else if commentGroup.Pos() > structPos {
			// We've gone past the struct, stop looking
			break
		}
	}

	// Only process the immediately preceding comment group
	if targetCommentGroup != nil {
		for _, comment := range targetCommentGroup.List {
			debugf("Processing comment: %s", comment.Text)
			directive, args, isXDR := parseXDRDirective(comment.Text)
			if !isXDR {
				debugf("Not an XDR directive: %s", comment.Text)
				continue
			}
			debugf("Found XDR directive: %s with args: %v", directive, args)

			switch directive {
			case "generate":
				directives.Generate = true
				debugf("Found +xdr:generate directive")
			case "union":
				directives.Union = parseUnionDirective(args)
				debugf("Found +xdr:union directive with args: %v", args)
			case "payload":
				directives.Payload = parsePayloadDirective(args)
				debugf("Found +xdr:payload directive with args: %v", args)
			}
		}
	}

	return directives
}

// parseFileInternal is the internal implementation that can optionally use package-level type definitions
func parseFileInternal(filename string, packageTypeDefs map[string]ast.Node, packageConstants map[string]ConstantInfo) ([]TypeInfo, []ValidationError, map[string]ast.Node, error) {
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
		if genDecl, ok := n.(*ast.GenDecl); ok {
			// Store the full GenDecl which contains the directive comments
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					typeDefs[typeSpec.Name.Name] = genDecl
				}
			}
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
	for typeName, typeNode := range packageTypeDefs {
		if _, exists := typeAliases[typeName]; !exists {
			if typeExpr, ok := typeNode.(ast.Expr); ok {
				underlyingType := formatType(typeExpr)
				typeAliases[typeName] = underlyingType
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
		// Process all structs with XDR tags in this case
		debugf("No go:generate directive found, will process all structs with XDR tags")
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
				// Collect XDR directives for this struct
				xdrDirectives := collectXDRDirectives(file, node.Pos())

				// Mark key field if this is a union container (do this before processing fields)
				if xdrDirectives.Union != nil && xdrDirectives.Union.Key != "" {
					typeInfo.IsDiscriminatedUnion = true
					debugf("Struct %s is a discriminated union container (has +xdr:union directive)", typeInfo.Name)
				}

				// Auto-generate for union containers and payload types
				shouldGenerate := xdrDirectives.Generate || xdrDirectives.Union != nil || xdrDirectives.Payload != nil
				if shouldGenerate {
					if xdrDirectives.Generate {
						debugf("Struct %s has +xdr:generate directive", typeInfo.Name)
					}
					if xdrDirectives.Union != nil {
						debugf("Struct %s has +xdr:union directive - auto-generating", typeInfo.Name)
					}
					if xdrDirectives.Payload != nil {
						debugf("Struct %s has +xdr:payload directive - auto-generating", typeInfo.Name)
					}
				} else {
					debugf("Struct %s - no generation needed: Generate=%v, Union=%v, Payload=%v", typeInfo.Name, xdrDirectives.Generate, xdrDirectives.Union != nil, xdrDirectives.Payload != nil)
				}

				// Union container detection is now handled during field processing

				// Store XDR directives for payload structs (for later cross-file processing)
				if shouldGenerate && (xdrDirectives.Union != nil || xdrDirectives.Payload != nil) {
					if xdrDirectives.Union != nil {
						debugf("Struct %s has union directive: key=%s, default=%s", typeInfo.Name, xdrDirectives.Union.Key, xdrDirectives.Union.Default)
					}
					if xdrDirectives.Payload != nil {
						debugf("Struct %s has payload directive: union=%s, discriminant=%s", typeInfo.Name, xdrDirectives.Payload.Union, xdrDirectives.Payload.Discriminant)
						// Mark this struct as a payload
						typeInfo.IsPayload = true
						typeInfo.PayloadConfig = &PayloadConfig{
							UnionType:    xdrDirectives.Payload.Union,
							Discriminant: xdrDirectives.Payload.Discriminant,
						}
					}
				}

				if !shouldGenerate {
					debugf("Skipping struct %s - no +xdr:generate directive", typeInfo.Name)
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
					// Skip embedded fields for now
					if len(field.Names) == 0 {
						continue
					}

					for _, name := range field.Names {
						fieldType := formatType(field.Type)
						fieldInfo := FieldInfo{
							Name:         name.Name,
							Type:         fieldType,
							ResolvedType: resolveAliasType(fieldType, typeAliases),
						}

						// Parse XDR tag if present
						xdrTag := ""
						if field.Tag != nil {
							tag := strings.Trim(field.Tag.Value, "`")
							fieldInfo.Tag = tag
							xdrTag = parseXDRTag(tag)
						}

						// Check for explicit exclusion (like json:"-")
						if xdrTag == "-" {
							continue // Skip this field
						}

						// Handle XDR tags in minimal mode (only xdr:"-" is supported)
						if xdrTag != "" {
							// Check for skip tag first
							if xdrTag == "-" {
								continue // Skip this field
							}
							// All other struct tags are ignored - we use directives and auto-detection
						}

						// Check if this field is the key field specified in the directive
						if xdrDirectives.Union != nil && xdrDirectives.Union.Key == fieldInfo.Name {
							fieldInfo.IsKey = true
							debugf("Marked field %s as key field from +xdr:union directive", fieldInfo.Name)
						}

						// Auto-discover XDR type from Go type
						autoType := autoDiscoverXDRType(fieldInfo.Type, typeAliases)
						fieldInfo.XDRType = autoType
						debugf("Auto-discovered XDR type for %s.%s: %s -> %s", typeInfo.Name, fieldInfo.Name, fieldInfo.Type, autoType)

						// Auto-detect union field: []byte immediately following a key field
						if len(typeInfo.Fields) > 0 && typeInfo.Fields[len(typeInfo.Fields)-1].IsKey && fieldInfo.Type == "[]byte" {
							fieldInfo.IsUnion = true
							// Union fields should be treated as bytes for encoding/decoding
							fieldInfo.XDRType = "bytes"
							debugf("Auto-detected union field %s.%s ([]byte following key field)", typeInfo.Name, fieldInfo.Name)
						}

						// Mark struct as discriminated union if it contains key field
						if fieldInfo.IsKey {
							typeInfo.IsDiscriminatedUnion = true
							// Key fields must be uint32 type or an alias of uint32
							underlyingType, ok := typeAliases[fieldInfo.Type]
							if fieldInfo.Type != "uint32" && (!ok || underlyingType != "uint32") {
								if !ok {
									log.Fatalf("Key field %s.%s references type %s which is not defined in this file. For cross-file dependencies, process the entire package directory instead of individual files", typeInfo.Name, fieldInfo.Name, fieldInfo.Type)
								}
								log.Fatalf("Key field %s.%s must be uint32 or an alias of uint32, got %s (resolves to %s)", typeInfo.Name, fieldInfo.Name, fieldInfo.Type, underlyingType)
							}
							// Key fields should be treated as uint32 for encoding/decoding
							fieldInfo.XDRType = "uint32"
						}

						// Validate that we can handle this XDR type (after key/union processing)
						if !isSupportedXDRType(fieldInfo.XDRType) {
							log.Fatalf("Unsupported XDR type '%s' for field %s.%s. Supported types: uint32, uint64, int32, int64, string, bytes, bool, struct, key, union. Arrays are auto-detected from []Type and [N]Type syntax.",
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
				// Filter constants by type
				typedConstants = make(map[string]string)
				for name, constantInfo := range packageConstants {
					if constantInfo.Type == keyField.Type {
						typedConstants[name] = constantInfo.Value
					}
				}
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

// getUnionFieldNames extracts the key and payload field names from a union struct AST node
func getUnionFieldNames(unionTypeName string, typeDefs map[string]ast.Node) (keyField, payloadField string) {
	node, exists := typeDefs[unionTypeName]
	if !exists {
		return "", ""
	}

	// Extract fields from the struct declaration and get the key field from the directive
	var structType *ast.StructType
	var keyFieldFromDirective string

	switch n := node.(type) {
	case *ast.GenDecl:
		// Check for +xdr:union,key=FieldName directive in the comments
		if n.Doc != nil {
			for _, comment := range n.Doc.List {
				if strings.Contains(comment.Text, "+xdr:union,key=") {
					// Extract the key field name from the directive
					parts := strings.Split(comment.Text, "key=")
					if len(parts) > 1 {
						keyFieldFromDirective = parts[1]
						// Remove any trailing commas or whitespace
						keyFieldFromDirective = strings.TrimSpace(strings.Split(keyFieldFromDirective, ",")[0])
					}
				}
			}
		}

		for _, spec := range n.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == unionTypeName {
				if st, ok := typeSpec.Type.(*ast.StructType); ok {
					structType = st
					break
				}
			}
		}
	case *ast.TypeSpec:
		if st, ok := n.Type.(*ast.StructType); ok {
			structType = st
		}
	case *ast.StructType:
		// Direct struct type - this shouldn't happen for union types with directives
		structType = n
	}

	if structType == nil {
		return "", ""
	}

	// Use the key field name from the directive, or fall back to looking for xdr:"key" tags
	if keyFieldFromDirective != "" {
		keyField = keyFieldFromDirective
	} else {
		// Fallback: look for xdr:"key" tags in struct field tags
		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}
			tag := strings.Trim(field.Tag.Value, "`")

			// Check if this is a key field
			if strings.Contains(tag, `xdr:"key"`) {
				if len(field.Names) > 0 {
					keyField = field.Names[0].Name
				}
			}
		}
	}

	// Find the payload field (auto-detect as the first non-key field)
	if keyField != "" {
		for _, field := range structType.Fields.List {
			if len(field.Names) > 0 && field.Names[0].Name != keyField {
				payloadField = field.Names[0].Name
				break
			}
		}
	}

	return keyField, payloadField
}
