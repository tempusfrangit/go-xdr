package main

import (
	"fmt"
	"go/ast"
	"strings"
)

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
			errors = append(errors, ValidationError{
				Location: fmt.Sprintf("union=%s,key=%s", containerType, keyField.Name),
				Message:  fmt.Sprintf("key field type %s not found in current file(s). If this type is defined in another file, process the entire package instead of individual files", keyField.Type),
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
