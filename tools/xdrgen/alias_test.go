package main

import (
	"strings"
	"testing"
)

func TestArrayAliasResolution(t *testing.T) {
	// Test that slice aliases don't get [:] conversion but fixed array aliases do
	tests := []struct {
		name         string
		field        FieldInfo
		expectedCode string
		description  string
	}{
		{
			name: "slice alias should not use [:]",
			field: FieldInfo{
				Name:         "Session",
				Type:         "SessionID",
				ResolvedType: "[]byte",
				XDRType:      "bytes",
			},
			expectedCode: "v.Session",
			description:  "SessionID []byte should generate v.Session, not v.Session[:]",
		},
		{
			name: "fixed array alias should use [:]",
			field: FieldInfo{
				Name:         "Hash",
				Type:         "Hash",
				ResolvedType: "[16]byte",
				XDRType:      "bytes",
			},
			expectedCode: "v.Hash[:]",
			description:  "Hash [16]byte should generate v.Hash[:]",
		},
		{
			name: "direct []byte should not use [:]",
			field: FieldInfo{
				Name:         "Data",
				Type:         "[]byte",
				ResolvedType: "[]byte",
				XDRType:      "bytes",
			},
			expectedCode: "v.Data",
			description:  "[]byte should generate v.Data, not v.Data[:]",
		},
		{
			name: "array of primitive aliases should use casting",
			field: FieldInfo{
				Name:         "StatusList",
				Type:         "[]StatusCode",
				ResolvedType: "[]uint32",
				XDRType:      "uint32",
			},
			expectedCode: "uint32(elem)",
			description:  "[]StatusCode should generate uint32(elem) for encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg, err := NewCodeGenerator([]string{})
			if err != nil {
				t.Fatalf("NewCodeGenerator failed: %v", err)
			}

			result, err := cg.generateBasicEncodeCode(tt.field)
			if err != nil {
				t.Fatalf("generateBasicEncodeCode failed: %v", err)
			}

			if !strings.Contains(result, tt.expectedCode) {
				t.Errorf("%s: expected code to contain %q, got:\n%s", tt.description, tt.expectedCode, result)
			}
		})
	}
}