//go:build ignore

package codegen_test

import (
	"testing"
)

// TestUnionFieldNamesCorrectness tests that ToUnion methods use the correct field names
// from the union struct definition rather than hardcoded 'Type' and 'Payload' values.

//go:generate ../bin/xdrgen $GOFILE
type TestStatus uint32

const (
	TestStatusSuccess TestStatus = 0
	TestStatusError   TestStatus = 1
)

// +xdr:union,key=Status
type TestResult struct {
	Status TestStatus // discriminant
	Data   []byte     // auto-detected as union payload
}

// +xdr:payload,union=TestResult,discriminant=TestStatusSuccess
type TestSuccessPayload struct {
	Message string // auto-detected as string
}

type TestMsgType uint32

const (
	TestMsgTypeText TestMsgType = 1
	TestMsgTypeVoid TestMsgType = 2
)

// +xdr:union,key=Type
type TestMessage struct {
	Type    TestMsgType // discriminant
	Payload []byte      // auto-detected as union payload
}

// +xdr:payload,union=TestMessage,discriminant=TestMsgTypeText
type TestTextPayload struct {
	Content string // auto-detected as string
}

type TestOpType uint32

const (
	TestOpRead  TestOpType = 1
	TestOpWrite TestOpType = 2
)

// +xdr:union,key=OpType
type TestOperation struct {
	OpType TestOpType // discriminant
	Data   []byte     // auto-detected as union payload
}

// +xdr:payload,union=TestOperation,discriminant=TestOpRead
type TestReadPayload struct {
	Size uint32
}

func TestUnionFieldNamesCorrectness(t *testing.T) {
	// Test TestResult union
	t.Run("TestResult with Status/Data fields", func(t *testing.T) {
		payload := &TestSuccessPayload{Message: "success"}
		union, err := payload.ToUnion()
		if err != nil {
			t.Fatalf("ToUnion() failed: %v", err)
		}
		
		// Verify the fields exist by checking we can access them
		if union.Status != TestStatusSuccess {
			t.Errorf("Expected Status to be TestStatusSuccess, got %v", union.Status)
		}
		if len(union.Data) == 0 {
			t.Errorf("Expected Data field to be populated")
		}
	})

	// Test TestMessage union
	t.Run("TestMessage with Type/Payload fields", func(t *testing.T) {
		payload := &TestTextPayload{Content: "hello"}
		union, err := payload.ToUnion()
		if err != nil {
			t.Fatalf("ToUnion() failed: %v", err)
		}
		
		// Verify the fields exist by checking we can access them
		if union.Type != TestMsgTypeText {
			t.Errorf("Expected Type to be TestMsgTypeText, got %v", union.Type)
		}
		if len(union.Payload) == 0 {
			t.Errorf("Expected Payload field to be populated")
		}
	})

	// Test TestOperation union
	t.Run("TestOperation with OpType/Data fields", func(t *testing.T) {
		payload := &TestReadPayload{Size: 1024}
		union, err := payload.ToUnion()
		if err != nil {
			t.Fatalf("ToUnion() failed: %v", err)
		}
		
		// Verify the fields exist by checking we can access them
		if union.OpType != TestOpRead {
			t.Errorf("Expected OpType to be TestOpRead, got %v", union.OpType)
		}
		if len(union.Data) == 0 {
			t.Errorf("Expected Data field to be populated")
		}
	})
}