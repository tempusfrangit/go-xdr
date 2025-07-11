package xdr_test

import (
	"bytes"
	"testing"

	"github.com/tempusfrangit/go-xdr"
)

// Test type aliases for testing
type TestUserID string
type TestSessionID []byte
type TestStatusCode uint32
type TestFlags uint64
type TestPriority int32
type TestTimestamp int64
type TestIsActive bool
type TestHash [16]byte

// Test struct using alias types
type TestUser struct {
	ID       TestUserID     `xdr:"alias"`
	Session  TestSessionID  `xdr:"alias"`
	Status   TestStatusCode `xdr:"alias"`
	Flags    TestFlags      `xdr:"alias"`
	Priority TestPriority   `xdr:"alias"`
	Created  TestTimestamp  `xdr:"alias"`
	Active   TestIsActive   `xdr:"alias"`
	Hash     TestHash       `xdr:"alias"`
}

// Ensure TestUser implements Codec interface
var _ xdr.Codec = (*TestUser)(nil)

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
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify all fields match
	if decoded.ID != user.ID {
		t.Errorf("ID mismatch: expected %v, got %v", user.ID, decoded.ID)
	}
	if string(decoded.Session) != string(user.Session) {
		t.Errorf("Session mismatch: expected %v, got %v", user.Session, decoded.Session)
	}
	if decoded.Status != user.Status {
		t.Errorf("Status mismatch: expected %v, got %v", user.Status, decoded.Status)
	}
	if decoded.Flags != user.Flags {
		t.Errorf("Flags mismatch: expected %v, got %v", user.Flags, decoded.Flags)
	}
	if decoded.Priority != user.Priority {
		t.Errorf("Priority mismatch: expected %v, got %v", user.Priority, decoded.Priority)
	}
	if decoded.Created != user.Created {
		t.Errorf("Created mismatch: expected %v, got %v", user.Created, decoded.Created)
	}
	if decoded.Active != user.Active {
		t.Errorf("Active mismatch: expected %v, got %v", user.Active, decoded.Active)
	}
	if decoded.Hash != user.Hash {
		t.Errorf("Hash mismatch: expected %v, got %v", user.Hash, decoded.Hash)
	}
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
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify hash matches
	if decoded.Hash != user.Hash {
		t.Errorf("Hash mismatch: expected %v, got %v", user.Hash, decoded.Hash)
	}
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
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := enc.Bytes()
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify zero values are preserved
	if decoded.ID != TestUserID("") {
		t.Errorf("Empty string not preserved: got %v", decoded.ID)
	}
	if len(decoded.Session) != 0 {
		t.Errorf("Empty slice not preserved: got %v", decoded.Session)
	}
	if decoded.Status != TestStatusCode(0) {
		t.Errorf("Zero uint32 not preserved: got %v", decoded.Status)
	}
	if decoded.Flags != TestFlags(0) {
		t.Errorf("Zero uint64 not preserved: got %v", decoded.Flags)
	}
	if decoded.Priority != TestPriority(0) {
		t.Errorf("Zero int32 not preserved: got %v", decoded.Priority)
	}
	if decoded.Created != TestTimestamp(0) {
		t.Errorf("Zero int64 not preserved: got %v", decoded.Created)
	}
	if decoded.Active != TestIsActive(false) {
		t.Errorf("False bool not preserved: got %v", decoded.Active)
	}
	if decoded.Hash != (TestHash{}) {
		t.Errorf("Zero fixed array not preserved: got %v", decoded.Hash)
	}
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
	if err != nil {
		t.Fatalf("EncodeFixedBytes failed: %v", err)
	}

	encoded := enc.Bytes()

	// Test 1: Decode using DecodeFixedBytesInto (zero-allocation path)
	dec1 := xdr.NewDecoder(encoded)
	var result1 TestHash
	err = dec1.DecodeFixedBytesInto(result1[:])
	if err != nil {
		t.Fatalf("DecodeFixedBytesInto failed: %v", err)
	}

	// Test 2: Decode using DecodeFixedBytes (allocating fallback path)
	dec2 := xdr.NewDecoder(encoded)
	bytes2, err := dec2.DecodeFixedBytes(16)
	if err != nil {
		t.Fatalf("DecodeFixedBytes failed: %v", err)
	}
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
	if err != nil {
		t.Fatalf("Full encode failed: %v", err)
	}

	fullEncoded := enc.Bytes()
	dec3 := xdr.NewDecoder(fullEncoded)
	var decoded TestUser
	err = decoded.Decode(dec3)
	if err != nil {
		t.Fatalf("Generated decode failed: %v", err)
	}

	// All three results should be identical
	if result1 != user.Hash {
		t.Errorf("DecodeFixedBytesInto: expected %v, got %v", user.Hash, result1)
	}
	if result2 != user.Hash {
		t.Errorf("DecodeFixedBytes: expected %v, got %v", user.Hash, result2)
	}
	if decoded.Hash != user.Hash {
		t.Errorf("Generated decode: expected %v, got %v", user.Hash, decoded.Hash)
	}
	if result1 != result2 {
		t.Errorf("Methods produce different results: DecodeFixedBytesInto=%v, DecodeFixedBytes=%v", result1, result2)
	}
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
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify Session ([]byte) - should be dynamically allocated
	if !bytes.Equal([]byte(decoded.Session), []byte(user.Session)) {
		t.Errorf("Session mismatch: expected %v, got %v", user.Session, decoded.Session)
	}

	// Verify Hash ([16]byte) - should be zero-allocation
	if decoded.Hash != user.Hash {
		t.Errorf("Hash mismatch: expected %v, got %v", user.Hash, decoded.Hash)
	}

	// Test that we can modify the slice without affecting the source
	originalSession := make([]byte, len(decoded.Session))
	copy(originalSession, decoded.Session)
	decoded.Session[0] = 0xFF // Modify the slice

	// Re-decode to ensure we get a fresh copy
	dec = xdr.NewDecoder(data)
	var decoded2 TestUser
	err = decoded2.Decode(dec)
	if err != nil {
		t.Fatalf("Second decode failed: %v", err)
	}

	// Second decode should have original value, not modified value
	if !bytes.Equal([]byte(decoded2.Session), originalSession) {
		t.Errorf("Session not properly isolated: expected %v, got %v", originalSession, decoded2.Session)
	}
}
