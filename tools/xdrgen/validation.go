package main

import (
	"fmt"
	"go/ast"
)

// validateUnionConfiguration validates discriminated union configuration
func validateUnionConfiguration(types []TypeInfo, constants map[string]ConstantInfo, typeDefs map[string]ast.Node, file *ast.File) []ValidationError {
	var errors []ValidationError

	// Helper to resolve type aliases recursively
	var resolveToStruct func(typeName string) (*ast.StructType, string)
	resolveToStruct = func(typeName string) (*ast.StructType, string) {
		node, ok := typeDefs[typeName]
		if !ok {
			return nil, "notype"
		}

		// Extract the actual type from GenDecl if needed
		var actualType ast.Expr
		if genDecl, ok := node.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == typeName {
					actualType = typeSpec.Type
					break
				}
			}
		} else {
			// Fallback for cases where it's not a GenDecl
			actualType = node.(ast.Expr)
		}

		if actualType == nil {
			return nil, "notype"
		}

		switch t := actualType.(type) {
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
		keyFieldGenDecl, exists := typeDefs[keyField.Type]
		if !exists {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("key field type %s not found in current file(s). If this type is defined in another file, process the entire package instead of individual files", keyField.Type),
			})
			continue
		}

		// Extract the actual type from the GenDecl
		var keyFieldNode ast.Expr
		if genDecl, ok := keyFieldGenDecl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == keyField.Type {
					keyFieldNode = typeSpec.Type
					break
				}
			}
		} else {
			// Fallback for cases where it's not a GenDecl
			keyFieldNode = keyFieldGenDecl.(ast.Expr)
		}

		if keyFieldNode == nil {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("could not extract type information for key field type %s", keyField.Type),
			})
			continue
		}

		// Check that the key field type is a uint32 alias
		ident, ok := keyFieldNode.(*ast.Ident)
		if !ok {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("key field type %s must be a uint32 alias", keyField.Type),
			})
			continue
		}
		// This is a type alias, check if it's uint32
		if ident.Name != "uint32" {
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("key field type %s must be a uint32 alias, got %s", keyField.Type, ident.Name),
			})
			continue
		}

		// Use aggregated constants for package-level processing
		// In package-level processing, constants parameter contains all constants from all files
		typedConstants := constants
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

			// Handle void cases (nil or empty string)
			if structName == "nil" || structName == "" {
				continue // Void case - no validation needed
			}

			// Validate struct or alias-to-struct
			_, kind := resolveToStruct(structName)
			switch kind {
			case "notype":
				errors = append(errors, ValidationError{
					Location: fmt.Sprintf("union=%s,case=%s", containerType, constantName),
					Message:  fmt.Sprintf("union case payload type %s not found", structName),
				})
			case "struct":
				// Fields are auto-detected, no explicit tags required
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
				_, kind := resolveToStruct(unionField.DefaultType)
				switch kind {
				case "notype":
					errors = append(errors, ValidationError{
						Location: fmt.Sprintf("union field %s", unionField.Name),
						Message:  fmt.Sprintf("default type %s not found", unionField.DefaultType),
					})
				case "struct":
					// Fields are auto-detected, no explicit tags required
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

// validateDiscriminatedUnions validates that discriminated union structures are correctly formed
func validateDiscriminatedUnions(types []TypeInfo, constants map[string]ConstantInfo) error {
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
				return fmt.Errorf("discriminated union struct %s must have exactly one key field, found %d", typeInfo.Name, keyCount)
			}

			// Check if this is an all-void union (no union field needed)
			allCasesVoid := true
			if typeInfo.UnionConfig != nil {
				debugf("Checking void cases for %s: %+v", typeInfo.Name, typeInfo.UnionConfig.Cases)
				for caseName, structName := range typeInfo.UnionConfig.Cases {
					debugf("Case %s -> struct %s", caseName, structName)
					if structName != "" && structName != "nil" {
						allCasesVoid = false
						debugf("Found non-void case: %s -> %s", caseName, structName)
						break
					}
				}
			}
			debugf("AllCasesVoid for %s: %v", typeInfo.Name, allCasesVoid)

			if unionCount == 0 {
				return fmt.Errorf("discriminated union struct %s must have exactly one union field", typeInfo.Name)
			}

			// Validate ordered pair requirement: key must be immediately followed by union (only if union field exists)
			if unionCount > 0 && unionIndex != keyIndex+1 {
				return fmt.Errorf("discriminated union struct %s: key field must be immediately followed by union field", typeInfo.Name)
			}

			debugf("Validated discriminated union %s: %d key, %d union fields", typeInfo.Name, keyCount, unionCount)
		}
	}
	return nil
}

// Helper: detect loops in union payload chains
