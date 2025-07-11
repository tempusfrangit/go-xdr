package xdr_test

import (
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
