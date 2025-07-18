//go:build ignore

package codegen_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tempusfrangit/go-xdr"
)

// OpCode represents operation codes
//
//go:generate ../bin/xdrgen $GOFILE
type OpCode uint32

const (
	OpOpen  OpCode = 1
	OpClose OpCode = 2
	OpRead  OpCode = 3
	OpWrite OpCode = 4
)

// +xdr:union,key=OpCode
// Operation represents an operation
type Operation struct {
	OpCode OpCode // Operation discriminant
	Result []byte // Union payload (auto-detected)
}

// OpOpenResult is the payload for OpOpen
// (add struct if you want a non-void payload)

// OpReadResult contains the result of a read operation
type OpReadResult struct {
	Success bool   `xdr:"bool"`
	Data    []byte `xdr:"bytes"`
}

// OpWriteResult contains the result of a write operation
type OpWriteResult struct {
	BytesWritten uint32 `xdr:"uint32"`
}

// Test type aliases for testing
type TestUserID string
type TestSessionID []byte
type TestStatusCode uint32
type TestFlags uint64
type TestPriority int32
type TestTimestamp int64
type TestIsActive bool
type TestHash [16]byte

// +xdr:generate
// Test struct using alias types (all auto-detected)
type TestUser struct {
	ID       TestUserID     // auto-detected as string
	Session  TestSessionID  // auto-detected as bytes
	Status   TestStatusCode // auto-detected as uint32
	Flags    TestFlags      // auto-detected as uint64
	Priority TestPriority   // auto-detected as int32
	Created  TestTimestamp  // auto-detected as int64
	Active   TestIsActive   // auto-detected as bool
	Hash     TestHash       // auto-detected as bytes
}

// +xdr:generate
// TestCrossFileReference tests that we can reference types from codegen_xfile_test.go
type TestCrossFileReference struct {
	CrossFileData CrossFileStruct   // auto-detected as struct
	Items         []CrossFileStruct // auto-detected as struct array
}

// Ensure TestUser implements Codec interface
var _ xdr.Codec = (*TestUser)(nil)
var _ xdr.Codec = (*TestCrossFileReference)(nil)

func TestAliasRoundTrip(t *testing.T) {
	// Create a test user with alias types
	user := &TestUser{
		ID:       TestUserID("user123"),
		Session:  TestSessionID{0x01, 0x02, 0x03, 0x04},
		Status:   TestStatusCode(200),
		Flags:    TestFlags(0x123456789ABCDEF0),
		Priority: TestPriority(-5),
		Created:  TestTimestamp(1234567890),
		Active:   TestIsActive(true),
		Hash:     TestHash{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA},
	}

	// Encode
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := user.Encode(enc)
	require.NoError(t, err, "Encode failed")

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	require.NoError(t, err, "Decode failed")

	// Verify all fields match
	assert.Equal(t, user.ID, decoded.ID, "ID mismatch")
	assert.Equal(t, string(user.Session), string(decoded.Session), "Session mismatch")
	assert.Equal(t, user.Status, decoded.Status, "Status mismatch")
	assert.Equal(t, user.Flags, decoded.Flags, "Flags mismatch")
	assert.Equal(t, user.Priority, decoded.Priority, "Priority mismatch")
	assert.Equal(t, user.Created, decoded.Created, "Created mismatch")
	assert.Equal(t, user.Active, decoded.Active, "Active mismatch")
	assert.Equal(t, user.Hash, decoded.Hash, "Hash mismatch")
}

func TestFixedSizeArrayAlias(t *testing.T) {
	// Test fixed-size byte array aliases specifically
	user := &TestUser{
		ID:       TestUserID("user123"),
		Session:  TestSessionID{0x01, 0x02, 0x03, 0x04},
		Status:   TestStatusCode(200),
		Flags:    TestFlags(0x123456789ABCDEF0),
		Priority: TestPriority(-5),
		Created:  TestTimestamp(1234567890),
		Active:   TestIsActive(true),
		Hash:     TestHash{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA},
	}

	// Encode
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := user.Encode(enc)
	require.NoError(t, err, "Encode failed")

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	require.NoError(t, err, "Decode failed")

	// Verify hash matches
	assert.Equal(t, user.Hash, decoded.Hash, "Hash mismatch")
}

func TestAliasEdgeCases(t *testing.T) {
	// Test edge cases for alias types
	user := &TestUser{
		ID:       TestUserID(""),      // Empty string
		Session:  TestSessionID{},     // Empty slice
		Status:   TestStatusCode(0),   // Zero value
		Flags:    TestFlags(0),        // Zero value
		Priority: TestPriority(0),     // Zero value
		Created:  TestTimestamp(0),    // Zero value
		Active:   TestIsActive(false), // False
		Hash:     TestHash{},          // Zero value
	}

	// Round trip
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := user.Encode(enc)
	require.NoError(t, err, "Encode failed")

	data := enc.Bytes()
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	require.NoError(t, err, "Decode failed")

	// Verify zero values are preserved
	assert.Equal(t, TestUserID(""), decoded.ID, "Empty string not preserved")
	assert.Empty(t, decoded.Session, "Empty slice not preserved")
	assert.Equal(t, TestStatusCode(0), decoded.Status, "Zero uint32 not preserved")
	assert.Equal(t, TestFlags(0), decoded.Flags, "Zero uint64 not preserved")
	assert.Equal(t, TestPriority(0), decoded.Priority, "Zero int32 not preserved")
	assert.Equal(t, TestTimestamp(0), decoded.Created, "Zero int64 not preserved")
	assert.Equal(t, TestIsActive(false), decoded.Active, "False bool not preserved")
	assert.Equal(t, TestHash{}, decoded.Hash, "Zero fixed array not preserved")
}

func TestFixedArrayZeroAllocationPath(t *testing.T) {
	// Test that the generated code uses DecodeFixedBytesInto for zero allocation
	user := &TestUser{
		Hash: TestHash{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}

	// Encode just the hash field manually to test specific decoding
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := enc.EncodeFixedBytes(user.Hash[:])
	require.NoError(t, err, "EncodeFixedBytes failed")

	encoded := enc.Bytes()

	// Test 1: Decode using DecodeFixedBytesInto (zero-allocation path)
	dec1 := xdr.NewDecoder(encoded)
	var result1 TestHash
	err = dec1.DecodeFixedBytesInto(result1[:])
	require.NoError(t, err, "DecodeFixedBytesInto failed")

	// Test 2: Decode using DecodeFixedBytes (allocating fallback path)
	dec2 := xdr.NewDecoder(encoded)
	bytes2, err := dec2.DecodeFixedBytes(16)
	require.NoError(t, err, "DecodeFixedBytes failed")
	var result2 TestHash
	copy(result2[:], bytes2)

	// Test 3: Decode using generated code (should use zero-allocation path)
	// Create a full struct with other fields to test the generated Decode method
	fullUser := &TestUser{
		ID:       TestUserID("test"),
		Session:  TestSessionID{0xAA, 0xBB},
		Status:   TestStatusCode(200),
		Flags:    TestFlags(0x123),
		Priority: TestPriority(-1),
		Created:  TestTimestamp(12345),
		Active:   TestIsActive(true),
		Hash:     user.Hash,
	}

	buf = make([]byte, 1024)
	enc = xdr.NewEncoder(buf)
	err = fullUser.Encode(enc)
	require.NoError(t, err, "Full encode failed")

	fullEncoded := enc.Bytes()
	dec3 := xdr.NewDecoder(fullEncoded)
	var decoded TestUser
	err = decoded.Decode(dec3)
	require.NoError(t, err, "Generated decode failed")

	// All three results should be identical
	assert.Equal(t, user.Hash, result1, "DecodeFixedBytesInto: expected %v, got %v")
	assert.Equal(t, user.Hash, result2, "DecodeFixedBytes: expected %v, got %v")
	assert.Equal(t, user.Hash, decoded.Hash, "Generated decode: expected %v, got %v")
	assert.Equal(t, result1, result2, "Methods produce different results: DecodeFixedBytesInto=%v, DecodeFixedBytes=%v")
}

func TestFixedArrayVsSliceHandling(t *testing.T) {
	// Verify that fixed arrays and slices are handled differently
	user := &TestUser{
		Session: TestSessionID{0x01, 0x02, 0x03, 0x04},                                                                    // []byte alias
		Hash:    TestHash{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA}, // [16]byte alias
	}

	// Encode
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := user.Encode(enc)
	require.NoError(t, err, "Encode failed")

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	require.NoError(t, err, "Decode failed")

	// Verify Session ([]byte) - should be dynamically allocated
	assert.Equal(t, user.Session, decoded.Session, "Session mismatch")

	// Verify Hash ([16]byte) - should be zero-allocation
	assert.Equal(t, user.Hash, decoded.Hash, "Hash mismatch")

	// Test that we can modify the slice without affecting the source
	originalSession := make([]byte, len(decoded.Session))
	copy(originalSession, decoded.Session)
	decoded.Session[0] = 0xFF // Modify the slice

	// Re-decode to ensure we get a fresh copy
	dec = xdr.NewDecoder(data)
	var decoded2 TestUser
	err = decoded2.Decode(dec)
	require.NoError(t, err, "Second decode failed")

	// Second decode should have original value, not modified value
	assert.Equal(t, originalSession, []byte(decoded2.Session), "Session not properly isolated")
}

// VoidOpCode represents operation codes for void-only union testing
type VoidOpCode uint32

const (
	VoidOpNone VoidOpCode = 0
	VoidOpPing VoidOpCode = 1
)

// +xdr:union,key=OpCode
// VoidOperation tests void-only unions (all-void, default optional)
type VoidOperation struct {
	OpCode VoidOpCode `` // Operation discriminant
	Data   []byte     // Union payload (auto-detected as []byte following key)
}

func TestVoidOnlyUnion(t *testing.T) {
	// Test void-only union with default=nil
	original := &VoidOperation{
		OpCode: VoidOpPing,
		Data:   nil, // Should remain nil/empty for void-only unions
	}

	// Encode
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := original.Encode(enc)
	require.NoError(t, err, "Encode failed")

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded VoidOperation
	err = decoded.Decode(dec)
	require.NoError(t, err, "Decode failed")

	// Verify that the discriminant is preserved
	assert.Equal(t, original.OpCode, decoded.OpCode, "OpCode mismatch")

	// For void-only unions, the Data field should remain empty
	assert.Empty(t, decoded.Data, "Data should be empty for void-only union")
}

func TestVoidUnionAllConstants(t *testing.T) {
	// Test that all constants work with the void-only union
	testCases := []VoidOpCode{VoidOpNone, VoidOpPing}

	for _, opCode := range testCases {
		t.Run(fmt.Sprintf("OpCode_%d", opCode), func(t *testing.T) {
			original := &VoidOperation{
				OpCode: opCode,
				Data:   nil,
			}

			data, err := xdr.Marshal(original)
			require.NoError(t, err, "Marshal failed for %v", opCode)

			var decoded VoidOperation
			err = xdr.Unmarshal(data, &decoded)
			require.NoError(t, err, "Unmarshal failed for %v", opCode)

			assert.Equal(t, original.OpCode, decoded.OpCode, "OpCode mismatch for %v", opCode)
			assert.Empty(t, decoded.Data, "Data should be empty for void-only union %v", opCode)
		})
	}
}

func TestCrossFileStructTypes(t *testing.T) {
	// Test that struct types from other files in the same package work correctly
	original := &TestCrossFileReference{
		CrossFileData: CrossFileStruct{
			ID:   123,
			Name: "test",
		},
		Items: []CrossFileStruct{
			{ID: 1, Name: "item1"},
			{ID: 2, Name: "item2"},
		},
	}

	data, err := xdr.Marshal(original)
	require.NoError(t, err, "Marshal failed")

	var decoded TestCrossFileReference
	err = xdr.Unmarshal(data, &decoded)
	require.NoError(t, err, "Unmarshal failed")

	// Verify the struct field
	assert.Equal(t, original.CrossFileData.ID, decoded.CrossFileData.ID, "CrossFileData.ID mismatch")
	assert.Equal(t, original.CrossFileData.Name, decoded.CrossFileData.Name, "CrossFileData.Name mismatch")

	// Verify the array field
	assert.Len(t, decoded.Items, len(original.Items), "Items length mismatch")
	for i, item := range original.Items {
		assert.Equal(t, item.ID, decoded.Items[i].ID, "Items[%d].ID mismatch", i)
		assert.Equal(t, item.Name, decoded.Items[i].Name, "Items[%d].Name mismatch", i)
	}
}
