package main

import (
	"testing"
)

func TestStructDependencyGraph(t *testing.T) {
	graph := NewStructDependencyGraph()
	
	// Add some structs
	graph.AddStruct("A")
	graph.AddStruct("B")
	graph.AddStruct("C")
	
	// Add dependencies
	graph.AddDependency("A", "B")
	graph.AddDependency("B", "C")
	
	// Test nodes were created
	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}
	
	// Test edges were created
	if len(graph.Edges["A"]) != 1 || graph.Edges["A"][0] != "B" {
		t.Errorf("Expected A -> B dependency")
	}
}

func TestCycleDetection(t *testing.T) {
	tests := []struct {
		name          string
		dependencies  map[string][]string
		expectedCycles int
		expectedCyclable []string
	}{
		{
			name: "no_cycles",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {},
			},
			expectedCycles: 0,
			expectedCyclable: []string{},
		},
		{
			name: "simple_cycle",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"A"},
			},
			expectedCycles: 1,
			expectedCyclable: []string{"A", "B"},
		},
		{
			name: "complex_cycle",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {"A"},
			},
			expectedCycles: 1,
			expectedCyclable: []string{"A", "B", "C"},
		},
		{
			name: "multiple_cycles",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"A"},
				"C": {"D"},
				"D": {"C"},
			},
			expectedCycles: 2,
			expectedCyclable: []string{"A", "B", "C", "D"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewStructDependencyGraph()
			
			// Add all structs and dependencies
			for from, toList := range tt.dependencies {
				graph.AddStruct(from)
				for _, to := range toList {
					graph.AddDependency(from, to)
				}
			}
			
			// Detect cycles
			cycles := graph.DetectCycles()
			
			// Check cycle count
			if len(cycles) != tt.expectedCycles {
				t.Errorf("Expected %d cycles, got %d", tt.expectedCycles, len(cycles))
			}
			
			// Check that expected types are marked as cyclable
			cyclableCount := 0
			for _, node := range graph.Nodes {
				if node.CanHaveLoops {
					cyclableCount++
				}
			}
			
			if cyclableCount != len(tt.expectedCyclable) {
				t.Errorf("Expected %d cyclable types, got %d", len(tt.expectedCyclable), cyclableCount)
			}
		})
	}
}

func TestContainsAnyOrInterface(t *testing.T) {
	tests := []struct {
		name     string
		fieldType string
		expected bool
	}{
		{"any", "any", true},
		{"interface{}", "interface{}", true},
		{"slice_of_any", "[]any", true},
		{"slice_of_interface", "[]interface{}", true},
		{"map_with_any_value", "map[string]any", true},
		{"map_with_interface_value", "map[string]interface{}", true},
		{"map_with_any_key", "map[any]string", true},
		{"map_with_interface_key", "map[interface{}]string", true},
		{"pointer_to_any", "*any", true},
		{"pointer_to_interface", "*interface{}", true},
		{"array_of_any", "[10]any", true},
		{"array_of_interface", "[10]interface{}", true},
		
		// Negative cases
		{"string", "string", false},
		{"slice_of_string", "[]string", false},
		{"map_string_string", "map[string]string", false},
		{"pointer_to_string", "*string", false},
		{"array_of_string", "[10]string", false},
		{"custom_type", "MyType", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAnyOrInterface(tt.fieldType)
			if result != tt.expected {
				t.Errorf("containsAnyOrInterface(%q) = %v, expected %v", tt.fieldType, result, tt.expected)
			}
		})
	}
}

func TestAnalyzeTypeDependencies(t *testing.T) {
	types := []TypeInfo{
		{
			Name: "Person",
			Fields: []FieldInfo{
				{Name: "Name", Type: "string"},
				{Name: "Friend", Type: "*Person"}, // Self-reference
			},
		},
		{
			Name: "Company",
			Fields: []FieldInfo{
				{Name: "Name", Type: "string"},
				{Name: "CEO", Type: "*Person"},
			},
		},
		{
			Name: "FlexibleStruct",
			Fields: []FieldInfo{
				{Name: "Data", Type: "any"}, // Should be marked as implicitly cyclable
			},
		},
	}
	
	graph := AnalyzeTypeDependencies(types)
	
	// Check that all types were added
	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}
	
	// Check dependencies
	personDeps := graph.Edges["Person"]
	if len(personDeps) != 1 || personDeps[0] != "Person" {
		t.Errorf("Expected Person -> Person self-reference, got %v", personDeps)
	}
	
	companyDeps := graph.Edges["Company"]
	if len(companyDeps) != 1 || companyDeps[0] != "Person" {
		t.Errorf("Expected Company -> Person dependency, got %v", companyDeps)
	}
	
	// Check that FlexibleStruct is marked as implicitly cyclable due to 'any' field
	flexibleNode := graph.Nodes["FlexibleStruct"]
	if !flexibleNode.CanHaveLoops {
		t.Errorf("Expected FlexibleStruct to be marked as cyclable due to 'any' field")
	}
	
	// Run cycle detection
	cycles := graph.DetectCycles()
	
	// Should detect Person -> Person self-cycle
	if len(cycles) != 1 {
		t.Errorf("Expected 1 cycle (Person self-reference), got %d", len(cycles))
	}
	
	// Person should be marked as cyclable due to explicit cycle
	personNode := graph.Nodes["Person"]
	if !personNode.CanHaveLoops {
		t.Errorf("Expected Person to be marked as cyclable due to self-reference cycle")
	}
}