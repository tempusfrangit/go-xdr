package main

import "fmt"

// FieldInfo represents a struct field with XDR encoding information
type FieldInfo struct {
	Name         string
	Type         string // Original Go type from source (e.g., "SessionID")
	ResolvedType string // Resolved underlying type (e.g., "[]byte")
	XDRType      string
	Tag          string
	IsKey        bool   // true if this field is a discriminated union key
	IsUnion      bool   // true if this field is a discriminated union payload
	DefaultType  string // default type from union tag (empty, "nil", or struct name)
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