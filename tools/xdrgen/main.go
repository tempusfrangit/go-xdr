// XDR Code Generator - Ultra-Minimal Edition
//
// Generates XDR encoding/decoding methods with maximum auto-detection and minimal tagging.
// Only 2 tag types: `xdr:"key[,default=X]"` and `xdr:"-"`. Everything else auto-detected.
//
// Usage: xdrgen [flags] <go-files...>
//
// Examples:
//
//	// +xdr:generate
//	type User struct {
//	    ID     UserID           // auto-detected as string
//	    Hash   [32]byte         // auto-detected as bytes
//	    Secret string `xdr:"-"` // excluded
//	}
//
//	// +xdr:generate
//	type Operation struct {
//	    OpCode OpCode `xdr:"key"` // union discriminant
//	    Result []byte            // auto-detected union payload
//	}
//
// See help.txt for complete documentation.
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

	// Cross-file union configuration gathering is already implemented above
	// through the union comment collection and association logic

	return fileTypes, fileTypeDefs, nil
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
		fmt.Fprintf(os.Stderr, "  xdrgen types.go         # Generate for single file (must be self-contained)\n")
		fmt.Fprintf(os.Stderr, "  xdrgen -s ./types/      # Generate silently (no output except errors)\n\n")
		fmt.Fprintf(os.Stderr, "Cross-file Dependencies:\n")
		fmt.Fprintf(os.Stderr, "  Single file mode requires all types to be defined in the same file.\n")
		fmt.Fprintf(os.Stderr, "  For cross-file dependencies (e.g., discriminant types in other files),\n")
		fmt.Fprintf(os.Stderr, "  process the entire package directory instead of individual files.\n")
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
		debugf("processFileWithPackageContext: files to process: %v", files)

		if len(files) == 0 {
			logf("No files with //go:generate xdrgen found in %s", inputPath)
			return
		}

		filesToProcess = files
		debugf("Found %d files with //go:generate xdrgen in %s", len(files), inputPath)
	} else {
		// Single file mode: process the specified file
		filesToProcess = []string{inputPath}
	}

	// Process each file
	for _, inputFile := range filesToProcess {
		processFile(inputFile)
	}
}

// processFile handles the processing of a single file using package-level context
func processFile(inputFile string) {
	// Always use package-level processing for consistency and cross-file union support
	dir := filepath.Dir(inputFile)
	processFileWithPackageContext(inputFile, dir)
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
								// Check for +xdr: directives
								if node.Doc != nil {
									// Parse +xdr: directives
									xdrDirectives := collectXDRDirectives(astFile, typeSpec.Pos())

									// Process +xdr:union directives
									if xdrDirectives.Union != "" && len(xdrDirectives.Cases) > 0 {
										// Create union config from directives
										unionConfig := &UnionConfig{
											ContainerType: xdrDirectives.Union, // Container is specified in union directive
											Cases:         make(map[string]string),
											VoidCases:     []string{},
										}

										// Map cases to this struct
										for _, caseName := range xdrDirectives.Cases {
											unionConfig.Cases[caseName] = typeSpec.Name.Name
											debugf("Found union directive: case %s -> struct %s for container %s", caseName, typeSpec.Name.Name, xdrDirectives.Union)
										}

										// Merge with existing config for this container type
										if existingConfig, exists := allUnionComments[unionConfig.ContainerType]; exists {
											// Merge cases
											for constantName, structName := range unionConfig.Cases {
												existingConfig.Cases[constantName] = structName
											}
											// Merge void cases
											existingConfig.VoidCases = append(existingConfig.VoidCases, unionConfig.VoidCases...)
										} else {
											// Store new config
											allUnionComments[unionConfig.ContainerType] = unionConfig
											debugf("Created union config for container %s", unionConfig.ContainerType)
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
									isKey, _ := parseKeyTag(xdrTag)
									if isKey {
										hasKey = true
										// Check if next field is []byte for union detection
										// This will be handled by the new parser logic
										hasUnion = true // Assume union follows key
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
										isKey, defaultType := parseKeyTag(xdrTag)
										isUnion := false // Will be auto-detected

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

	// Now process all files with XDR generation using complete package context
	for _, file := range files {
		processFileWithPackageUnionContext(file, allUnionComments, allTypeDefs, allConstants, allStructTypes, allContainerStructs)
	}
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
	types, _, _, err := parseFileWithPackageTypeDefs(inputFile, allTypeDefs, allConstants)
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

	// Type counts are shown in the generation message, no need for duplicate output

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
