//go:build ignore

package main

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"

	"github.com/tempusfrangit/go-xdr"
)

// Node represents a node in a potentially circular data structure
// +xdr:generate
type Node struct {
	ID       uint32
	Value    string
	Children []*Node // Can create cycles
}

// FlexibleData contains any type that could implicitly create cycles
// +xdr:generate
type FlexibleData struct {
	ID   uint32
	Data any // Implicitly cyclable
}

// SimpleStruct has no potential for cycles (should get fast path)
// +xdr:generate
type SimpleStruct struct {
	ID    uint32
	Name  string
	Count int64
}

//go:generate ../../bin/xdrgen cycle_test.go

func TestCycleDetection(t *testing.T) {
	tests := []struct {
		name          string
		setupData     func() interface{ Encode(enc *xdr.Encoder) error }
		expectError   bool
		errorContains string
	}{
		{
			name: "simple_struct_no_cycles",
			setupData: func() interface{ Encode(enc *xdr.Encoder) error } {
				return &SimpleStruct{ID: 1, Name: "test", Count: 42}
			},
			expectError: false,
		},
		{
			name: "node_no_cycles",
			setupData: func() interface{ Encode(enc *xdr.Encoder) error } {
				return &Node{
					ID:    1,
					Value: "root",
					Children: []*Node{
						{ID: 2, Value: "child1", Children: nil},
						{ID: 3, Value: "child2", Children: nil},
					},
				}
			},
			expectError: false,
		},
		{
			name: "node_with_cycle",
			setupData: func() interface{ Encode(enc *xdr.Encoder) error } {
				root := &Node{ID: 1, Value: "root"}
				child := &Node{ID: 2, Value: "child"}

				// Create cycle: root -> child -> root
				root.Children = []*Node{child}
				child.Children = []*Node{root}

				return root
			},
			expectError:   true,
			errorContains: "encoding loop detected",
		},
		{
			name: "flexible_data_no_cycles",
			setupData: func() interface{ Encode(enc *xdr.Encoder) error } {
				return &FlexibleData{ID: 1, Data: "simple string"}
			},
			expectError: false,
		},
		{
			name: "flexible_data_with_cycle",
			setupData: func() interface{ Encode(enc *xdr.Encoder) error } {
				flex := &FlexibleData{ID: 1}
				flex.Data = flex // Self-reference through any field
				return flex
			},
			expectError:   true,
			errorContains: "encoding loop detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create encoder with sufficient buffer
			enc := xdr.NewEncoder(make([]byte, 4096))

			// Setup test data
			testData := tt.setupData()

			// Attempt encoding
			err := testData.Encode(enc)

			// Check expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestCycleDetectionCleanup(t *testing.T) {
	// Test that the encoding set is properly cleaned up after successful encoding
	node1 := &Node{ID: 1, Value: "node1"}
	node2 := &Node{ID: 2, Value: "node2"}
	node1.Children = []*Node{node2}

	enc := xdr.NewEncoder(make([]byte, 1024))

	// First encoding should succeed
	err := node1.Encode(enc)
	if err != nil {
		t.Errorf("First encoding failed: %v", err)
	}

	// Reset encoder position
	enc.Reset(make([]byte, 1024))

	// Second encoding should also succeed (proves cleanup works)
	err = node1.Encode(enc)
	if err != nil {
		t.Errorf("Second encoding failed: %v", err)
	}
}

func BenchmarkEncodingWithoutLoopDetection(b *testing.B) {
	// SimpleStruct should not have loop detection (fast path)
	simple := &SimpleStruct{ID: 1, Name: "benchmark", Count: 12345}
	enc := xdr.NewEncoder(make([]byte, 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Reset(make([]byte, 1024))
		err := simple.Encode(enc)
		if err != nil {
			b.Fatalf("Encoding failed: %v", err)
		}
	}
}

func BenchmarkEncodingWithLoopDetection(b *testing.B) {
	// Node should have loop detection (due to potential cycles)
	node := &Node{
		ID:    1,
		Value: "benchmark",
		Children: []*Node{
			{ID: 2, Value: "child", Children: nil},
		},
	}
	enc := xdr.NewEncoder(make([]byte, 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Reset(make([]byte, 1024))
		err := node.Encode(enc)
		if err != nil {
			b.Fatalf("Encoding failed: %v", err)
		}
	}
}

func BenchmarkEncodingFlexibleData(b *testing.B) {
	// FlexibleData should have loop detection (due to any field)
	flex := &FlexibleData{ID: 1, Data: "benchmark string"}
	enc := xdr.NewEncoder(make([]byte, 1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Reset(make([]byte, 1024))
		err := flex.Encode(enc)
		if err != nil {
			b.Fatalf("Encoding failed: %v", err)
		}
	}
}
