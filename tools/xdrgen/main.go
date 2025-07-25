// XDR Code Generator - Ultra-Minimal Edition
//
// Generates XDR encoding/decoding methods with maximum auto-detection and minimal tagging.
// Directive-based system: `// +xdr:generate`, `// +xdr:union,key=Field[,default=X]`, `// +xdr:payload,union=Name,discriminant=Const`. Only 1 tag type: `xdr:"-"`. Everything else auto-detected.
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
//	// +xdr:union,key=OpCode
//	type Operation struct {
//	    OpCode OpCode // union discriminant
//	    Result []byte // auto-detected union payload
//	}
//
// See help.txt for complete documentation.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var silent bool
var debug = false
var disableLoopDetection = false

// logf logs a message unless in silent mode
func logf(msg string, args ...any) {
	if !silent {
		log.Printf(msg, args...)
	}
}

// debugf logs a message only in debug mode
func debugf(msg string, args ...any) {
	if debug {
		log.Printf("DEBUG: "+msg, args...)
	}
}

// PayloadMapping tracks payload-to-discriminant relationships
type PayloadMapping struct {
	PayloadType  string
	Discriminant string
}

// parsePackage parses all files in a package to build complete union configurations

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
	flag.BoolVar(&disableLoopDetection, "disable-loop-detection", false, "disable runtime loop detection even for types with potential cycles")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "xdrgen - XDR Code Generator\n\n")
		fmt.Fprintf(os.Stderr, "Generates Encode and Decode methods for Go structs using directive-based configuration.\n")
		fmt.Fprintf(os.Stderr, "Features ultra-minimal tagging with automatic type detection and discriminated unions.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  xdrgen [flags] <package-dir>     # Process package (files with //go:generate)\n")
		fmt.Fprintf(os.Stderr, "  xdrgen [flags] <file.go>         # Process single file\n\n")
		fmt.Fprintf(os.Stderr, "The generator processes files with XDR generation directives:\n")
		fmt.Fprintf(os.Stderr, "  //go:generate xdrgen $GOFILE              (process current file)\n")
		fmt.Fprintf(os.Stderr, "  //go:generate ../../bin/xdrgen types.go  (with relative path)\n\n")
		fmt.Fprintf(os.Stderr, "XDR Generation Directives:\n")
		fmt.Fprintf(os.Stderr, "  // +xdr:generate                              - Generate for standalone struct\n")
		fmt.Fprintf(os.Stderr, "  // +xdr:union,key=FieldName,default=TypeName  - Union container\n")
		fmt.Fprintf(os.Stderr, "  // +xdr:payload,union=UnionName,discriminant=Const - Union payload\n\n")
		fmt.Fprintf(os.Stderr, "Supported Go Types (auto-detected):\n")
		fmt.Fprintf(os.Stderr, "  uint32, uint64, int32, int64  - Integer types\n")
		fmt.Fprintf(os.Stderr, "  string                        - Variable-length string\n")
		fmt.Fprintf(os.Stderr, "  []byte, [N]byte               - Byte arrays (variable/fixed)\n")
		fmt.Fprintf(os.Stderr, "  bool                          - Boolean (encoded as uint32)\n")
		fmt.Fprintf(os.Stderr, "  struct types                  - Nested structs (auto-detected)\n")
		fmt.Fprintf(os.Stderr, "  []Type                        - Variable-length arrays\n")
		fmt.Fprintf(os.Stderr, "  type aliases                  - Automatically resolved\n\n")
		fmt.Fprintf(os.Stderr, "Optional struct tags:\n")
		fmt.Fprintf(os.Stderr, "  `xdr:\"-\"`          - exclude field from encoding/decoding\n\n")
		fmt.Fprintf(os.Stderr, "Discriminated Unions:\n")
		fmt.Fprintf(os.Stderr, "  Union containers use: // +xdr:union,key=FieldName,default=DefaultType\n")
		fmt.Fprintf(os.Stderr, "  Payload types use: // +xdr:payload,union=ContainerName,discriminant=ConstName\n")
		fmt.Fprintf(os.Stderr, "  Void cases are auto-detected when no payload exists for a discriminant\n")
		fmt.Fprintf(os.Stderr, "  Example:\n")
		fmt.Fprintf(os.Stderr, "    // +xdr:union,key=Type,default=MessageTypeVoid\n")
		fmt.Fprintf(os.Stderr, "    type NetworkMessage struct { Type MessageType; Payload []byte }\n")
		fmt.Fprintf(os.Stderr, "    // +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText\n")
		fmt.Fprintf(os.Stderr, "    type TextPayload struct { Content string }\n\n")
		fmt.Fprintf(os.Stderr, "Type Aliases:\n")
		fmt.Fprintf(os.Stderr, "  All type aliases are automatically detected and resolved\n")
		fmt.Fprintf(os.Stderr, "  No explicit tags needed - the generator infers XDR types from Go types\n")
		fmt.Fprintf(os.Stderr, "  Example: type UserID string  // automatically handled as string\n\n")
		fmt.Fprintf(os.Stderr, "Output:\n")
		fmt.Fprintf(os.Stderr, "  Creates <input>_xdr.go with generated Encode/Decode methods\n")
		fmt.Fprintf(os.Stderr, "  Includes compile-time assertions that types implement xdr.Codec\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  xdrgen types.go         # Generate for single file\n")
		fmt.Fprintf(os.Stderr, "  xdrgen ./               # Process package directory\n")
		fmt.Fprintf(os.Stderr, "  xdrgen -s types.go      # Generate silently (no output except errors)\n")
		fmt.Fprintf(os.Stderr, "  go generate             # Use with //go:generate directives\n\n")
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

	// Set debug from environment variable if present
	if os.Getenv("XDRGEN_DEBUG") != "" {
		debug = true
	}

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
	debugf("processFileWithPackageContext: called with inputFile=%s, packageDir=%s", inputFile, packageDir)

	// Find ALL Go files in the package for complete type resolution
	allFiles, err := findAllGoFiles(packageDir)
	if err != nil {
		log.Fatal("Error finding all Go files:", err)
	}
	debugf("findAllGoFiles found %d files", len(allFiles))

	// Also discover files with generation directives for processing
	generatableFiles, err := discoverGoFiles(packageDir)
	if err != nil {
		log.Fatal("Error discovering generatable Go files:", err)
	}
	debugf("discoverGoFiles found %d generatable files", len(generatableFiles))

	// Parse all files to gather type information and build union configurations
	allUnionConfigs := make(map[string]*UnionConfig)
	allTypeDefs := make(map[string]ast.Node)
	allConstants := make(map[string]ConstantInfo)
	allStructTypes := make(map[string]bool)
	allTypeAliases := make(map[string]string)            // Collect type aliases from all files
	payloadMappings := make(map[string][]PayloadMapping) // unionType -> []PayloadMapping

	// Parse ALL files in the package to gather complete type information
	debugf("Package-level collection: allFiles contains %d files", len(allFiles))
	for i, file := range allFiles {
		debugf("Package-level collection: processing file %d/%d: %s", i+1, len(allFiles), file)
		// Parse each file to get type definitions and constants
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			log.Fatalf("Package-level collection: failed to parse file %s: %v", file, err)
		}
		debugf("Package-level collection: successfully parsed file %s", file)

		// Collect type definitions, constants, and type aliases
		ast.Inspect(astFile, func(n ast.Node) bool {
			if genDecl, ok := n.(*ast.GenDecl); ok {
				// Store the full GenDecl which contains the directive comments
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						allTypeDefs[typeSpec.Name.Name] = genDecl
						if _, ok := typeSpec.Type.(*ast.StructType); ok {
							allStructTypes[typeSpec.Name.Name] = true
						}

						// Collect type aliases
						if typeSpec.Assign != token.NoPos || typeSpec.Assign == 0 {
							if ident, ok := typeSpec.Type.(*ast.Ident); ok {
								allTypeAliases[typeSpec.Name.Name] = ident.Name
							}
						}
					}
				}
			}
			return true
		})

		// Collect constants
		fileConstants := collectConstants(astFile)
		for name, constantInfo := range fileConstants {
			allConstants[name] = constantInfo
		}

		// Parse file to collect payload directives (using collection mode to skip strict validation)
		types, _, _, err := parseFileWithPackageTypeDefsForCollection(file, allTypeDefs, allConstants, allTypeAliases)
		if err != nil {
			debugf("Package-level collection: failed to parse payload directives from %s: %v", file, err)
			continue // Skip files that can't be parsed for payload directives
		}

		// Collect payload mappings from parsed types
		for _, typeInfo := range types {
			if typeInfo.IsPayload && typeInfo.PayloadConfig != nil {
				mapping := PayloadMapping{
					PayloadType:  typeInfo.Name,
					Discriminant: typeInfo.PayloadConfig.Discriminant,
				}
				unionType := typeInfo.PayloadConfig.UnionType
				payloadMappings[unionType] = append(payloadMappings[unionType], mapping)
				debugf("Found payload mapping: %s -> %s (discriminant: %s)",
					typeInfo.Name, unionType, typeInfo.PayloadConfig.Discriminant)
			}
		}
	}

	// Build union configurations from payload mappings
	for unionType, mappings := range payloadMappings {
		unionConfig := &UnionConfig{
			ContainerType: unionType,
			Cases:         make(map[string]string),
			VoidCases:     []string{},
		}

		// Map discriminants to payload types
		for _, mapping := range mappings {
			unionConfig.Cases[mapping.Discriminant] = mapping.PayloadType
		}

		allUnionConfigs[unionType] = unionConfig
		debugf("Created union config for %s: %+v", unionType, unionConfig)
	}

	// Compute void cases for each union by finding constants without payload mappings
	// We need to find the discriminant type for each union by looking at the union container struct
	for unionType, unionConfig := range allUnionConfigs {
		// Find the discriminant type by looking at the union container struct
		if unionTypeDef, exists := allTypeDefs[unionType]; exists {
			if structType, ok := unionTypeDef.(*ast.StructType); ok {
				// Find the key field to determine discriminant type
				for _, field := range structType.Fields.List {
					if field.Tag != nil {
						tag := strings.Trim(field.Tag.Value, "`")
						if strings.Contains(tag, `xdr:"key"`) {
							// Get the field type
							if ident, ok := field.Type.(*ast.Ident); ok {
								discriminantType := ident.Name
								unionConfig.DiscriminantType = discriminantType
								debugf("Found discriminant type %s for union %s", discriminantType, unionType)

								// Now find constants of this discriminant type
								for constantName, constantInfo := range allConstants {
									if constantInfo.Type == discriminantType {
										if _, hasPayload := unionConfig.Cases[constantName]; !hasPayload {
											unionConfig.VoidCases = append(unionConfig.VoidCases, constantName)
											debugf("Added void case %s for union %s", constantName, unionType)
										}
									}
								}
								break
							}
						}
					}
				}
			}
		}
	}

	if debug {
		var keys []string
		for k := range allTypeDefs {
			keys = append(keys, k)
		}
		debugf("processFileWithPackageContext: allTypeDefs keys: %v", keys)
	}

	// Now process only generatable files for XDR code generation using complete package context
	debugf("Processing %d generatable files for code generation", len(generatableFiles))
	for _, file := range generatableFiles {
		debugf("Processing generatable file: %s", file)
		processFileWithPackageUnionContext(file, allUnionConfigs, allTypeDefs, allConstants, allStructTypes, allTypeAliases)
	}
}

// processFileWithPackageUnionContext processes a file with complete package-level union configuration context
func processFileWithPackageUnionContext(inputFile string, allUnionConfigs map[string]*UnionConfig, allTypeDefs map[string]ast.Node, allConstants map[string]ConstantInfo, allStructTypes map[string]bool, allTypeAliases map[string]string) {
	// Generate output file name, handling test files specially
	var outputFile string
	if strings.HasSuffix(inputFile, "_test.go") {
		outputFile = strings.TrimSuffix(inputFile, "_test.go") + "_xdr_test.go"
	} else {
		outputFile = strings.TrimSuffix(inputFile, ".go") + "_xdr.go"
	}

	debugf("Processing input file: %s", inputFile)
	types, _, _, err := parseFileWithPackageTypeDefs(inputFile, allTypeDefs, allConstants, allTypeAliases)
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
			if unionConfig, exists := allUnionConfigs[types[i].Name]; exists {
				types[i].UnionConfig = unionConfig
				debugf("Updated container struct %s with package-level union config", types[i].Name)
			} else {
				debugf("No union config found for container %s", types[i].Name)
			}
		}
	}

	// Package-level validation is done before individual file processing

	if len(types) == 0 {
		logf("No types requiring XDR generation found in %s", inputFile)
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
	debugf("Final allTypeAliases (%d entries): %v", len(allTypeAliases), allTypeAliases)

	// Analyze type dependencies and detect potential cycles
	graph := AnalyzeTypeDependencies(types)
	cycles := graph.DetectCycles()

	// Update types with cycle detection results
	UpdateTypeInfoWithCycles(types, graph)

	// Print warnings if cycles were detected or types are implicitly cyclable
	if len(cycles) > 0 || hasImplicitlyCyclableTypes(graph) {
		PrintCycleWarnings(cycles, graph)
	}

	// Create code generator
	codeGen, err := NewCodeGenerator(structTypeNames, allTypeAliases)
	if err != nil {
		log.Fatal("Error creating code generator:", err)
	}

	externalImports := collectExternalImports(types, file)

	// Generate output
	var output strings.Builder

	// Generate file header with consistent path for determinism
	// Use just the filename to avoid path variations based on execution directory
	relativeInputFile := filepath.Base(inputFile)
	header, err := codeGen.GenerateFileHeader(relativeInputFile, packageName, externalImports, buildTags, len(types))
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

		// Generate payload-specific methods if this is a payload type
		if typeInfo.IsPayload {
			// Generate ToUnion method
			toUnionMethod, err := codeGen.GeneratePayloadToUnion(typeInfo, allTypeDefs)
			if err != nil {
				log.Fatal("Error generating ToUnion method:", err)
			}
			output.WriteString(toUnionMethod)
			output.WriteString("\n")

			// Generate EncodeToUnion method
			encodeToUnionMethod, err := codeGen.GeneratePayloadEncodeToUnion(typeInfo)
			if err != nil {
				log.Fatal("Error generating EncodeToUnion method:", err)
			}
			output.WriteString(encodeToUnionMethod)
			output.WriteString("\n")
		}

		// Generate compile-time assertion
		assertion, err := codeGen.GenerateAssertion(typeInfo.Name)
		if err != nil {
			log.Fatal("Error generating assertion:", err)
		}
		output.WriteString(assertion)
		output.WriteString("\n")
	}

	// Write to output file
	if err := os.WriteFile(outputFile, []byte(output.String()), 0600); err != nil {
		log.Fatal("Error writing output file:", err)
	}

	// Format the generated code
	if err := formatGeneratedCode(outputFile); err != nil {
		log.Printf("Warning: failed to format generated code: %v", err)
	}

	if !silent {
		logf("Generated XDR methods for %d types in %s", len(types), outputFile)
	}
}

// formatGeneratedCode formats the generated Go code using go/format
func formatGeneratedCode(filename string) error {
	// Read the generated file
	src, err := os.ReadFile(filename) // #nosec G304 -- Code generator needs to read generated files for formatting
	if err != nil {
		return fmt.Errorf("failed to read generated file: %w", err)
	}

	// Format the code
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write back the formatted code
	if err := os.WriteFile(filename, formatted, 0600); err != nil {
		return fmt.Errorf("failed to write formatted code: %w", err)
	}

	return nil
}
