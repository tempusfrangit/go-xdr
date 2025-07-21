package main

import (
	"log"
	"strings"
)

// StructDependencyGraph represents the dependency relationships between structs
type StructDependencyGraph struct {
	Nodes map[string]*StructNode
	Edges map[string][]string // struct name -> list of referenced struct names
}

// StructNode represents a struct in the dependency graph
type StructNode struct {
	Name         string
	CanHaveLoops bool // determined by cycle detection
}

// NewStructDependencyGraph creates a new dependency graph
func NewStructDependencyGraph() *StructDependencyGraph {
	return &StructDependencyGraph{
		Nodes: make(map[string]*StructNode),
		Edges: make(map[string][]string),
	}
}

// AddStruct adds a struct to the dependency graph
func (g *StructDependencyGraph) AddStruct(name string) {
	if _, exists := g.Nodes[name]; !exists {
		g.Nodes[name] = &StructNode{
			Name:         name,
			CanHaveLoops: false,
		}
		g.Edges[name] = []string{}
	}
}

// AddDependency adds a dependency relationship (from -> to)
func (g *StructDependencyGraph) AddDependency(from, to string) {
	g.AddStruct(from)
	g.AddStruct(to)

	// Avoid duplicate edges
	for _, existing := range g.Edges[from] {
		if existing == to {
			return
		}
	}

	g.Edges[from] = append(g.Edges[from], to)
	debugf("Added dependency: %s -> %s", from, to)
}

// DetectCycles performs cycle detection and returns all detected cycles
func (g *StructDependencyGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for structName := range g.Nodes {
		if !visited[structName] {
			if cycle := g.findCycleDFS(structName, visited, recursionStack, []string{}); cycle != nil {
				cycles = append(cycles, cycle)
				// Mark all structs in this cycle as potentially having loops
				for _, cycleName := range cycle {
					if node := g.Nodes[cycleName]; node != nil {
						node.CanHaveLoops = true
					}
				}
			}
		}
	}

	return cycles
}

// findCycleDFS performs DFS to find cycles, returns the cycle path if found
func (g *StructDependencyGraph) findCycleDFS(current string, visited, recursionStack map[string]bool, path []string) []string {
	// Mark current node as visited and add to recursion stack
	visited[current] = true
	recursionStack[current] = true
	path = append(path, current)

	// Check all neighbors of current node
	for _, neighbor := range g.Edges[current] {
		if cycle := g.processNeighbor(neighbor, visited, recursionStack, path); cycle != nil {
			return cycle
		}
	}

	// Backtrack: remove from recursion stack when done exploring this path
	recursionStack[current] = false
	return nil
}

// processNeighbor handles the logic for processing a single neighbor during DFS
func (g *StructDependencyGraph) processNeighbor(neighbor string, visited, recursionStack map[string]bool, path []string) []string {
	if !visited[neighbor] {
		// Unvisited neighbor: recurse deeper
		return g.findCycleDFS(neighbor, visited, recursionStack, path)
	}

	if recursionStack[neighbor] {
		// Back edge found: there's a cycle from neighbor back to current path
		return g.extractCycle(neighbor, path)
	}

	// Neighbor is visited but not in recursion stack: no cycle through this edge
	return nil
}

// extractCycle extracts the cycle path when a back edge is detected
func (g *StructDependencyGraph) extractCycle(backEdgeTarget string, path []string) []string {
	cycleStartIndex := g.findNodeInPath(backEdgeTarget, path)

	if cycleStartIndex < 0 {
		// This shouldn't happen in correct DFS - log error but don't crash
		debugf("ERROR: Back edge target %s not found in current path %v", backEdgeTarget, path)
		return nil
	}

	// Extract cycle: from where the back edge target appears to end of path, plus the target again
	cycleNodes := make([]string, 0, len(path)-cycleStartIndex+1)
	cycleNodes = append(cycleNodes, path[cycleStartIndex:]...)
	cycleNodes = append(cycleNodes, backEdgeTarget)

	return cycleNodes
}

// findNodeInPath searches for a node in the current DFS path
func (g *StructDependencyGraph) findNodeInPath(targetNode string, path []string) int {
	for i, node := range path {
		if node == targetNode {
			return i
		}
	}
	return -1
}

// AnalyzeTypeDependencies builds the dependency graph from parsed types
func AnalyzeTypeDependencies(types []TypeInfo) *StructDependencyGraph {
	graph := NewStructDependencyGraph()

	// Add all struct types to the graph
	for _, typeInfo := range types {
		graph.AddStruct(typeInfo.Name)
	}

	// Build dependency edges based on field types
	for _, typeInfo := range types {
		debugf("Analyzing dependencies for type: %s", typeInfo.Name)
		for _, field := range typeInfo.Fields {
			referencedType := extractStructTypeFromField(field)
			debugf("  Field %s (type: %s) -> referenced type: %s", field.Name, field.Type, referencedType)
			if referencedType != "" {
				if referencedType == "ANY_TYPE" {
					// This struct contains any/interface{} - mark as implicitly cyclable
					if node := graph.Nodes[typeInfo.Name]; node != nil {
						node.CanHaveLoops = true
						debugf("Marked %s as cyclable due to any/interface{} field: %s %s",
							typeInfo.Name, field.Name, field.Type)
					}
				} else {
					// Check if the referenced type exists in our type set
					if _, exists := graph.Nodes[referencedType]; exists {
						graph.AddDependency(typeInfo.Name, referencedType)
					}
				}
			}
		}
	}

	return graph
}

// extractStructTypeFromField extracts the struct type name from a field, if any
// Returns the struct name, or "ANY_TYPE" if the field can contain arbitrary types (implicitly cyclable)
func extractStructTypeFromField(field FieldInfo) string {
	fieldType := field.Type

	// Check for any/interface{} types that can implicitly cause cycles
	if containsAnyOrInterface(fieldType) {
		return "ANY_TYPE" // Special marker for implicitly cyclable types
	}

	// Handle pointer types (*StructName -> StructName)
	fieldType = strings.TrimPrefix(fieldType, "*")

	// Handle slice types ([]StructName -> StructName, []*StructName -> StructName)
	if strings.HasPrefix(fieldType, "[]") {
		fieldType = fieldType[2:]
		// Handle slice of pointers []*StructName -> StructName
		fieldType = strings.TrimPrefix(fieldType, "*")
	}

	// Handle array types ([N]StructName -> StructName)
	if strings.HasPrefix(fieldType, "[") && strings.Contains(fieldType, "]") {
		if closeBracket := strings.Index(fieldType, "]"); closeBracket >= 0 {
			fieldType = fieldType[closeBracket+1:]
		}
	}

	// Skip primitive types
	switch fieldType {
	case "uint32", "uint64", "int32", "int64", "string", "bool", "byte":
		return ""
	case "[]byte": // Special case for byte slices
		return ""
	}

	// Skip types with package prefixes (external types)
	if strings.Contains(fieldType, ".") {
		return ""
	}

	return fieldType
}

// containsAnyOrInterface checks if a type contains any, interface{}, or maps/slices of any
func containsAnyOrInterface(fieldType string) bool {
	// Direct any/interface{} types
	if fieldType == "any" || fieldType == "interface{}" {
		return true
	}

	// Slices of any: []any, []interface{}
	if fieldType == "[]any" || fieldType == "[]interface{}" {
		return true
	}

	// Maps with any values: map[K]any, map[K]interface{}
	if strings.HasPrefix(fieldType, "map[") {
		if strings.Contains(fieldType, "]any") || strings.Contains(fieldType, "]interface{}") {
			return true
		}
		// Maps with any keys: map[any]V, map[interface{}]V
		if strings.Contains(fieldType, "map[any]") || strings.Contains(fieldType, "map[interface{}]") {
			return true
		}
	}

	// Pointers to any: *any, *interface{}
	if fieldType == "*any" || fieldType == "*interface{}" {
		return true
	}

	// Arrays of any: [N]any, [N]interface{}
	if strings.HasPrefix(fieldType, "[") && strings.Contains(fieldType, "]") {
		if strings.Contains(fieldType, "]any") || strings.Contains(fieldType, "]interface{}") {
			return true
		}
	}

	return false
}

// UpdateTypeInfoWithCycles updates TypeInfo structs with cycle detection results
func UpdateTypeInfoWithCycles(types []TypeInfo, graph *StructDependencyGraph) {
	for i := range types {
		if node := graph.Nodes[types[i].Name]; node != nil {
			types[i].CanHaveLoops = node.CanHaveLoops
		}
	}
}

// PrintCycleWarnings prints warnings for detected cycles and implicitly cyclable types
func PrintCycleWarnings(cycles [][]string, graph *StructDependencyGraph) {
	// Print explicit cycle warnings
	for _, cycle := range cycles {
		if len(cycle) > 1 {
			// Format the cycle path nicely
			cyclePath := strings.Join(cycle, " -> ")

			// Warning 1: Always print when cycles detected
			log.Printf("WARNING: Potential circular reference detected in type dependency graph:")
			log.Printf("  %s", cyclePath)

			// Warning 2: Only print if loop detection is disabled
			if disableLoopDetection {
				log.Printf("")
				log.Printf("WARNING: Loop protection disabled by --disable-loop-detection flag.")
				log.Printf("This will allow infinite recursion if circular data structures exist at runtime,")
				log.Printf("resulting in stack overflow and program crash.")
				log.Printf("")
				log.Printf("It is recommended to either:")
				log.Printf("  1. Restructure types to eliminate circular references, or")
				log.Printf("  2. Remove --disable-loop-detection to enable runtime protection")
				log.Printf("")
				log.Printf("Use --disable-loop-detection only when certain that runtime data contains no cycles.")
			}
			log.Printf("")
		}
	}

	// Print warnings for types with any/interface{} fields (implicitly cyclable)
	var implicitlyCyclable []string
	for name, node := range graph.Nodes {
		// Check if this type is marked as cyclable but not part of an explicit cycle
		if node.CanHaveLoops {
			inExplicitCycle := false
			for _, cycle := range cycles {
				for _, cycleName := range cycle {
					if cycleName == name {
						inExplicitCycle = true
						break
					}
				}
				if inExplicitCycle {
					break
				}
			}

			if !inExplicitCycle {
				implicitlyCyclable = append(implicitlyCyclable, name)
			}
		}
	}

	if len(implicitlyCyclable) > 0 {
		log.Printf("WARNING: Types with any/interface{} fields are implicitly cyclable:")
		for _, typeName := range implicitlyCyclable {
			log.Printf("  %s (contains any/interface{} fields)", typeName)
		}

		if disableLoopDetection {
			log.Printf("")
			log.Printf("WARNING: Loop protection disabled by --disable-loop-detection flag.")
			log.Printf("Types with any/interface{} can hold circular references at runtime,")
			log.Printf("potentially causing infinite recursion and stack overflow.")
		}
		log.Printf("")
	}
}

// hasImplicitlyCyclableTypes checks if any types are marked as implicitly cyclable
func hasImplicitlyCyclableTypes(graph *StructDependencyGraph) bool {
	for _, node := range graph.Nodes {
		if node.CanHaveLoops {
			return true
		}
	}
	return false
}
